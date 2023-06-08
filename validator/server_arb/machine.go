// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/fogr/blob/master/LICENSE

package server_fog

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#include "fogitrator.h"

ResolvedPreimage preimageResolverC(size_t context, const uint8_t* hash);
*/
import "C"
import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/FOGRCC/fogr/validator"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

type MachineInterface interface {
	CloneMachineInterface() MachineInterface
	GetStepCount() uint64
	IsRunning() bool
	ValidForStep(uint64) bool
	Status() uint8
	Step(context.Context, uint64) error
	Hash() common.Hash
	GetGlobalState() validator.GoGlobalState
	ProveNextStep() []byte
	Freeze()
	Destroy()
}

// fogitratorMachine holds an fogitrator machine pointer, and manages its lifetime
type fogitratorMachine struct {
	mutex     sync.Mutex // needed because go finalizers don't synchronize (meaning they aren't thread safe)
	ptr       *C.struct_Machine
	contextId *int64 // has a finalizer attached to remove the preimage resolver from the global map
	frozen    bool   // does not allow anything that changes machine state, not cloned with the machine
}

// Assert that fogitratorMachine implements MachineInterface
var _ MachineInterface = (*fogitratorMachine)(nil)

var preimageResolvers sync.Map
var lastPreimageResolverId int64 // atomic

// Any future calls to this machine will result in a panic
func (m *fogitratorMachine) Destroy() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.ptr != nil {
		C.fogitrator_free_machine(m.ptr)
		m.ptr = nil
		// We no longer need a finalizer
		runtime.SetFinalizer(m, nil)
	}
	m.contextId = nil
}

func freeContextId(context *int64) {
	preimageResolvers.Delete(*context)
}

func machineFromPointer(ptr *C.struct_Machine) *fogitratorMachine {
	if ptr == nil {
		return nil
	}
	mach := &fogitratorMachine{ptr: ptr}
	C.fogitrator_set_preimage_resolver(ptr, (*[0]byte)(C.preimageResolverC))
	runtime.SetFinalizer(mach, (*fogitratorMachine).Destroy)
	return mach
}

func LoadSimpleMachine(wasm string, libraries []string) (*fogitratorMachine, error) {
	cWasm := C.CString(wasm)
	cLibraries := CreateCStringList(libraries)
	mach := C.fogitrator_load_machine(cWasm, cLibraries, C.long(len(libraries)))
	C.free(unsafe.Pointer(cWasm))
	FreeCStringList(cLibraries, len(libraries))
	if mach == nil {
		return nil, errors.Errorf("failed to load simple machine at path %v", wasm)
	}
	return machineFromPointer(mach), nil
}

func (m *fogitratorMachine) Freeze() {
	m.frozen = true
}

// Even if origin is frozen - clone is not
func (m *fogitratorMachine) Clone() *fogitratorMachine {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	newMach := machineFromPointer(C.fogitrator_clone_machine(m.ptr))
	newMach.contextId = m.contextId
	return newMach
}

func (m *fogitratorMachine) CloneMachineInterface() MachineInterface {
	return m.Clone()
}

func (m *fogitratorMachine) SetGlobalState(globalState validator.GoGlobalState) error {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.frozen {
		return errors.New("machine frozen")
	}
	cGlobalState := GlobalStateToC(globalState)
	C.fogitrator_set_global_state(m.ptr, cGlobalState)
	return nil
}

func (m *fogitratorMachine) GetGlobalState() validator.GoGlobalState {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	cGlobalState := C.fogitrator_global_state(m.ptr)
	return GlobalStateFromC(cGlobalState)
}

func (m *fogitratorMachine) GetStepCount() uint64 {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return uint64(C.fogitrator_get_num_steps(m.ptr))
}

func (m *fogitratorMachine) IsRunning() bool {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return C.fogitrator_get_status(m.ptr) == C.fogITRATOR_MACHINE_STATUS_RUNNING
}

func (m *fogitratorMachine) IsErrored() bool {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return C.fogitrator_get_status(m.ptr) == C.fogITRATOR_MACHINE_STATUS_ERRORED
}

func (m *fogitratorMachine) Status() uint8 {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return uint8(C.fogitrator_get_status(m.ptr))
}

func (m *fogitratorMachine) ValidForStep(requestedStep uint64) bool {
	haveStep := m.GetStepCount()
	if haveStep > requestedStep {
		return false
	} else if haveStep == requestedStep {
		return true
	} else { // haveStep < requestedStep
		// if the machine is halted, its state persists for future steps
		return !m.IsRunning()
	}
}

func manageConditionByte(ctx context.Context) (*C.uint8_t, func()) {
	var zero C.uint8_t
	conditionByte := &zero

	doneEarlyChan := make(chan struct{})

	go (func() {
		defer runtime.KeepAlive(conditionByte)
		select {
		case <-ctx.Done():
			C.atomic_u8_store(conditionByte, 1)
		case <-doneEarlyChan:
		}
	})()

	cancel := func() {
		runtime.KeepAlive(conditionByte)
		close(doneEarlyChan)
	}

	return conditionByte, cancel
}

func (m *fogitratorMachine) Step(ctx context.Context, count uint64) error {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.frozen {
		return errors.New("machine frozen")
	}
	conditionByte, cancel := manageConditionByte(ctx)
	defer cancel()

	err := C.fogitrator_step(m.ptr, C.uint64_t(count), conditionByte)
	if err != nil {
		errString := C.GoString(err)
		C.free(unsafe.Pointer(err))
		return errors.New(errString)
	}

	return ctx.Err()
}

func (m *fogitratorMachine) StepUntilHostIo(ctx context.Context) error {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.frozen {
		return errors.New("machine frozen")
	}

	conditionByte, cancel := manageConditionByte(ctx)
	defer cancel()

	C.fogitrator_step_until_host_io(m.ptr, conditionByte)

	return ctx.Err()
}

func (m *fogitratorMachine) Hash() (hash common.Hash) {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	bytes := C.fogitrator_hash(m.ptr)
	for i, b := range bytes.bytes {
		hash[i] = byte(b)
	}
	return
}

func (m *fogitratorMachine) GetModuleRoot() (hash common.Hash) {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	bytes := C.fogitrator_module_root(m.ptr)
	for i, b := range bytes.bytes {
		hash[i] = byte(b)
	}
	return
}
func (m *fogitratorMachine) ProveNextStep() []byte {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	rustProof := C.fogitrator_gen_proof(m.ptr)
	proofBytes := C.GoBytes(unsafe.Pointer(rustProof.ptr), C.int(rustProof.len))
	C.fogitrator_free_proof(rustProof)

	return proofBytes
}

func (m *fogitratorMachine) SerializeState(path string) error {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	cPath := C.CString(path)
	status := C.fogitrator_serialize_state(m.ptr, cPath)
	C.free(unsafe.Pointer(cPath))

	if status != 0 {
		return errors.New("failed to serialize machine state")
	} else {
		return nil
	}
}

func (m *fogitratorMachine) DeserializeAndReplaceState(path string) error {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.frozen {
		return errors.New("machine frozen")
	}

	cPath := C.CString(path)
	status := C.fogitrator_deserialize_and_replace_state(m.ptr, cPath)
	C.free(unsafe.Pointer(cPath))

	if status != 0 {
		return errors.New("failed to deserialize machine state")
	} else {
		return nil
	}
}

func (m *fogitratorMachine) AddSequencerInboxMessage(index uint64, data []byte) error {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.frozen {
		return errors.New("machine frozen")
	}
	cbyte := CreateCByteArray(data)
	status := C.fogitrator_add_inbox_message(m.ptr, C.uint64_t(0), C.uint64_t(index), cbyte)
	DestroyCByteArray(cbyte)
	if status != 0 {
		return errors.New("failed to add sequencer inbox message")
	} else {
		return nil
	}
}

func (m *fogitratorMachine) AddDelayedInboxMessage(index uint64, data []byte) error {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.frozen {
		return errors.New("machine frozen")
	}

	cbyte := CreateCByteArray(data)
	status := C.fogitrator_add_inbox_message(m.ptr, C.uint64_t(1), C.uint64_t(index), cbyte)
	DestroyCByteArray(cbyte)
	if status != 0 {
		return errors.New("failed to add sequencer inbox message")
	} else {
		return nil
	}
}

type GoPreimageResolver = func(common.Hash) ([]byte, error)

//export preimageResolver
func preimageResolver(context C.size_t, ptr unsafe.Pointer) C.ResolvedPreimage {
	var hash common.Hash
	input := (*[1 << 30]byte)(ptr)[:32]
	copy(hash[:], input)
	resolver, ok := preimageResolvers.Load(int64(context))
	if !ok {
		return C.ResolvedPreimage{
			len: -1,
		}
	}
	resolverFunc, ok := resolver.(GoPreimageResolver)
	if !ok {
		log.Warn("preimage resolver has wrong type")
		return C.ResolvedPreimage{
			len: -1,
		}
	}
	preimage, err := resolverFunc(hash)
	if err != nil {
		log.Error("preimage resolution failed", "err", err)
		return C.ResolvedPreimage{
			len: -1,
		}
	}
	return C.ResolvedPreimage{
		ptr: (*C.uint8_t)(C.CBytes(preimage)),
		len: (C.ptrdiff_t)(len(preimage)),
	}
}

func (m *fogitratorMachine) SetPreimageResolver(resolver GoPreimageResolver) error {
	defer runtime.KeepAlive(m)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.frozen {
		return errors.New("machine frozen")
	}
	id := atomic.AddInt64(&lastPreimageResolverId, 1)
	preimageResolvers.Store(id, resolver)
	m.contextId = &id
	runtime.SetFinalizer(m.contextId, freeContextId)
	C.fogitrator_set_context(m.ptr, C.uint64_t(id))
	return nil
}
