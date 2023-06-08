// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/fogr/blob/master/LICENSE

// race detection makes things slow and miss timeouts
//go:build !race
// +build !race

package fogtest

import (
	"context"
	"testing"

	"github.com/FOGRCC/fogr/fognode"
	"github.com/FOGRCC/fogr/fogos/l2pricing"
	"github.com/FOGRCC/fogr/solgen/go/precompilesgen"
	"github.com/FOGRCC/fogr/util/fogmath"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestInfraFee(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nodeconfig := fognode.ConfigDefaultL2Test()

	l2info, node, client := CreateTestL2WithConfig(t, ctx, nil, nodeconfig, true)
	defer node.StopAndWait()

	l2info.GenerateAccount("User2")

	ownerTxOpts := l2info.GetDefaultTransactOpts("Owner", ctx)
	ownerTxOpts.Context = ctx
	ownerCallOpts := l2info.GetDefaultCallOpts("Owner", ctx)

	fogowner, err := precompilesgen.NewfogOwner(common.HexToAddress("70"), client)
	Require(t, err)
	fogownerPublic, err := precompilesgen.NewfogOwnerPublic(common.HexToAddress("6b"), client)
	Require(t, err)
	networkFeeAddr, err := fogownerPublic.GetNetworkFeeAccount(ownerCallOpts)
	Require(t, err)
	infraFeeAddr := common.BytesToAddress(crypto.Keccak256([]byte{3, 2, 6}))
	tx, err := fogowner.SetInfraFeeAccount(&ownerTxOpts, infraFeeAddr)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	_, simple := deploySimple(t, ctx, ownerTxOpts, client)

	netFeeBalanceBefore, err := client.BalanceAt(ctx, networkFeeAddr, nil)
	Require(t, err)
	infraFeeBalanceBefore, err := client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)

	tx, err = simple.Increment(&ownerTxOpts)
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
	l2GasUsed := receipt.GasUsed - receipt.GasUsedForL1
	expectedFunds := fogmath.BigMulByUint(fogmath.UintToBig(l2pricing.InitialBaseFeeWei), l2GasUsed)
	expectedBalanceAfter := fogmath.BigAdd(infraFeeBalanceBefore, expectedFunds)

	netFeeBalanceAfter, err := client.BalanceAt(ctx, networkFeeAddr, nil)
	Require(t, err)
	infraFeeBalanceAfter, err := client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)

	if !fogmath.BigEquals(netFeeBalanceBefore, netFeeBalanceAfter) {
		Fail(t, netFeeBalanceBefore, netFeeBalanceAfter)
	}
	if !fogmath.BigEquals(infraFeeBalanceAfter, expectedBalanceAfter) {
		Fail(t, infraFeeBalanceBefore, expectedFunds, infraFeeBalanceAfter, expectedBalanceAfter)
	}
}
