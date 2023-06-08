// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/fogr/blob/master/LICENSE

package fognode

import (
	"context"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

type TransactionPublisher interface {
	PublishTransaction(ctx context.Context, tx *types.Transaction) error
	CheckHealth(ctx context.Context) error
	Initialize(context.Context) error
	Start(context.Context) error
	StopAndWait()
	Started() bool
}

type fogInterface struct {
	txStreamer  *TransactionStreamer
	txPublisher TransactionPublisher
	fogNode     *Node
}

func NewfogInterface(txStreamer *TransactionStreamer, txPublisher TransactionPublisher) (*fogInterface, error) {
	return &fogInterface{
		txStreamer:  txStreamer,
		txPublisher: txPublisher,
	}, nil
}

func (a *fogInterface) Initialize(n *Node) {
	a.fogNode = n
}

func (a *fogInterface) PublishTransaction(ctx context.Context, tx *types.Transaction) error {
	return a.txPublisher.PublishTransaction(ctx, tx)
}

func (a *fogInterface) TransactionStreamer() *TransactionStreamer {
	return a.txStreamer
}

func (a *fogInterface) BlockChain() *core.BlockChain {
	return a.txStreamer.bc
}

func (a *fogInterface) fogNode() interface{} {
	return a.fogNode
}
