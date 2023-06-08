// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/fogr/blob/master/LICENSE

package fogtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/FOGRCC/fogr/fogos"
	"github.com/FOGRCC/fogr/solgen/go/mocksgen"
	"github.com/FOGRCC/fogr/solgen/go/precompilesgen"
	"github.com/FOGRCC/fogr/util/fogmath"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

func TestPurePrecompileMethodCalls(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, node, client := CreateTestL2(t, ctx)
	defer node.StopAndWait()

	fogSys, err := precompilesgen.NewfogSys(common.HexToAddress("0x64"), client)
	Require(t, err, "could not deploy fogSys contract")
	chainId, err := fogSys.fogChainID(&bind.CallOpts{})
	Require(t, err, "failed to get the ChainID")
	if chainId.Uint64() != params.FOGDevTestChainConfig().ChainID.Uint64() {
		Fail(t, "Wrong ChainID", chainId.Uint64())
	}
}

func TestCustomSolidityErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, node, client := CreateTestL2(t, ctx)
	defer node.StopAndWait()

	callOpts := &bind.CallOpts{Context: ctx}
	fogDebug, err := precompilesgen.NewfogDebug(common.HexToAddress("0xff"), client)
	Require(t, err, "could not bind fogDebug contract")
	customError := fogDebug.CustomRevert(callOpts, 1024)
	if customError == nil {
		Fail(t, "customRevert call should have errored")
	}
	observedMessage := customError.Error()
	expectedMessage := "execution reverted: error Custom(1024, This spider family wards off bugs: /\\oo/\\ //\\(oo)/\\ /\\oo/\\, true)"
	if observedMessage != expectedMessage {
		Fail(t, observedMessage)
	}

	fogSys, err := precompilesgen.NewfogSys(fogos.fogSysAddress, client)
	Require(t, err, "could not bind fogSys contract")
	_, customError = fogSys.fogBlockHash(callOpts, big.NewInt(1e9))
	if customError == nil {
		Fail(t, "out of range fogBlockHash call should have errored")
	}
	observedMessage = customError.Error()
	expectedMessage = "execution reverted: error InvalidBlockNumber(1000000000, 1)"
	if observedMessage != expectedMessage {
		Fail(t, observedMessage)
	}
}

func TestPrecompileErrorGasLeft(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	info, node, client := CreateTestL2(t, ctx)
	defer node.StopAndWait()

	auth := info.GetDefaultTransactOpts("Faucet", ctx)
	_, _, simple, err := mocksgen.DeploySimple(&auth, client)
	Require(t, err)

	assertNotAllGasConsumed := func(to common.Address, input []byte) {
		gas, err := simple.CheckGasUsed(&bind.CallOpts{Context: ctx}, to, input)
		Require(t, err, "Failed to call CheckGasUsed to precompile", to)
		maxGas := big.NewInt(100_000)
		if fogmath.BigGreaterThan(gas, maxGas) {
			Fail(t, "Precompile", to, "used", gas, "gas reverting, greater than max expected", maxGas)
		}
	}

	fogSys, err := precompilesgen.fogSysMetaData.GetAbi()
	Require(t, err)

	fogBlockHash := fogSys.Methods["fogBlockHash"]
	data, err := fogBlockHash.Inputs.Pack(big.NewInt(1e9))
	Require(t, err)
	input := append([]byte{}, fogBlockHash.ID...)
	input = append(input, data...)
	assertNotAllGasConsumed(fogos.fogSysAddress, input)

	fogDebug, err := precompilesgen.fogDebugMetaData.GetAbi()
	Require(t, err)
	assertNotAllGasConsumed(common.HexToAddress("0xff"), fogDebug.Methods["legacyError"].ID)
}
