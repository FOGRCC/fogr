// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/fogr/blob/master/LICENSE

package fogtest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/FOGRCC/fogr/cmd/genericconf"
	"github.com/ethereum/go-ethereum/ethclient"
)

func TestIpcRpc(t *testing.T) {
	ipcPath := filepath.Join(t.TempDir(), "test.ipc")

	ipcConfig := genericconf.IPCConfigDefault
	ipcConfig.Path = ipcPath

	stackConf := getTestStackConfig(t)
	ipcConfig.Apply(stackConf)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, l2node, _, _, _, _, l1stack := createTestNodeOnL1WithConfig(t, ctx, true, nil, nil, stackConf)
	defer requireClose(t, l1stack)
	defer l2node.StopAndWait()

	_, err := ethclient.Dial(ipcPath)
	Require(t, err)
}
