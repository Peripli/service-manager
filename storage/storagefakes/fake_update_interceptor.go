// Code generated by counterfeiter. DO NOT EDIT.
package storagefakes

import (
	"sync"

	"github.com/Peripli/service-manager/storage"
)

type FakeUpdateInterceptor struct {
	AroundTxUpdateStub        func(storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc
	aroundTxUpdateMutex       sync.RWMutex
	aroundTxUpdateArgsForCall []struct {
		arg1 storage.InterceptUpdateAroundTxFunc
	}
	aroundTxUpdateReturns struct {
		result1 storage.InterceptUpdateAroundTxFunc
	}
	aroundTxUpdateReturnsOnCall map[int]struct {
		result1 storage.InterceptUpdateAroundTxFunc
	}
	NameStub        func() string
	nameMutex       sync.RWMutex
	nameArgsForCall []struct {
	}
	nameReturns struct {
		result1 string
	}
	nameReturnsOnCall map[int]struct {
		result1 string
	}
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

func (fake *FakeUpdateInterceptor) AroundTxUpdate(arg1 storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
	fake.aroundTxUpdateMutex.Lock()
	ret, specificReturn := fake.aroundTxUpdateReturnsOnCall[len(fake.aroundTxUpdateArgsForCall)]
	fake.aroundTxUpdateArgsForCall = append(fake.aroundTxUpdateArgsForCall, struct {
		arg1 storage.InterceptUpdateAroundTxFunc
	}{arg1})
	fake.recordInvocation("AroundTxUpdate", []interface{}{arg1})
	fake.aroundTxUpdateMutex.Unlock()
	if fake.AroundTxUpdateStub != nil {
		return fake.AroundTxUpdateStub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.aroundTxUpdateReturns
	return fakeReturns.result1
}

func (fake *FakeUpdateInterceptor) AroundTxUpdateCallCount() int {
	fake.aroundTxUpdateMutex.RLock()
	defer fake.aroundTxUpdateMutex.RUnlock()
	return len(fake.aroundTxUpdateArgsForCall)
}

func (fake *FakeUpdateInterceptor) AroundTxUpdateCalls(stub func(storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc) {
	fake.aroundTxUpdateMutex.Lock()
	defer fake.aroundTxUpdateMutex.Unlock()
	fake.AroundTxUpdateStub = stub
}

func (fake *FakeUpdateInterceptor) AroundTxUpdateArgsForCall(i int) storage.InterceptUpdateAroundTxFunc {
	fake.aroundTxUpdateMutex.RLock()
	defer fake.aroundTxUpdateMutex.RUnlock()
	argsForCall := fake.aroundTxUpdateArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeUpdateInterceptor) AroundTxUpdateReturns(result1 storage.InterceptUpdateAroundTxFunc) {
	fake.aroundTxUpdateMutex.Lock()
	defer fake.aroundTxUpdateMutex.Unlock()
	fake.AroundTxUpdateStub = nil
	fake.aroundTxUpdateReturns = struct {
		result1 storage.InterceptUpdateAroundTxFunc
	}{result1}
}

func (fake *FakeUpdateInterceptor) AroundTxUpdateReturnsOnCall(i int, result1 storage.InterceptUpdateAroundTxFunc) {
	fake.aroundTxUpdateMutex.Lock()
	defer fake.aroundTxUpdateMutex.Unlock()
	fake.AroundTxUpdateStub = nil
	if fake.aroundTxUpdateReturnsOnCall == nil {
		fake.aroundTxUpdateReturnsOnCall = make(map[int]struct {
			result1 storage.InterceptUpdateAroundTxFunc
		})
	}
	fake.aroundTxUpdateReturnsOnCall[i] = struct {
		result1 storage.InterceptUpdateAroundTxFunc
	}{result1}
}

func (fake *FakeUpdateInterceptor) Name() string {
	fake.nameMutex.Lock()
	ret, specificReturn := fake.nameReturnsOnCall[len(fake.nameArgsForCall)]
	fake.nameArgsForCall = append(fake.nameArgsForCall, struct {
	}{})
	fake.recordInvocation("Name", []interface{}{})
	fake.nameMutex.Unlock()
	if fake.NameStub != nil {
		return fake.NameStub()
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.nameReturns
	return fakeReturns.result1
}

func (fake *FakeUpdateInterceptor) NameCallCount() int {
	fake.nameMutex.RLock()
	defer fake.nameMutex.RUnlock()
	return len(fake.nameArgsForCall)
}

func (fake *FakeUpdateInterceptor) NameCalls(stub func() string) {
	fake.nameMutex.Lock()
	defer fake.nameMutex.Unlock()
	fake.NameStub = stub
}

func (fake *FakeUpdateInterceptor) NameReturns(result1 string) {
	fake.nameMutex.Lock()
	defer fake.nameMutex.Unlock()
	fake.NameStub = nil
	fake.nameReturns = struct {
		result1 string
	}{result1}
}

func (fake *FakeUpdateInterceptor) NameReturnsOnCall(i int, result1 string) {
	fake.nameMutex.Lock()
	defer fake.nameMutex.Unlock()
	fake.NameStub = nil
	if fake.nameReturnsOnCall == nil {
		fake.nameReturnsOnCall = make(map[int]struct {
			result1 string
		})
	}
	fake.nameReturnsOnCall[i] = struct {
		result1 string
	}{result1}
}

func (fake *FakeUpdateInterceptor) OnTxUpdate(arg1 storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
	fake.onTxUpdateMutex.Lock()
	ret, specificReturn := fake.onTxUpdateReturnsOnCall[len(fake.onTxUpdateArgsForCall)]
	fake.onTxUpdateArgsForCall = append(fake.onTxUpdateArgsForCall, struct {
		arg1 storage.InterceptUpdateOnTxFunc
	}{arg1})
	fake.recordInvocation("OnTxUpdate", []interface{}{arg1})
	fake.onTxUpdateMutex.Unlock()
	if fake.OnTxUpdateStub != nil {
		return fake.OnTxUpdateStub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.onTxUpdateReturns
	return fakeReturns.result1
}

func (fake *FakeUpdateInterceptor) OnTxUpdateCallCount() int {
	fake.onTxUpdateMutex.RLock()
	defer fake.onTxUpdateMutex.RUnlock()
	return len(fake.onTxUpdateArgsForCall)
}

func (fake *FakeUpdateInterceptor) OnTxUpdateCalls(stub func(storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc) {
	fake.onTxUpdateMutex.Lock()
	defer fake.onTxUpdateMutex.Unlock()
	fake.OnTxUpdateStub = stub
}

func (fake *FakeUpdateInterceptor) OnTxUpdateArgsForCall(i int) storage.InterceptUpdateOnTxFunc {
	fake.onTxUpdateMutex.RLock()
	defer fake.onTxUpdateMutex.RUnlock()
	argsForCall := fake.onTxUpdateArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeUpdateInterceptor) OnTxUpdateReturns(result1 storage.InterceptUpdateOnTxFunc) {
	fake.onTxUpdateMutex.Lock()
	defer fake.onTxUpdateMutex.Unlock()
	fake.OnTxUpdateStub = nil
	fake.onTxUpdateReturns = struct {
		result1 storage.InterceptUpdateOnTxFunc
	}{result1}
}

func (fake *FakeUpdateInterceptor) OnTxUpdateReturnsOnCall(i int, result1 storage.InterceptUpdateOnTxFunc) {
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

func (fake *FakeUpdateInterceptor) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.aroundTxUpdateMutex.RLock()
	defer fake.aroundTxUpdateMutex.RUnlock()
	fake.nameMutex.RLock()
	defer fake.nameMutex.RUnlock()
	fake.onTxUpdateMutex.RLock()
	defer fake.onTxUpdateMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeUpdateInterceptor) recordInvocation(key string, args []interface{}) {
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

var _ storage.UpdateInterceptor = new(FakeUpdateInterceptor)
