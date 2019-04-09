// Code generated by counterfeiter. DO NOT EDIT.
package storagefakes

import (
	"sync"

	"github.com/Peripli/service-manager/storage"
)

type FakeDeleteInterceptorProvider struct {
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
	ProvideStub        func() storage.DeleteInterceptor
	provideMutex       sync.RWMutex
	provideArgsForCall []struct {
	}
	provideReturns struct {
		result1 storage.DeleteInterceptor
	}
	provideReturnsOnCall map[int]struct {
		result1 storage.DeleteInterceptor
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeDeleteInterceptorProvider) Name() string {
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

func (fake *FakeDeleteInterceptorProvider) NameCallCount() int {
	fake.nameMutex.RLock()
	defer fake.nameMutex.RUnlock()
	return len(fake.nameArgsForCall)
}

func (fake *FakeDeleteInterceptorProvider) NameCalls(stub func() string) {
	fake.nameMutex.Lock()
	defer fake.nameMutex.Unlock()
	fake.NameStub = stub
}

func (fake *FakeDeleteInterceptorProvider) NameReturns(result1 string) {
	fake.nameMutex.Lock()
	defer fake.nameMutex.Unlock()
	fake.NameStub = nil
	fake.nameReturns = struct {
		result1 string
	}{result1}
}

func (fake *FakeDeleteInterceptorProvider) NameReturnsOnCall(i int, result1 string) {
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

func (fake *FakeDeleteInterceptorProvider) Provide() storage.DeleteInterceptor {
	fake.provideMutex.Lock()
	ret, specificReturn := fake.provideReturnsOnCall[len(fake.provideArgsForCall)]
	fake.provideArgsForCall = append(fake.provideArgsForCall, struct {
	}{})
	fake.recordInvocation("Provide", []interface{}{})
	fake.provideMutex.Unlock()
	if fake.ProvideStub != nil {
		return fake.ProvideStub()
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.provideReturns
	return fakeReturns.result1
}

func (fake *FakeDeleteInterceptorProvider) ProvideCallCount() int {
	fake.provideMutex.RLock()
	defer fake.provideMutex.RUnlock()
	return len(fake.provideArgsForCall)
}

func (fake *FakeDeleteInterceptorProvider) ProvideCalls(stub func() storage.DeleteInterceptor) {
	fake.provideMutex.Lock()
	defer fake.provideMutex.Unlock()
	fake.ProvideStub = stub
}

func (fake *FakeDeleteInterceptorProvider) ProvideReturns(result1 storage.DeleteInterceptor) {
	fake.provideMutex.Lock()
	defer fake.provideMutex.Unlock()
	fake.ProvideStub = nil
	fake.provideReturns = struct {
		result1 storage.DeleteInterceptor
	}{result1}
}

func (fake *FakeDeleteInterceptorProvider) ProvideReturnsOnCall(i int, result1 storage.DeleteInterceptor) {
	fake.provideMutex.Lock()
	defer fake.provideMutex.Unlock()
	fake.ProvideStub = nil
	if fake.provideReturnsOnCall == nil {
		fake.provideReturnsOnCall = make(map[int]struct {
			result1 storage.DeleteInterceptor
		})
	}
	fake.provideReturnsOnCall[i] = struct {
		result1 storage.DeleteInterceptor
	}{result1}
}

func (fake *FakeDeleteInterceptorProvider) Invocations() map[string][][]interface{} {
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

func (fake *FakeDeleteInterceptorProvider) recordInvocation(key string, args []interface{}) {
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

var _ storage.DeleteInterceptorProvider = new(FakeDeleteInterceptorProvider)
