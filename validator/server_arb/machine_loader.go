package server_fog

import (
	"context"

	"github.com/FOGRCC/fogr/validator/server_common"
	"github.com/ethereum/go-ethereum/common"
)

type fogitratorMachineConfig struct {
	WavmBinaryPath       string
	UntilHostIoStatePath string
}

var DefaultfogitratorMachineConfig = fogitratorMachineConfig{
	WavmBinaryPath:       "machine.wavm.br",
	UntilHostIoStatePath: "until-host-io-state.bin",
}

type fogMachines struct {
	zeroStep *fogitratorMachine
	hostIo   *fogitratorMachine
}

type fogMachineLoader struct {
	server_common.MachineLoader[fogMachines]
}

func NewfogMachineLoader(config *fogitratorMachineConfig, locator *server_common.MachineLocator) *fogMachineLoader {
	createMachineFunc := func(ctx context.Context, moduleRoot common.Hash) (*fogMachines, error) {
		return createfogMachine(ctx, locator, config, moduleRoot)
	}
	return &fogMachineLoader{
		MachineLoader: *server_common.NewMachineLoader[fogMachines](locator, createMachineFunc),
	}
}

func (a *fogMachineLoader) GetHostIoMachine(ctx context.Context, moduleRoot common.Hash) (*fogitratorMachine, error) {
	machines, err := a.GetMachine(ctx, moduleRoot)
	if err != nil {
		return nil, err
	}
	return machines.hostIo, nil
}

func (a *fogMachineLoader) GetZeroStepMachine(ctx context.Context, moduleRoot common.Hash) (*fogitratorMachine, error) {
	machines, err := a.GetMachine(ctx, moduleRoot)
	if err != nil {
		return nil, err
	}
	return machines.zeroStep, nil
}
