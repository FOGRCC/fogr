package valnode

import (
	"context"

	"github.com/FOGRCC/fogr/validator"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	flag "github.com/spf13/pflag"

	"github.com/FOGRCC/fogr/validator/server_api"
	"github.com/FOGRCC/fogr/validator/server_common"
	"github.com/FOGRCC/fogr/validator/server_fog"
	"github.com/FOGRCC/fogr/validator/server_jit"
)

type WasmConfig struct {
	RootPath string `koanf:"root-path"`
}

func WasmConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".root-path", DefaultWasmConfig.RootPath, "path to machine folders, each containing wasm files (machine.wavm.br, replay.wasm)")
}

var DefaultWasmConfig = WasmConfig{
	RootPath: "",
}

type Config struct {
	UseJit     bool                               `koanf:"use-jit"`
	ApiAuth    bool                               `koanf:"api-auth"`
	ApiPublic  bool                               `koanf:"api-public"`
	fogitrator server_fog.fogitratorSpawnerConfig `koanf:"fogitrator" reload:"hot"`
	Jit        server_jit.JitSpawnerConfig        `koanf:"jit" reload:"hot"`
	Wasm       WasmConfig                         `koanf:"wasm"`
}

type ValidationConfigFetcher func() *Config

var DefaultValidationConfig = Config{
	UseJit:     true,
	Jit:        server_jit.DefaultJitSpawnerConfig,
	ApiAuth:    true,
	ApiPublic:  false,
	fogitrator: server_fog.DefaultfogitratorSpawnerConfig,
	Wasm:       DefaultWasmConfig,
}

var TestValidationConfig = Config{
	UseJit:     true,
	Jit:        server_jit.DefaultJitSpawnerConfig,
	ApiAuth:    false,
	ApiPublic:  true,
	fogitrator: server_fog.DefaultfogitratorSpawnerConfig,
	Wasm:       DefaultWasmConfig,
}

func ValidationConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".use-jit", DefaultValidationConfig.UseJit, "use jit for validation")
	f.Bool(prefix+".api-auth", DefaultValidationConfig.ApiAuth, "validate is an authenticated API")
	f.Bool(prefix+".api-public", DefaultValidationConfig.ApiPublic, "validate is a public API")
	server_fog.fogitratorSpawnerConfigAddOptions(prefix+".fogitrator", f)
	server_jit.JitSpawnerConfigAddOptions(prefix+".jit", f)
	WasmConfigAddOptions(prefix+".wasm", f)
}

type ValidationNode struct {
	config     ValidationConfigFetcher
	fogSpawner *server_fog.fogitratorSpawner
	jitSpawner *server_jit.JitSpawner
}

func CreateValidationNode(configFetcher ValidationConfigFetcher, stack *node.Node, fatalErrChan chan error) (*ValidationNode, error) {
	config := configFetcher()
	locator, err := server_common.NewMachineLocator(config.Wasm.RootPath)
	if err != nil {
		return nil, err
	}
	fogConfigFetcher := func() *server_fog.fogitratorSpawnerConfig {
		return &configFetcher().fogitrator
	}
	fogSpawner, err := server_fog.NewfogitratorSpawner(locator, fogConfigFetcher)
	if err != nil {
		return nil, err
	}
	node.DefaultAuthModules = []string{server_api.Namespace}
	var serverAPI *server_api.ExecServerAPI
	var jitSpawner *server_jit.JitSpawner
	if config.UseJit {
		jitConfigFetcher := func() *server_jit.JitSpawnerConfig { return &configFetcher().Jit }
		var err error
		jitSpawner, err = server_jit.NewJitSpawner(locator, jitConfigFetcher, fatalErrChan)
		if err != nil {
			return nil, err
		}
		serverAPI = server_api.NewExecutionServerAPI(jitSpawner, fogSpawner, fogConfigFetcher)
	} else {
		serverAPI = server_api.NewExecutionServerAPI(fogSpawner, fogSpawner, fogConfigFetcher)
	}
	valAPIs := []rpc.API{{
		Namespace:     server_api.Namespace,
		Version:       "1.0",
		Service:       serverAPI,
		Public:        config.ApiPublic,
		Authenticated: config.ApiAuth,
	}}
	stack.RegisterAPIs(valAPIs)

	return &ValidationNode{configFetcher, fogSpawner, jitSpawner}, nil
}

func (v *ValidationNode) Start(ctx context.Context) error {
	if err := v.fogSpawner.Start(ctx); err != nil {
		return err
	}
	if v.jitSpawner != nil {
		if err := v.jitSpawner.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (v *ValidationNode) GetExec() validator.ExecutionSpawner {
	return v.fogSpawner
}
