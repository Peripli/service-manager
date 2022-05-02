// Code generated by counterfeiter. DO NOT EDIT.
package storagefakes

import (
	"sync"

	"github.com/Peripli/service-manager/storage"
)

type FakeDeleteOnTxInterceptorProvider struct {
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
	ProvideStub        func() storage.DeleteOnTxInterceptor
	provideMutex       sync.RWMutex
	provideArgsForCall []struct {
	}
	provideReturns struct {
		result1 storage.DeleteOnTxInterceptor
	}
	provideReturnsOnCall map[int]struct {
		result1 storage.DeleteOnTxInterceptor
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeDeleteOnTxInterceptorProvider) Name() string {
	fake.nameMutex.Lock()
	ret, specificReturn := fake.nameReturnsOnCall[len(fake.nameArgsForCall)]
	fake.nameArgsForCall = append(fake.nameArgsForCall, struct {
	}{})
	stub := fake.NameStub
	fakeReturns := fake.nameReturns
	fake.recordInvocation("Name", []interface{}{})
	fake.nameMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeDeleteOnTxInterceptorProvider) NameCallCount() int {
	fake.nameMutex.RLock()
	defer fake.nameMutex.RUnlock()
	return len(fake.nameArgsForCall)
}

func (fake *FakeDeleteOnTxInterceptorProvider) NameCalls(stub func() string) {
	fake.nameMutex.Lock()
	defer fake.nameMutex.Unlock()
	fake.NameStub = stub
}

func (fake *FakeDeleteOnTxInterceptorProvider) NameReturns(result1 string) {
	fake.nameMutex.Lock()
	defer fake.nameMutex.Unlock()
	fake.NameStub = nil
	fake.nameReturns = struct {
		result1 string
	}{result1}
}

func (fake *FakeDeleteOnTxInterceptorProvider) NameReturnsOnCall(i int, result1 string) {
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

func (fake *FakeDeleteOnTxInterceptorProvider) Provide() storage.DeleteOnTxInterceptor {
	fake.provideMutex.Lock()
	ret, specificReturn := fake.provideReturnsOnCall[len(fake.provideArgsForCall)]
	fake.provideArgsForCall = append(fake.provideArgsForCall, struct {
	}{})
	stub := fake.ProvideStub
	fakeReturns := fake.provideReturns
	fake.recordInvocation("Provide", []interface{}{})
	fake.provideMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeDeleteOnTxInterceptorProvider) ProvideCallCount() int {
	fake.provideMutex.RLock()
	defer fake.provideMutex.RUnlock()
	return len(fake.provideArgsForCall)
}

func (fake *FakeDeleteOnTxInterceptorProvider) ProvideCalls(stub func() storage.DeleteOnTxInterceptor) {
	fake.provideMutex.Lock()
	defer fake.provideMutex.Unlock()
	fake.ProvideStub = stub
}

func (fake *FakeDeleteOnTxInterceptorProvider) ProvideReturns(result1 storage.DeleteOnTxInterceptor) {
	fake.provideMutex.Lock()
	defer fake.provideMutex.Unlock()
	fake.ProvideStub = nil
	fake.provideReturns = struct {
		result1 storage.DeleteOnTxInterceptor
	}{result1}
}

func (fake *FakeDeleteOnTxInterceptorProvider) ProvideReturnsOnCall(i int, result1 storage.DeleteOnTxInterceptor) {
	fake.provideMutex.Lock()
	defer fake.provideMutex.Unlock()
	fake.ProvideStub = nil
	if fake.provideReturnsOnCall == nil {
		fake.provideReturnsOnCall = make(map[int]struct {
			result1 storage.DeleteOnTxInterceptor
		})
	}
	fake.provideReturnsOnCall[i] = struct {
		result1 storage.DeleteOnTxInterceptor
	}{result1}
}

func (fake *FakeDeleteOnTxInterceptorProvider) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.nameMutex.RLock()
	defer fake.nameMutex.RUnlock()
	fake.provideMutex.RLock()
	defer fake.provideMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeDeleteOnTxInterceptorProvider) recordInvocation(key string, args []interface{}) {
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

var _ storage.DeleteOnTxInterceptorProvider = new(FakeDeleteOnTxInterceptorProvider)
