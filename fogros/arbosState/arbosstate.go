// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/fogr/blob/master/LICENSE

package fogosState

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/FOGRCC/fogr/fogos/addressSet"
	"github.com/FOGRCC/fogr/fogos/addressTable"
	"github.com/FOGRCC/fogr/fogos/blockhash"
	"github.com/FOGRCC/fogr/fogos/burn"
	"github.com/FOGRCC/fogr/fogos/l1pricing"
	"github.com/FOGRCC/fogr/fogos/l2pricing"
	"github.com/FOGRCC/fogr/fogos/merkleAccumulator"
	"github.com/FOGRCC/fogr/fogos/retryables"
	"github.com/FOGRCC/fogr/fogos/storage"
	"github.com/FOGRCC/fogr/fogos/util"
)

// fogosState contains fogOS-related state. It is backed by fogOS's storage in the persistent stateDB.
// Modifications to the fogosState are written through to the underlying StateDB so that the StateDB always
// has the definitive state, stored persistently. (Note that some tests use memory-backed StateDB's that aren't
// persisted beyond the end of the test.)

type fogosState struct {
	fogosVersion      uint64                      // version of the fogOS storage format and semantics
	upgradeVersion    storage.StorageBackedUint64 // version we're planning to upgrade to, or 0 if not planning to upgrade
	upgradeTimestamp  storage.StorageBackedUint64 // when to do the planned upgrade
	networkFeeAccount storage.StorageBackedAddress
	l1PricingState    *l1pricing.L1PricingState
	l2PricingState    *l2pricing.L2PricingState
	retryableState    *retryables.RetryableState
	addressTable      *addressTable.AddressTable
	chainOwners       *addressSet.AddressSet
	sendMerkle        *merkleAccumulator.MerkleAccumulator
	blockhashes       *blockhash.Blockhashes
	chainId           storage.StorageBackedBigInt
	genesisBlockNum   storage.StorageBackedUint64
	infraFeeAccount   storage.StorageBackedAddress
	backingStorage    *storage.Storage
	Burner            burn.Burner
}

var ErrUninitializedfogOS = errors.New("fogOS uninitialized")
var ErrAlreadyInitialized = errors.New("fogOS is already initialized")

func OpenfogosState(stateDB vm.StateDB, burner burn.Burner) (*fogosState, error) {
	backingStorage := storage.NewGeth(stateDB, burner)
	fogosVersion, err := backingStorage.GetUint64ByUint64(uint64(versionOffset))
	if err != nil {
		return nil, err
	}
	if fogosVersion == 0 {
		return nil, ErrUninitializedfogOS
	}
	return &fogosState{
		fogosVersion,
		backingStorage.OpenStorageBackedUint64(uint64(upgradeVersionOffset)),
		backingStorage.OpenStorageBackedUint64(uint64(upgradeTimestampOffset)),
		backingStorage.OpenStorageBackedAddress(uint64(networkFeeAccountOffset)),
		l1pricing.OpenL1PricingState(backingStorage.OpenSubStorage(l1PricingSubspace)),
		l2pricing.OpenL2PricingState(backingStorage.OpenSubStorage(l2PricingSubspace)),
		retryables.OpenRetryableState(backingStorage.OpenSubStorage(retryablesSubspace), stateDB),
		addressTable.Open(backingStorage.OpenSubStorage(addressTableSubspace)),
		addressSet.OpenAddressSet(backingStorage.OpenSubStorage(chainOwnerSubspace)),
		merkleAccumulator.OpenMerkleAccumulator(backingStorage.OpenSubStorage(sendMerkleSubspace)),
		blockhash.OpenBlockhashes(backingStorage.OpenSubStorage(blockhashesSubspace)),
		backingStorage.OpenStorageBackedBigInt(uint64(chainIdOffset)),
		backingStorage.OpenStorageBackedUint64(uint64(genesisBlockNumOffset)),
		backingStorage.OpenStorageBackedAddress(uint64(infraFeeAccountOffset)),
		backingStorage,
		burner,
	}, nil
}

func OpenSystemfogosState(stateDB vm.StateDB, tracingInfo *util.TracingInfo, readOnly bool) (*fogosState, error) {
	burner := burn.NewSystemBurner(tracingInfo, readOnly)
	newState, err := OpenfogosState(stateDB, burner)
	burner.Restrict(err)
	return newState, err
}

func OpenSystemfogosStateOrPanic(stateDB vm.StateDB, tracingInfo *util.TracingInfo, readOnly bool) *fogosState {
	newState, err := OpenSystemfogosState(stateDB, tracingInfo, readOnly)
	if err != nil {
		panic(err)
	}
	return newState
}

// NewfogosMemoryBackedfogOSState creates and initializes a memory-backed fogOS state (for testing only)
func NewfogosMemoryBackedfogOSState() (*fogosState, *state.StateDB) {
	raw := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(raw)
	statedb, err := state.New(common.Hash{}, db, nil)
	if err != nil {
		log.Crit("failed to init empty statedb", "error", err)
	}
	burner := burn.NewSystemBurner(nil, false)
	newState, err := InitializefogosState(statedb, burner, params.FOGDevTestChainConfig())
	if err != nil {
		log.Crit("failed to open the fogOS state", "error", err)
	}
	return newState, statedb
}

// fogOSVersion returns the fogOS version
func fogOSVersion(stateDB vm.StateDB) uint64 {
	backingStorage := storage.NewGeth(stateDB, burn.NewSystemBurner(nil, false))
	fogosVersion, err := backingStorage.GetUint64ByUint64(uint64(versionOffset))
	if err != nil {
		log.Crit("failed to get the fogOS version", "error", err)
	}
	return fogosVersion
}

type Offset uint64

const (
	versionOffset Offset = iota
	upgradeVersionOffset
	upgradeTimestampOffset
	networkFeeAccountOffset
	chainIdOffset
	genesisBlockNumOffset
	infraFeeAccountOffset
)

type SubspaceID []byte

var (
	l1PricingSubspace    SubspaceID = []byte{0}
	l2PricingSubspace    SubspaceID = []byte{1}
	retryablesSubspace   SubspaceID = []byte{2}
	addressTableSubspace SubspaceID = []byte{3}
	chainOwnerSubspace   SubspaceID = []byte{4}
	sendMerkleSubspace   SubspaceID = []byte{5}
	blockhashesSubspace  SubspaceID = []byte{6}
)

// Returns a list of precompiles that only appear in FOGR chains (i.e. fogOS precompiles) at the genesis block
func getFOGOnlyPrecompiles(chainConfig *params.ChainConfig) []common.Address {
	rules := chainConfig.Rules(big.NewInt(0), false)
	fogPrecompiles := vm.ActivePrecompiles(rules)
	rules.IsFOG = false
	ethPrecompiles := vm.ActivePrecompiles(rules)

	ethPrecompilesSet := make(map[common.Address]bool)
	for _, addr := range ethPrecompiles {
		ethPrecompilesSet[addr] = true
	}

	var fogOnlyPrecompiles []common.Address
	for _, addr := range fogPrecompiles {
		if !ethPrecompilesSet[addr] {
			fogOnlyPrecompiles = append(fogOnlyPrecompiles, addr)
		}
	}
	return fogOnlyPrecompiles
}

// During early development we sometimes change the storage format of version 1, for convenience. But as soon as we
// start running long-lived chains, every change to the storage format will require defining a new version and
// providing upgrade code.

func InitializefogosState(stateDB vm.StateDB, burner burn.Burner, chainConfig *params.ChainConfig) (*fogosState, error) {
	sto := storage.NewGeth(stateDB, burner)
	fogosVersion, err := sto.GetUint64ByUint64(uint64(versionOffset))
	if err != nil {
		return nil, err
	}
	if fogosVersion != 0 {
		return nil, ErrAlreadyInitialized
	}

	desiredfogosVersion := chainConfig.FOGChainParams.InitialfogOSVersion
	if desiredfogosVersion == 0 {
		return nil, errors.New("cannot initialize to fogOS version 0")
	}

	// Solidity requires call targets have code, but precompiles don't.
	// To work around this, we give precompiles fake code.
	for _, precompile := range getFOGOnlyPrecompiles(chainConfig) {
		stateDB.SetCode(precompile, []byte{byte(vm.INVALID)})
	}

	// may be the zero address
	initialChainOwner := chainConfig.FOGChainParams.InitialChainOwner

	_ = sto.SetUint64ByUint64(uint64(versionOffset), 1) // initialize to version 1; upgrade at end of this func if needed
	_ = sto.SetUint64ByUint64(uint64(upgradeVersionOffset), 0)
	_ = sto.SetUint64ByUint64(uint64(upgradeTimestampOffset), 0)
	if desiredfogosVersion >= 2 {
		_ = sto.SetByUint64(uint64(networkFeeAccountOffset), util.AddressToHash(initialChainOwner))
	} else {
		_ = sto.SetByUint64(uint64(networkFeeAccountOffset), common.Hash{}) // the 0 address until an owner sets it
	}
	_ = sto.SetByUint64(uint64(chainIdOffset), common.BigToHash(chainConfig.ChainID))
	_ = sto.SetUint64ByUint64(uint64(genesisBlockNumOffset), chainConfig.FOGChainParams.GenesisBlockNum)

	initialRewardsRecipient := l1pricing.BatchPosterAddress
	if desiredfogosVersion >= 2 {
		initialRewardsRecipient = initialChainOwner
	}
	_ = l1pricing.InitializeL1PricingState(sto.OpenSubStorage(l1PricingSubspace), initialRewardsRecipient)
	_ = l2pricing.InitializeL2PricingState(sto.OpenSubStorage(l2PricingSubspace))
	_ = retryables.InitializeRetryableState(sto.OpenSubStorage(retryablesSubspace))
	addressTable.Initialize(sto.OpenSubStorage(addressTableSubspace))
	merkleAccumulator.InitializeMerkleAccumulator(sto.OpenSubStorage(sendMerkleSubspace))
	blockhash.InitializeBlockhashes(sto.OpenSubStorage(blockhashesSubspace))

	ownersStorage := sto.OpenSubStorage(chainOwnerSubspace)
	_ = addressSet.Initialize(ownersStorage)
	_ = addressSet.OpenAddressSet(ownersStorage).Add(initialChainOwner)

	aState, err := OpenfogosState(stateDB, burner)
	if err != nil {
		return nil, err
	}
	if desiredfogosVersion > 1 {
		err = aState.UpgradefogosVersion(desiredfogosVersion, true, stateDB, chainConfig)
		if err != nil {
			return nil, err
		}
	}
	return aState, nil
}

func (state *fogosState) UpgradefogosVersionIfNecessary(
	currentTimestamp uint64, stateDB vm.StateDB, chainConfig *params.ChainConfig,
) error {
	upgradeTo, err := state.upgradeVersion.Get()
	state.Restrict(err)
	flagday, _ := state.upgradeTimestamp.Get()
	if state.fogosVersion < upgradeTo && currentTimestamp >= flagday {
		return state.UpgradefogosVersion(upgradeTo, false, stateDB, chainConfig)
	}
	return nil
}

var ErrFatalNodeOutOfDate = errors.New("please upgrade to the latest version of the node software")

func (state *fogosState) UpgradefogosVersion(
	upgradeTo uint64, firstTime bool, stateDB vm.StateDB, chainConfig *params.ChainConfig,
) error {
	for state.fogosVersion < upgradeTo {
		ensure := func(err error) {
			if err != nil {
				message := fmt.Sprintf(
					"Failed to upgrade fogOS version %v to version %v: %v",
					state.fogosVersion, state.fogosVersion+1, err,
				)
				panic(message)
			}
		}

		switch state.fogosVersion {
		case 1:
			ensure(state.l1PricingState.SetLastSurplus(common.Big0, 1))
		case 2:
			ensure(state.l1PricingState.SetPerBatchGasCost(0))
			ensure(state.l1PricingState.SetAmortizedCostCapBips(math.MaxUint64))
		case 3:
			// no state changes needed
		case 4:
			// no state changes needed
		case 5:
			// no state changes needed
		case 6:
			// no state changes needed
		case 7:
			// no state changes needed
		case 8:
			// no state changes needed
		case 9:
			ensure(state.l1PricingState.SetL1FeesAvailable(stateDB.GetBalance(
				l1pricing.L1PricerFundsPoolAddress,
			)))
		case 10:
			if !chainConfig.DebugMode() {
				// This upgrade isn't finalized so we only want to support it for testing
				return fmt.Errorf(
					"the chain is upgrading to unsupported fogOS version %v, %w",
					state.fogosVersion+1,
					ErrFatalNodeOutOfDate,
				)
			}
			// no state changes needed
		default:
			return fmt.Errorf(
				"the chain is upgrading to unsupported fogOS version %v, %w",
				state.fogosVersion+1,
				ErrFatalNodeOutOfDate,
			)
		}
		state.fogosVersion++
	}

	if firstTime && upgradeTo >= 6 {
		state.Restrict(state.l1PricingState.SetPerBatchGasCost(l1pricing.InitialPerBatchGasCostV6))
		state.Restrict(state.l1PricingState.SetEquilibrationUnits(l1pricing.InitialEquilibrationUnitsV6))
		state.Restrict(state.l2PricingState.SetSpeedLimitPerSecond(l2pricing.InitialSpeedLimitPerSecondV6))
		state.Restrict(state.l2PricingState.SetMaxPerBlockGasLimit(l2pricing.InitialPerBlockGasLimitV6))
	}

	state.Restrict(state.backingStorage.SetUint64ByUint64(uint64(versionOffset), state.fogosVersion))

	return nil
}

func (state *fogosState) SchedulefogOSUpgrade(newVersion uint64, timestamp uint64) error {
	err := state.upgradeVersion.Set(newVersion)
	if err != nil {
		return err
	}
	return state.upgradeTimestamp.Set(timestamp)
}

func (state *fogosState) GetScheduledUpgrade() (uint64, uint64, error) {
	version, err := state.upgradeVersion.Get()
	if err != nil {
		return 0, 0, err
	}
	timestamp, err := state.upgradeTimestamp.Get()
	if err != nil {
		return 0, 0, err
	}
	return version, timestamp, nil
}

func (state *fogosState) BackingStorage() *storage.Storage {
	return state.backingStorage
}

func (state *fogosState) Restrict(err error) {
	state.Burner.Restrict(err)
}

func (state *fogosState) fogOSVersion() uint64 {
	return state.fogosVersion
}

func (state *fogosState) SetFormatVersion(val uint64) {
	state.fogosVersion = val
	state.Restrict(state.backingStorage.SetUint64ByUint64(uint64(versionOffset), val))
}

func (state *fogosState) RetryableState() *retryables.RetryableState {
	return state.retryableState
}

func (state *fogosState) L1PricingState() *l1pricing.L1PricingState {
	return state.l1PricingState
}

func (state *fogosState) L2PricingState() *l2pricing.L2PricingState {
	return state.l2PricingState
}

func (state *fogosState) AddressTable() *addressTable.AddressTable {
	return state.addressTable
}

func (state *fogosState) ChainOwners() *addressSet.AddressSet {
	return state.chainOwners
}

func (state *fogosState) SendMerkleAccumulator() *merkleAccumulator.MerkleAccumulator {
	if state.sendMerkle == nil {
		state.sendMerkle = merkleAccumulator.OpenMerkleAccumulator(state.backingStorage.OpenSubStorage(sendMerkleSubspace))
	}
	return state.sendMerkle
}

func (state *fogosState) Blockhashes() *blockhash.Blockhashes {
	return state.blockhashes
}

func (state *fogosState) NetworkFeeAccount() (common.Address, error) {
	return state.networkFeeAccount.Get()
}

func (state *fogosState) SetNetworkFeeAccount(account common.Address) error {
	return state.networkFeeAccount.Set(account)
}

func (state *fogosState) InfraFeeAccount() (common.Address, error) {
	return state.infraFeeAccount.Get()
}

func (state *fogosState) SetInfraFeeAccount(account common.Address) error {
	return state.infraFeeAccount.Set(account)
}

func (state *fogosState) Keccak(data ...[]byte) ([]byte, error) {
	return state.backingStorage.Keccak(data...)
}

func (state *fogosState) KeccakHash(data ...[]byte) (common.Hash, error) {
	return state.backingStorage.KeccakHash(data...)
}

func (state *fogosState) ChainId() (*big.Int, error) {
	return state.chainId.Get()
}

func (state *fogosState) GenesisBlockNum() (uint64, error) {
	return state.genesisBlockNum.Get()
}
