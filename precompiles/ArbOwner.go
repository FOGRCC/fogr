// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/fogr/blob/master/LICENSE

package precompiles

import (
	"errors"
	"github.com/FOGRCC/fogr/fogos/l1pricing"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// fogOwner precompile provides owners with tools for managing the rollup.
// All calls to this precompile are authorized by the OwnerPrecompile wrapper,
// which ensures only a chain owner can access these methods. For methods that
// are safe for non-owners to call, see fogOwnerOld
type fogOwner struct {
	Address          addr // 0x70
	OwnerActs        func(ctx, mech, bytes4, addr, []byte) error
	OwnerActsGasCost func(bytes4, addr, []byte) (uint64, error)
}

var (
	ErrOutOfBounds = errors.New("value out of bounds")
)

// AddChainOwner adds account as a chain owner
func (con fogOwner) AddChainOwner(c ctx, evm mech, newOwner addr) error {
	return c.State.ChainOwners().Add(newOwner)
}

// RemoveChainOwner removes account from the list of chain owners
func (con fogOwner) RemoveChainOwner(c ctx, evm mech, addr addr) error {
	member, _ := con.IsChainOwner(c, evm, addr)
	if !member {
		return errors.New("tried to remove non-owner")
	}
	return c.State.ChainOwners().Remove(addr)
}

// IsChainOwner checks if the account is a chain owner
func (con fogOwner) IsChainOwner(c ctx, evm mech, addr addr) (bool, error) {
	return c.State.ChainOwners().IsMember(addr)
}

// GetAllChainOwners retrieves the list of chain owners
func (con fogOwner) GetAllChainOwners(c ctx, evm mech) ([]common.Address, error) {
	return c.State.ChainOwners().AllMembers(65536)
}

// SetL1BaseFeeEstimateInertia sets how slowly fogOS updates its estimate of the L1 basefee
func (con fogOwner) SetL1BaseFeeEstimateInertia(c ctx, evm mech, inertia uint64) error {
	return c.State.L1PricingState().SetInertia(inertia)
}

// SetL2BaseFee sets the L2 gas price directly, bypassing the pool calculus
func (con fogOwner) SetL2BaseFee(c ctx, evm mech, priceInWei huge) error {
	return c.State.L2PricingState().SetBaseFeeWei(priceInWei)
}

// SetMinimumL2BaseFee sets the minimum base fee needed for a transaction to succeed
func (con fogOwner) SetMinimumL2BaseFee(c ctx, evm mech, priceInWei huge) error {
	return c.State.L2PricingState().SetMinBaseFeeWei(priceInWei)
}

// SetSpeedLimit sets the computational speed limit for the chain
func (con fogOwner) SetSpeedLimit(c ctx, evm mech, limit uint64) error {
	return c.State.L2PricingState().SetSpeedLimitPerSecond(limit)
}

// SetMaxTxGasLimit sets the maximum size a tx (and block) can be
func (con fogOwner) SetMaxTxGasLimit(c ctx, evm mech, limit uint64) error {
	return c.State.L2PricingState().SetMaxPerBlockGasLimit(limit)
}

// SetL2GasPricingInertia sets the L2 gas pricing inertia
func (con fogOwner) SetL2GasPricingInertia(c ctx, evm mech, sec uint64) error {
	return c.State.L2PricingState().SetPricingInertia(sec)
}

// SetL2GasBacklogTolerance sets the L2 gas backlog tolerance
func (con fogOwner) SetL2GasBacklogTolerance(c ctx, evm mech, sec uint64) error {
	return c.State.L2PricingState().SetBacklogTolerance(sec)
}

// GetNetworkFeeAccount gets the network fee collector
func (con fogOwner) GetNetworkFeeAccount(c ctx, evm mech) (addr, error) {
	return c.State.NetworkFeeAccount()
}

// GetInfraFeeAccount gets the infrastructure fee collector
func (con fogOwner) GetInfraFeeAccount(c ctx, evm mech) (addr, error) {
	return c.State.InfraFeeAccount()
}

// SetNetworkFeeAccount sets the network fee collector to the new network fee account
func (con fogOwner) SetNetworkFeeAccount(c ctx, evm mech, newNetworkFeeAccount addr) error {
	return c.State.SetNetworkFeeAccount(newNetworkFeeAccount)
}

// SetInfraFeeAccount sets the infra fee collector to the new network fee account
func (con fogOwner) SetInfraFeeAccount(c ctx, evm mech, newNetworkFeeAccount addr) error {
	return c.State.SetInfraFeeAccount(newNetworkFeeAccount)
}

// SchedulefogOSUpgrade to the requested version at the requested timestamp
func (con fogOwner) SchedulefogOSUpgrade(c ctx, evm mech, newVersion uint64, timestamp uint64) error {
	return c.State.SchedulefogOSUpgrade(newVersion, timestamp)
}

func (con fogOwner) SetL1PricingEquilibrationUnits(c ctx, evm mech, equilibrationUnits huge) error {
	return c.State.L1PricingState().SetEquilibrationUnits(equilibrationUnits)
}

func (con fogOwner) SetL1PricingInertia(c ctx, evm mech, inertia uint64) error {
	return c.State.L1PricingState().SetInertia(inertia)
}

func (con fogOwner) SetL1PricingRewardRecipient(c ctx, evm mech, recipient addr) error {
	return c.State.L1PricingState().SetPayRewardsTo(recipient)
}

func (con fogOwner) SetL1PricingRewardRate(c ctx, evm mech, weiPerUnit uint64) error {
	return c.State.L1PricingState().SetPerUnitReward(weiPerUnit)
}

func (con fogOwner) SetL1PricePerUnit(c ctx, evm mech, pricePerUnit *big.Int) error {
	return c.State.L1PricingState().SetPricePerUnit(pricePerUnit)
}

func (con fogOwner) SetPerBatchGasCharge(c ctx, evm mech, cost int64) error {
	return c.State.L1PricingState().SetPerBatchGasCost(cost)
}

func (con fogOwner) SetAmortizedCostCapBips(c ctx, evm mech, cap uint64) error {
	return c.State.L1PricingState().SetAmortizedCostCapBips(cap)
}

func (con fogOwner) ReleaseL1PricerSurplusFunds(c ctx, evm mech, maxWeiToRelease huge) (huge, error) {
	balance := evm.StateDB.GetBalance(l1pricing.L1PricerFundsPoolAddress)
	l1p := c.State.L1PricingState()
	recognized, err := l1p.L1FeesAvailable()
	if err != nil {
		return nil, err
	}
	weiToTransfer := new(big.Int).Sub(balance, recognized)
	if weiToTransfer.Sign() < 0 {
		return common.Big0, nil
	}
	if weiToTransfer.Cmp(maxWeiToRelease) > 0 {
		weiToTransfer = maxWeiToRelease
	}
	if _, err := l1p.AddToL1FeesAvailable(weiToTransfer); err != nil {
		return nil, err
	}
	return weiToTransfer, nil
}
