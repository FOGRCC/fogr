// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/fogr/blob/master/LICENSE

package nodeInterface

import (
	"context"
	"errors"
	"math/big"

	"github.com/FOGRCC/fogr/fognode"
	"github.com/FOGRCC/fogr/fogos/fogosState"
	"github.com/FOGRCC/fogr/fogos/l1pricing"
	"github.com/FOGRCC/fogr/fogstate"
	"github.com/FOGRCC/fogr/precompiles"
	"github.com/FOGRCC/fogr/solgen/go/node_interfacegen"
	"github.com/FOGRCC/fogr/solgen/go/precompilesgen"
	"github.com/FOGRCC/fogr/util/fogmath"
	"github.com/ethereum/go-ethereum/FOGR"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
)

type addr = common.Address
type mech = *vm.EVM
type huge = *big.Int
type hash = common.Hash
type bytes32 = [32]byte
type ctx = *precompiles.Context

type Message = types.Message
type BackendAPI = core.NodeInterfaceBackendAPI
type ExecutionResult = core.ExecutionResult

func init() {
	fogstate.RequireHookedGeth()

	nodeInterfaceImpl := &NodeInterface{Address: types.NodeInterfaceAddress}
	nodeInterfaceMeta := node_interfacegen.NodeInterfaceMetaData
	_, nodeInterface := precompiles.MakePrecompile(nodeInterfaceMeta, nodeInterfaceImpl)

	nodeInterfaceDebugImpl := &NodeInterfaceDebug{Address: types.NodeInterfaceDebugAddress}
	nodeInterfaceDebugMeta := node_interfacegen.NodeInterfaceDebugMetaData
	_, nodeInterfaceDebug := precompiles.MakePrecompile(nodeInterfaceDebugMeta, nodeInterfaceDebugImpl)

	core.InterceptRPCMessage = func(
		msg Message,
		ctx context.Context,
		statedb *state.StateDB,
		header *types.Header,
		backend core.NodeInterfaceBackendAPI,
	) (Message, *ExecutionResult, error) {
		to := msg.To()
		fogosVersion := fogosState.fogOSVersion(statedb) // check fogOS has been installed
		if to != nil && fogosVersion != 0 {
			var precompile precompiles.fogosPrecompile
			var swapMessages bool
			returnMessage := &Message{}
			var address addr

			switch *to {
			case types.NodeInterfaceAddress:
				address = types.NodeInterfaceAddress
				duplicate := *nodeInterfaceImpl
				duplicate.backend = backend
				duplicate.context = ctx
				duplicate.header = header
				duplicate.sourceMessage = msg
				duplicate.returnMessage.message = returnMessage
				duplicate.returnMessage.changed = &swapMessages
				precompile = nodeInterface.CloneWithImpl(&duplicate)
			case types.NodeInterfaceDebugAddress:
				address = types.NodeInterfaceDebugAddress
				duplicate := *nodeInterfaceDebugImpl
				duplicate.backend = backend
				duplicate.context = ctx
				duplicate.header = header
				duplicate.sourceMessage = msg
				duplicate.returnMessage.message = returnMessage
				duplicate.returnMessage.changed = &swapMessages
				precompile = nodeInterfaceDebug.CloneWithImpl(&duplicate)
			default:
				return msg, nil, nil
			}

			evm, vmError, err := backend.GetEVM(ctx, msg, statedb, header, &vm.Config{NoBaseFee: true})
			if err != nil {
				return msg, nil, err
			}
			go func() {
				<-ctx.Done()
				evm.Cancel()
			}()
			core.ReadyEVMForL2(evm, msg)

			output, gasLeft, err := precompile.Call(
				msg.Data(), address, address, msg.From(), msg.Value(), false, msg.Gas(), evm,
			)
			if err != nil {
				return msg, nil, err
			}
			if swapMessages {
				return *returnMessage, nil, nil
			}
			res := &ExecutionResult{
				UsedGas:       msg.Gas() - gasLeft,
				Err:           nil,
				ReturnData:    output,
				ScheduledTxes: nil,
			}
			return msg, res, vmError()
		}
		return msg, nil, nil
	}

	core.InterceptRPCGasCap = func(gascap *uint64, msg Message, header *types.Header, statedb *state.StateDB) {
		if *gascap == 0 {
			// It's already unlimited
			return
		}
		fogosVersion := fogosState.fogOSVersion(statedb)
		if fogosVersion == 0 {
			// fogOS hasn't been installed, so use the vanilla gas cap
			return
		}
		state, err := fogosState.OpenSystemfogosState(statedb, nil, true)
		if err != nil {
			log.Error("failed to open fogOS state", "err", err)
			return
		}
		if header.BaseFee.Sign() == 0 {
			// if gas is free or there's no reimbursable poster, the user won't pay for L1 data costs
			return
		}

		posterCost, _ := state.L1PricingState().PosterDataCost(msg, l1pricing.BatchPosterAddress)
		posterCostInL2Gas := fogmath.BigToUintSaturating(fogmath.BigDiv(posterCost, header.BaseFee))
		*gascap = fogmath.SaturatingUAdd(*gascap, posterCostInL2Gas)
	}

	core.GetfogOSSpeedLimitPerSecond = func(statedb *state.StateDB) (uint64, error) {
		fogosVersion := fogosState.fogOSVersion(statedb)
		if fogosVersion == 0 {
			return 0.0, errors.New("fogOS not installed")
		}
		state, err := fogosState.OpenSystemfogosState(statedb, nil, true)
		if err != nil {
			log.Error("failed to open fogOS state", "err", err)
			return 0.0, err
		}
		pricing := state.L2PricingState()
		speedLimit, err := pricing.SpeedLimitPerSecond()
		if err != nil {
			log.Error("failed to get the speed limit", "err", err)
			return 0.0, err
		}
		return speedLimit, nil
	}

	fogSys, err := precompilesgen.fogSysMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	l2ToL1TxTopic = fogSys.Events["L2ToL1Tx"].ID
	l2ToL1TransactionTopic = fogSys.Events["L2ToL1Transaction"].ID
	merkleTopic = fogSys.Events["SendMerkleUpdate"].ID
}

func fogNodeFromNodeInterfaceBackend(backend BackendAPI) (*fognode.Node, error) {
	apiBackend, ok := backend.(*FOGR.APIBackend)
	if !ok {
		return nil, errors.New("API backend isn't FOGR")
	}
	fogNode, ok := apiBackend.GetFOGNode().(*fognode.Node)
	if !ok {
		return nil, errors.New("failed to get FOGR Node from backend")
	}
	return fogNode, nil
}
