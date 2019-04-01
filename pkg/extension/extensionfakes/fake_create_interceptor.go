// Code generated by counterfeiter. DO NOT EDIT.
package extensionfakes

import (
	"sync"

	"github.com/Peripli/service-manager/pkg/extension"
)

type FakeCreateInterceptor struct {
	OnAPICreateStub        func(h extension.InterceptCreateOnAPI) extension.InterceptCreateOnAPI
	onAPICreateMutex       sync.RWMutex
	onAPICreateArgsForCall []struct {
		h extension.InterceptCreateOnAPI
	}
	onAPICreateReturns struct {
		result1 extension.InterceptCreateOnAPI
	}
	onAPICreateReturnsOnCall map[int]struct {
		result1 extension.InterceptCreateOnAPI
	}
	OnTxCreateStub        func(f extension.InterceptCreateOnTx) extension.InterceptCreateOnTx
	onTxCreateMutex       sync.RWMutex
	onTxCreateArgsForCall []struct {
		f extension.InterceptCreateOnTx
	}
	onTxCreateReturns struct {
		result1 extension.InterceptCreateOnTx
	}
	onTxCreateReturnsOnCall map[int]struct {
		result1 extension.InterceptCreateOnTx
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeCreateInterceptor) OnAPICreate(h extension.InterceptCreateOnAPI) extension.InterceptCreateOnAPI {
	fake.onAPICreateMutex.Lock()
	ret, specificReturn := fake.onAPICreateReturnsOnCall[len(fake.onAPICreateArgsForCall)]
	fake.onAPICreateArgsForCall = append(fake.onAPICreateArgsForCall, struct {
		h extension.InterceptCreateOnAPI
	}{h})
	fake.recordInvocation("OnAPICreate", []interface{}{h})
	fake.onAPICreateMutex.Unlock()
	if fake.OnAPICreateStub != nil {
		return fake.OnAPICreateStub(h)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.onAPICreateReturns.result1
}

func (fake *FakeCreateInterceptor) OnAPICreateCallCount() int {
	fake.onAPICreateMutex.RLock()
	defer fake.onAPICreateMutex.RUnlock()
	return len(fake.onAPICreateArgsForCall)
}

func (fake *FakeCreateInterceptor) OnAPICreateArgsForCall(i int) extension.InterceptCreateOnAPI {
	fake.onAPICreateMutex.RLock()
	defer fake.onAPICreateMutex.RUnlock()
	return fake.onAPICreateArgsForCall[i].h
}

func (fake *FakeCreateInterceptor) OnAPICreateReturns(result1 extension.InterceptCreateOnAPI) {
	fake.OnAPICreateStub = nil
	fake.onAPICreateReturns = struct {
		result1 extension.InterceptCreateOnAPI
	}{result1}
}

func (fake *FakeCreateInterceptor) OnAPICreateReturnsOnCall(i int, result1 extension.InterceptCreateOnAPI) {
	fake.OnAPICreateStub = nil
	if fake.onAPICreateReturnsOnCall == nil {
		fake.onAPICreateReturnsOnCall = make(map[int]struct {
			result1 extension.InterceptCreateOnAPI
		})
	}
	fake.onAPICreateReturnsOnCall[i] = struct {
		result1 extension.InterceptCreateOnAPI
	}{result1}
}

func (fake *FakeCreateInterceptor) OnTxCreate(f extension.InterceptCreateOnTx) extension.InterceptCreateOnTx {
	fake.onTxCreateMutex.Lock()
	ret, specificReturn := fake.onTxCreateReturnsOnCall[len(fake.onTxCreateArgsForCall)]
	fake.onTxCreateArgsForCall = append(fake.onTxCreateArgsForCall, struct {
		f extension.InterceptCreateOnTx
	}{f})
	fake.recordInvocation("OnTxCreate", []interface{}{f})
	fake.onTxCreateMutex.Unlock()
	if fake.OnTxCreateStub != nil {
		return fake.OnTxCreateStub(f)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.onTxCreateReturns.result1
}

func (fake *FakeCreateInterceptor) OnTxCreateCallCount() int {
	fake.onTxCreateMutex.RLock()
	defer fake.onTxCreateMutex.RUnlock()
	return len(fake.onTxCreateArgsForCall)
}

func (fake *FakeCreateInterceptor) OnTxCreateArgsForCall(i int) extension.InterceptCreateOnTx {
	fake.onTxCreateMutex.RLock()
	defer fake.onTxCreateMutex.RUnlock()
	return fake.onTxCreateArgsForCall[i].f
}

func (fake *FakeCreateInterceptor) OnTxCreateReturns(result1 extension.InterceptCreateOnTx) {
	fake.OnTxCreateStub = nil
	fake.onTxCreateReturns = struct {
		result1 extension.InterceptCreateOnTx
	}{result1}
}

func (fake *FakeCreateInterceptor) OnTxCreateReturnsOnCall(i int, result1 extension.InterceptCreateOnTx) {
	fake.OnTxCreateStub = nil
	if fake.onTxCreateReturnsOnCall == nil {
		fake.onTxCreateReturnsOnCall = make(map[int]struct {
			result1 extension.InterceptCreateOnTx
		})
	}
	fake.onTxCreateReturnsOnCall[i] = struct {
		result1 extension.InterceptCreateOnTx
	}{result1}
}

func (fake *FakeCreateInterceptor) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.onAPICreateMutex.RLock()
	defer fake.onAPICreateMutex.RUnlock()
	fake.onTxCreateMutex.RLock()
	defer fake.onTxCreateMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeCreateInterceptor) recordInvocation(key string, args []interface{}) {
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

var _ extension.CreateInterceptor = new(FakeCreateInterceptor)
