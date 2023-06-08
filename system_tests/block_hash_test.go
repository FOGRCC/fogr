// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/fogr/blob/master/LICENSE

package fogtest

import (
	"context"
	"testing"

	"github.com/FOGRCC/fogr/solgen/go/mocksgen"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

func TestBlockHash(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Even though we don't use the L1, we need to create this node on L1 to get accurate L1 block numbers
	l2info, l2node, l2client, _, _, _, l1stack := createTestNodeOnL1(t, ctx, true)
	defer requireClose(t, l1stack)
	defer l2node.StopAndWait()

	auth := l2info.GetDefaultTransactOpts("Faucet", ctx)

	_, _, simple, err := mocksgen.DeploySimple(&auth, l2client)
	Require(t, err)

	_, err = simple.CheckBlockHashes(&bind.CallOpts{Context: ctx})
	Require(t, err)
}
