// Code generated by counterfeiter. DO NOT EDIT.
package httpfakes

import (
	"context"
	"sync"

	"github.com/Peripli/service-manager/pkg/security/http"
)

type FakeTokenVerifier struct {
	VerifyStub        func(context.Context, string) (http.TokenData, error)
	verifyMutex       sync.RWMutex
	verifyArgsForCall []struct {
		arg1 context.Context
		arg2 string
	}
	verifyReturns struct {
		result1 http.TokenData
		result2 error
	}
	verifyReturnsOnCall map[int]struct {
		result1 http.TokenData
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeTokenVerifier) Verify(arg1 context.Context, arg2 string) (http.TokenData, error) {
	fake.verifyMutex.Lock()
	ret, specificReturn := fake.verifyReturnsOnCall[len(fake.verifyArgsForCall)]
	fake.verifyArgsForCall = append(fake.verifyArgsForCall, struct {
		arg1 context.Context
		arg2 string
	}{arg1, arg2})
	stub := fake.VerifyStub
	fakeReturns := fake.verifyReturns
	fake.recordInvocation("Verify", []interface{}{arg1, arg2})
	fake.verifyMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeTokenVerifier) VerifyCallCount() int {
	fake.verifyMutex.RLock()
	defer fake.verifyMutex.RUnlock()
	return len(fake.verifyArgsForCall)
}

func (fake *FakeTokenVerifier) VerifyCalls(stub func(context.Context, string) (http.TokenData, error)) {
	fake.verifyMutex.Lock()
	defer fake.verifyMutex.Unlock()
	fake.VerifyStub = stub
}

func (fake *FakeTokenVerifier) VerifyArgsForCall(i int) (context.Context, string) {
	fake.verifyMutex.RLock()
	defer fake.verifyMutex.RUnlock()
	argsForCall := fake.verifyArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeTokenVerifier) VerifyReturns(result1 http.TokenData, result2 error) {
	fake.verifyMutex.Lock()
	defer fake.verifyMutex.Unlock()
	fake.VerifyStub = nil
	fake.verifyReturns = struct {
		result1 http.TokenData
		result2 error
	}{result1, result2}
}

func (fake *FakeTokenVerifier) VerifyReturnsOnCall(i int, result1 http.TokenData, result2 error) {
	fake.verifyMutex.Lock()
	defer fake.verifyMutex.Unlock()
	fake.VerifyStub = nil
	if fake.verifyReturnsOnCall == nil {
		fake.verifyReturnsOnCall = make(map[int]struct {
			result1 http.TokenData
			result2 error
		})
	}
	fake.verifyReturnsOnCall[i] = struct {
		result1 http.TokenData
		result2 error
	}{result1, result2}
}

func (fake *FakeTokenVerifier) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.verifyMutex.RLock()
	defer fake.verifyMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeTokenVerifier) recordInvocation(key string, args []interface{}) {
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

var _ http.TokenVerifier = new(FakeTokenVerifier)
