// Code generated by counterfeiter. DO NOT EDIT.
package storagefakes

import (
	"context"
	"sync"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
)

type FakePinger struct {
	PingContextStub        func(context.Context) error
	pingContextMutex       sync.RWMutex
	pingContextArgsForCall []struct {
		arg1 context.Context
	}
	pingContextReturns struct {
		result1 error
	}
	pingContextReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakePinger) PingContext(arg1 context.Context) error {
	fake.pingContextMutex.Lock()
	ret, specificReturn := fake.pingContextReturnsOnCall[len(fake.pingContextArgsForCall)]
	fake.pingContextArgsForCall = append(fake.pingContextArgsForCall, struct {
		arg1 context.Context
	}{arg1})
	stub := fake.PingContextStub
	fakeReturns := fake.pingContextReturns
	fake.recordInvocation("PingContext", []interface{}{arg1})
	fake.pingContextMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakePinger) PingContextCallCount() int {
	fake.pingContextMutex.RLock()
	defer fake.pingContextMutex.RUnlock()
	return len(fake.pingContextArgsForCall)
}

func (fake *FakePinger) PingContextCalls(stub func(context.Context) error) {
	fake.pingContextMutex.Lock()
	defer fake.pingContextMutex.Unlock()
	fake.PingContextStub = stub
}

func (fake *FakePinger) PingContextArgsForCall(i int) context.Context {
	fake.pingContextMutex.RLock()
	defer fake.pingContextMutex.RUnlock()
	argsForCall := fake.pingContextArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakePinger) PingContextReturns(result1 error) {
	fake.pingContextMutex.Lock()
	defer fake.pingContextMutex.Unlock()
	fake.PingContextStub = nil
	fake.pingContextReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakePinger) PingContextReturnsOnCall(i int, result1 error) {
	fake.pingContextMutex.Lock()
	defer fake.pingContextMutex.Unlock()
	fake.PingContextStub = nil
	if fake.pingContextReturnsOnCall == nil {
		fake.pingContextReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.pingContextReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakePinger) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.pingContextMutex.RLock()
	defer fake.pingContextMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakePinger) recordInvocation(key string, args []interface{}) {
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

var _ storage.Pinger = new(FakePinger)
