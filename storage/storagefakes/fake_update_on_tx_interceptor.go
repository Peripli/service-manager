// Code generated by counterfeiter. DO NOT EDIT.
package storagefakes

import (
	"sync"

	"github.com/Peripli/service-manager/storage"
)

type FakeUpdateOnTxInterceptor struct {
	OnTxUpdateStub        func(storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc
	onTxUpdateMutex       sync.RWMutex
	onTxUpdateArgsForCall []struct {
		arg1 storage.InterceptUpdateOnTxFunc
	}
	onTxUpdateReturns struct {
		result1 storage.InterceptUpdateOnTxFunc
	}
	onTxUpdateReturnsOnCall map[int]struct {
		result1 storage.InterceptUpdateOnTxFunc
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeUpdateOnTxInterceptor) OnTxUpdate(arg1 storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
	fake.onTxUpdateMutex.Lock()
	ret, specificReturn := fake.onTxUpdateReturnsOnCall[len(fake.onTxUpdateArgsForCall)]
	fake.onTxUpdateArgsForCall = append(fake.onTxUpdateArgsForCall, struct {
		arg1 storage.InterceptUpdateOnTxFunc
	}{arg1})
	stub := fake.OnTxUpdateStub
	fakeReturns := fake.onTxUpdateReturns
	fake.recordInvocation("OnTxUpdate", []interface{}{arg1})
	fake.onTxUpdateMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeUpdateOnTxInterceptor) OnTxUpdateCallCount() int {
	fake.onTxUpdateMutex.RLock()
	defer fake.onTxUpdateMutex.RUnlock()
	return len(fake.onTxUpdateArgsForCall)
}

func (fake *FakeUpdateOnTxInterceptor) OnTxUpdateCalls(stub func(storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc) {
	fake.onTxUpdateMutex.Lock()
	defer fake.onTxUpdateMutex.Unlock()
	fake.OnTxUpdateStub = stub
}

func (fake *FakeUpdateOnTxInterceptor) OnTxUpdateArgsForCall(i int) storage.InterceptUpdateOnTxFunc {
	fake.onTxUpdateMutex.RLock()
	defer fake.onTxUpdateMutex.RUnlock()
	argsForCall := fake.onTxUpdateArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeUpdateOnTxInterceptor) OnTxUpdateReturns(result1 storage.InterceptUpdateOnTxFunc) {
	fake.onTxUpdateMutex.Lock()
	defer fake.onTxUpdateMutex.Unlock()
	fake.OnTxUpdateStub = nil
	fake.onTxUpdateReturns = struct {
		result1 storage.InterceptUpdateOnTxFunc
	}{result1}
}

func (fake *FakeUpdateOnTxInterceptor) OnTxUpdateReturnsOnCall(i int, result1 storage.InterceptUpdateOnTxFunc) {
	fake.onTxUpdateMutex.Lock()
	defer fake.onTxUpdateMutex.Unlock()
	fake.OnTxUpdateStub = nil
	if fake.onTxUpdateReturnsOnCall == nil {
		fake.onTxUpdateReturnsOnCall = make(map[int]struct {
			result1 storage.InterceptUpdateOnTxFunc
		})
	}
	fake.onTxUpdateReturnsOnCall[i] = struct {
		result1 storage.InterceptUpdateOnTxFunc
	}{result1}
}

func (fake *FakeUpdateOnTxInterceptor) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.onTxUpdateMutex.RLock()
	defer fake.onTxUpdateMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeUpdateOnTxInterceptor) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ storage.UpdateOnTxInterceptor = new(FakeUpdateOnTxInterceptor)
