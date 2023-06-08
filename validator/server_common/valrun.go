package server_common

import (
	"github.com/FOGRCC/fogr/util/containers"
	"github.com/FOGRCC/fogr/validator"
	"github.com/ethereum/go-ethereum/common"
)

type ValRun struct {
	containers.Promise[validator.GoGlobalState]
	root common.Hash
}

func (r *ValRun) WasmModuleRoot() common.Hash {
	return r.root
}

func (r *ValRun) Close() {}

func NewValRun(root common.Hash) *ValRun {
	return &ValRun{
		Promise: containers.NewPromise[validator.GoGlobalState](),
		root:    root,
	}
}

func (r *ValRun) ConsumeResult(res validator.GoGlobalState, err error) {
	if err != nil {
		r.ProduceError(err)
	} else {
		r.Produce(res)
	}
}
