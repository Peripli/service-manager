// Code generated by counterfeiter. DO NOT EDIT.
package storagefakes

import (
	"context"
	"sync"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

type FakeStorage struct {
	CloseStub        func() error
	closeMutex       sync.RWMutex
	closeArgsForCall []struct {
	}
	closeReturns struct {
		result1 error
	}
	closeReturnsOnCall map[int]struct {
		result1 error
	}
	CountStub        func(context.Context, types.ObjectType, ...query.Criterion) (int, error)
	countMutex       sync.RWMutex
	countArgsForCall []struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 []query.Criterion
	}
	countReturns struct {
		result1 int
		result2 error
	}
	countReturnsOnCall map[int]struct {
		result1 int
		result2 error
	}
	CreateStub        func(context.Context, types.Object) (types.Object, error)
	createMutex       sync.RWMutex
	createArgsForCall []struct {
		arg1 context.Context
		arg2 types.Object
	}
	createReturns struct {
		result1 types.Object
		result2 error
	}
	createReturnsOnCall map[int]struct {
		result1 types.Object
		result2 error
	}
	DeleteStub        func(context.Context, types.ObjectType, ...query.Criterion) error
	deleteMutex       sync.RWMutex
	deleteArgsForCall []struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 []query.Criterion
	}
	deleteReturns struct {
		result1 error
	}
	deleteReturnsOnCall map[int]struct {
		result1 error
	}
	DeleteReturningStub        func(context.Context, types.ObjectType, ...query.Criterion) (types.ObjectList, error)
	deleteReturningMutex       sync.RWMutex
	deleteReturningArgsForCall []struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 []query.Criterion
	}
	deleteReturningReturns struct {
		result1 types.ObjectList
		result2 error
	}
	deleteReturningReturnsOnCall map[int]struct {
		result1 types.ObjectList
		result2 error
	}
	GetStub        func(context.Context, types.ObjectType, ...query.Criterion) (types.Object, error)
	getMutex       sync.RWMutex
	getArgsForCall []struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 []query.Criterion
	}
	getReturns struct {
		result1 types.Object
		result2 error
	}
	getReturnsOnCall map[int]struct {
		result1 types.Object
		result2 error
	}
	GetForUpdateStub        func(context.Context, types.ObjectType, ...query.Criterion) (types.Object, error)
	getForUpdateMutex       sync.RWMutex
	getForUpdateArgsForCall []struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 []query.Criterion
	}
	getForUpdateReturns struct {
		result1 types.Object
		result2 error
	}
	getForUpdateReturnsOnCall map[int]struct {
		result1 types.Object
		result2 error
	}
	InTransactionStub        func(context.Context, func(ctx context.Context, storage storage.Repository) error) error
	inTransactionMutex       sync.RWMutex
	inTransactionArgsForCall []struct {
		arg1 context.Context
		arg2 func(ctx context.Context, storage storage.Repository) error
	}
	inTransactionReturns struct {
		result1 error
	}
	inTransactionReturnsOnCall map[int]struct {
		result1 error
	}
	IntroduceStub        func(storage.Entity)
	introduceMutex       sync.RWMutex
	introduceArgsForCall []struct {
		arg1 storage.Entity
	}
	ListStub        func(context.Context, types.ObjectType, ...query.Criterion) (types.ObjectList, error)
	listMutex       sync.RWMutex
	listArgsForCall []struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 []query.Criterion
	}
	listReturns struct {
		result1 types.ObjectList
		result2 error
	}
	listReturnsOnCall map[int]struct {
		result1 types.ObjectList
		result2 error
	}
	ListNoLabelsStub        func(context.Context, types.ObjectType, ...query.Criterion) (types.ObjectList, error)
	listNoLabelsMutex       sync.RWMutex
	listNoLabelsArgsForCall []struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 []query.Criterion
	}
	listNoLabelsReturns struct {
		result1 types.ObjectList
		result2 error
	}
	listNoLabelsReturnsOnCall map[int]struct {
		result1 types.ObjectList
		result2 error
	}
	OpenStub        func(*storage.Settings) error
	openMutex       sync.RWMutex
	openArgsForCall []struct {
		arg1 *storage.Settings
	}
	openReturns struct {
		result1 error
	}
	openReturnsOnCall map[int]struct {
		result1 error
	}
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
	QueryForListStub        func(context.Context, types.ObjectType, storage.NamedQuery, map[string]interface{}) (types.ObjectList, error)
	queryForListMutex       sync.RWMutex
	queryForListArgsForCall []struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 storage.NamedQuery
		arg4 map[string]interface{}
	}
	queryForListReturns struct {
		result1 types.ObjectList
		result2 error
	}
	queryForListReturnsOnCall map[int]struct {
		result1 types.ObjectList
		result2 error
	}
	UpdateStub        func(context.Context, types.Object, types.LabelChanges, ...query.Criterion) (types.Object, error)
	updateMutex       sync.RWMutex
	updateArgsForCall []struct {
		arg1 context.Context
		arg2 types.Object
		arg3 types.LabelChanges
		arg4 []query.Criterion
	}
	updateReturns struct {
		result1 types.Object
		result2 error
	}
	updateReturnsOnCall map[int]struct {
		result1 types.Object
		result2 error
	}
	UpdateLabelsStub        func(context.Context, types.ObjectType, string, types.LabelChanges, ...query.Criterion) error
	updateLabelsMutex       sync.RWMutex
	updateLabelsArgsForCall []struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 string
		arg4 types.LabelChanges
		arg5 []query.Criterion
	}
	updateLabelsReturns struct {
		result1 error
	}
	updateLabelsReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeStorage) Close() error {
	fake.closeMutex.Lock()
	ret, specificReturn := fake.closeReturnsOnCall[len(fake.closeArgsForCall)]
	fake.closeArgsForCall = append(fake.closeArgsForCall, struct {
	}{})
	fake.recordInvocation("Close", []interface{}{})
	fake.closeMutex.Unlock()
	if fake.CloseStub != nil {
		return fake.CloseStub()
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.closeReturns
	return fakeReturns.result1
}

func (fake *FakeStorage) CloseCallCount() int {
	fake.closeMutex.RLock()
	defer fake.closeMutex.RUnlock()
	return len(fake.closeArgsForCall)
}

func (fake *FakeStorage) CloseCalls(stub func() error) {
	fake.closeMutex.Lock()
	defer fake.closeMutex.Unlock()
	fake.CloseStub = stub
}

func (fake *FakeStorage) CloseReturns(result1 error) {
	fake.closeMutex.Lock()
	defer fake.closeMutex.Unlock()
	fake.CloseStub = nil
	fake.closeReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeStorage) CloseReturnsOnCall(i int, result1 error) {
	fake.closeMutex.Lock()
	defer fake.closeMutex.Unlock()
	fake.CloseStub = nil
	if fake.closeReturnsOnCall == nil {
		fake.closeReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.closeReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeStorage) Count(arg1 context.Context, arg2 types.ObjectType, arg3 ...query.Criterion) (int, error) {
	fake.countMutex.Lock()
	ret, specificReturn := fake.countReturnsOnCall[len(fake.countArgsForCall)]
	fake.countArgsForCall = append(fake.countArgsForCall, struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 []query.Criterion
	}{arg1, arg2, arg3})
	fake.recordInvocation("Count", []interface{}{arg1, arg2, arg3})
	fake.countMutex.Unlock()
	if fake.CountStub != nil {
		return fake.CountStub(arg1, arg2, arg3...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.countReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeStorage) CountCallCount() int {
	fake.countMutex.RLock()
	defer fake.countMutex.RUnlock()
	return len(fake.countArgsForCall)
}

func (fake *FakeStorage) CountCalls(stub func(context.Context, types.ObjectType, ...query.Criterion) (int, error)) {
	fake.countMutex.Lock()
	defer fake.countMutex.Unlock()
	fake.CountStub = stub
}

func (fake *FakeStorage) CountArgsForCall(i int) (context.Context, types.ObjectType, []query.Criterion) {
	fake.countMutex.RLock()
	defer fake.countMutex.RUnlock()
	argsForCall := fake.countArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeStorage) CountReturns(result1 int, result2 error) {
	fake.countMutex.Lock()
	defer fake.countMutex.Unlock()
	fake.CountStub = nil
	fake.countReturns = struct {
		result1 int
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) CountReturnsOnCall(i int, result1 int, result2 error) {
	fake.countMutex.Lock()
	defer fake.countMutex.Unlock()
	fake.CountStub = nil
	if fake.countReturnsOnCall == nil {
		fake.countReturnsOnCall = make(map[int]struct {
			result1 int
			result2 error
		})
	}
	fake.countReturnsOnCall[i] = struct {
		result1 int
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) Create(arg1 context.Context, arg2 types.Object) (types.Object, error) {
	fake.createMutex.Lock()
	ret, specificReturn := fake.createReturnsOnCall[len(fake.createArgsForCall)]
	fake.createArgsForCall = append(fake.createArgsForCall, struct {
		arg1 context.Context
		arg2 types.Object
	}{arg1, arg2})
	fake.recordInvocation("Create", []interface{}{arg1, arg2})
	fake.createMutex.Unlock()
	if fake.CreateStub != nil {
		return fake.CreateStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.createReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeStorage) CreateCallCount() int {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	return len(fake.createArgsForCall)
}

func (fake *FakeStorage) CreateCalls(stub func(context.Context, types.Object) (types.Object, error)) {
	fake.createMutex.Lock()
	defer fake.createMutex.Unlock()
	fake.CreateStub = stub
}

func (fake *FakeStorage) CreateArgsForCall(i int) (context.Context, types.Object) {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	argsForCall := fake.createArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeStorage) CreateReturns(result1 types.Object, result2 error) {
	fake.createMutex.Lock()
	defer fake.createMutex.Unlock()
	fake.CreateStub = nil
	fake.createReturns = struct {
		result1 types.Object
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) CreateReturnsOnCall(i int, result1 types.Object, result2 error) {
	fake.createMutex.Lock()
	defer fake.createMutex.Unlock()
	fake.CreateStub = nil
	if fake.createReturnsOnCall == nil {
		fake.createReturnsOnCall = make(map[int]struct {
			result1 types.Object
			result2 error
		})
	}
	fake.createReturnsOnCall[i] = struct {
		result1 types.Object
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) Delete(arg1 context.Context, arg2 types.ObjectType, arg3 ...query.Criterion) error {
	fake.deleteMutex.Lock()
	ret, specificReturn := fake.deleteReturnsOnCall[len(fake.deleteArgsForCall)]
	fake.deleteArgsForCall = append(fake.deleteArgsForCall, struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 []query.Criterion
	}{arg1, arg2, arg3})
	fake.recordInvocation("Delete", []interface{}{arg1, arg2, arg3})
	fake.deleteMutex.Unlock()
	if fake.DeleteStub != nil {
		return fake.DeleteStub(arg1, arg2, arg3...)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.deleteReturns
	return fakeReturns.result1
}

func (fake *FakeStorage) DeleteCallCount() int {
	fake.deleteMutex.RLock()
	defer fake.deleteMutex.RUnlock()
	return len(fake.deleteArgsForCall)
}

func (fake *FakeStorage) DeleteCalls(stub func(context.Context, types.ObjectType, ...query.Criterion) error) {
	fake.deleteMutex.Lock()
	defer fake.deleteMutex.Unlock()
	fake.DeleteStub = stub
}

func (fake *FakeStorage) DeleteArgsForCall(i int) (context.Context, types.ObjectType, []query.Criterion) {
	fake.deleteMutex.RLock()
	defer fake.deleteMutex.RUnlock()
	argsForCall := fake.deleteArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeStorage) DeleteReturns(result1 error) {
	fake.deleteMutex.Lock()
	defer fake.deleteMutex.Unlock()
	fake.DeleteStub = nil
	fake.deleteReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeStorage) DeleteReturnsOnCall(i int, result1 error) {
	fake.deleteMutex.Lock()
	defer fake.deleteMutex.Unlock()
	fake.DeleteStub = nil
	if fake.deleteReturnsOnCall == nil {
		fake.deleteReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.deleteReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeStorage) DeleteReturning(arg1 context.Context, arg2 types.ObjectType, arg3 ...query.Criterion) (types.ObjectList, error) {
	fake.deleteReturningMutex.Lock()
	ret, specificReturn := fake.deleteReturningReturnsOnCall[len(fake.deleteReturningArgsForCall)]
	fake.deleteReturningArgsForCall = append(fake.deleteReturningArgsForCall, struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 []query.Criterion
	}{arg1, arg2, arg3})
	fake.recordInvocation("DeleteReturning", []interface{}{arg1, arg2, arg3})
	fake.deleteReturningMutex.Unlock()
	if fake.DeleteReturningStub != nil {
		return fake.DeleteReturningStub(arg1, arg2, arg3...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.deleteReturningReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeStorage) DeleteReturningCallCount() int {
	fake.deleteReturningMutex.RLock()
	defer fake.deleteReturningMutex.RUnlock()
	return len(fake.deleteReturningArgsForCall)
}

func (fake *FakeStorage) DeleteReturningCalls(stub func(context.Context, types.ObjectType, ...query.Criterion) (types.ObjectList, error)) {
	fake.deleteReturningMutex.Lock()
	defer fake.deleteReturningMutex.Unlock()
	fake.DeleteReturningStub = stub
}

func (fake *FakeStorage) DeleteReturningArgsForCall(i int) (context.Context, types.ObjectType, []query.Criterion) {
	fake.deleteReturningMutex.RLock()
	defer fake.deleteReturningMutex.RUnlock()
	argsForCall := fake.deleteReturningArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeStorage) DeleteReturningReturns(result1 types.ObjectList, result2 error) {
	fake.deleteReturningMutex.Lock()
	defer fake.deleteReturningMutex.Unlock()
	fake.DeleteReturningStub = nil
	fake.deleteReturningReturns = struct {
		result1 types.ObjectList
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) DeleteReturningReturnsOnCall(i int, result1 types.ObjectList, result2 error) {
	fake.deleteReturningMutex.Lock()
	defer fake.deleteReturningMutex.Unlock()
	fake.DeleteReturningStub = nil
	if fake.deleteReturningReturnsOnCall == nil {
		fake.deleteReturningReturnsOnCall = make(map[int]struct {
			result1 types.ObjectList
			result2 error
		})
	}
	fake.deleteReturningReturnsOnCall[i] = struct {
		result1 types.ObjectList
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) Get(arg1 context.Context, arg2 types.ObjectType, arg3 ...query.Criterion) (types.Object, error) {
	fake.getMutex.Lock()
	ret, specificReturn := fake.getReturnsOnCall[len(fake.getArgsForCall)]
	fake.getArgsForCall = append(fake.getArgsForCall, struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 []query.Criterion
	}{arg1, arg2, arg3})
	fake.recordInvocation("Get", []interface{}{arg1, arg2, arg3})
	fake.getMutex.Unlock()
	if fake.GetStub != nil {
		return fake.GetStub(arg1, arg2, arg3...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.getReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeStorage) GetCallCount() int {
	fake.getMutex.RLock()
	defer fake.getMutex.RUnlock()
	return len(fake.getArgsForCall)
}

func (fake *FakeStorage) GetCalls(stub func(context.Context, types.ObjectType, ...query.Criterion) (types.Object, error)) {
	fake.getMutex.Lock()
	defer fake.getMutex.Unlock()
	fake.GetStub = stub
}

func (fake *FakeStorage) GetArgsForCall(i int) (context.Context, types.ObjectType, []query.Criterion) {
	fake.getMutex.RLock()
	defer fake.getMutex.RUnlock()
	argsForCall := fake.getArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeStorage) GetReturns(result1 types.Object, result2 error) {
	fake.getMutex.Lock()
	defer fake.getMutex.Unlock()
	fake.GetStub = nil
	fake.getReturns = struct {
		result1 types.Object
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) GetReturnsOnCall(i int, result1 types.Object, result2 error) {
	fake.getMutex.Lock()
	defer fake.getMutex.Unlock()
	fake.GetStub = nil
	if fake.getReturnsOnCall == nil {
		fake.getReturnsOnCall = make(map[int]struct {
			result1 types.Object
			result2 error
		})
	}
	fake.getReturnsOnCall[i] = struct {
		result1 types.Object
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) GetForUpdate(arg1 context.Context, arg2 types.ObjectType, arg3 ...query.Criterion) (types.Object, error) {
	fake.getForUpdateMutex.Lock()
	ret, specificReturn := fake.getForUpdateReturnsOnCall[len(fake.getForUpdateArgsForCall)]
	fake.getForUpdateArgsForCall = append(fake.getForUpdateArgsForCall, struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 []query.Criterion
	}{arg1, arg2, arg3})
	fake.recordInvocation("GetForUpdate", []interface{}{arg1, arg2, arg3})
	fake.getForUpdateMutex.Unlock()
	if fake.GetForUpdateStub != nil {
		return fake.GetForUpdateStub(arg1, arg2, arg3...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.getForUpdateReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeStorage) GetForUpdateCallCount() int {
	fake.getForUpdateMutex.RLock()
	defer fake.getForUpdateMutex.RUnlock()
	return len(fake.getForUpdateArgsForCall)
}

func (fake *FakeStorage) GetForUpdateCalls(stub func(context.Context, types.ObjectType, ...query.Criterion) (types.Object, error)) {
	fake.getForUpdateMutex.Lock()
	defer fake.getForUpdateMutex.Unlock()
	fake.GetForUpdateStub = stub
}

func (fake *FakeStorage) GetForUpdateArgsForCall(i int) (context.Context, types.ObjectType, []query.Criterion) {
	fake.getForUpdateMutex.RLock()
	defer fake.getForUpdateMutex.RUnlock()
	argsForCall := fake.getForUpdateArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeStorage) GetForUpdateReturns(result1 types.Object, result2 error) {
	fake.getForUpdateMutex.Lock()
	defer fake.getForUpdateMutex.Unlock()
	fake.GetForUpdateStub = nil
	fake.getForUpdateReturns = struct {
		result1 types.Object
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) GetForUpdateReturnsOnCall(i int, result1 types.Object, result2 error) {
	fake.getForUpdateMutex.Lock()
	defer fake.getForUpdateMutex.Unlock()
	fake.GetForUpdateStub = nil
	if fake.getForUpdateReturnsOnCall == nil {
		fake.getForUpdateReturnsOnCall = make(map[int]struct {
			result1 types.Object
			result2 error
		})
	}
	fake.getForUpdateReturnsOnCall[i] = struct {
		result1 types.Object
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) InTransaction(arg1 context.Context, arg2 func(ctx context.Context, storage storage.Repository) error) error {
	fake.inTransactionMutex.Lock()
	ret, specificReturn := fake.inTransactionReturnsOnCall[len(fake.inTransactionArgsForCall)]
	fake.inTransactionArgsForCall = append(fake.inTransactionArgsForCall, struct {
		arg1 context.Context
		arg2 func(ctx context.Context, storage storage.Repository) error
	}{arg1, arg2})
	fake.recordInvocation("InTransaction", []interface{}{arg1, arg2})
	fake.inTransactionMutex.Unlock()
	if fake.InTransactionStub != nil {
		return fake.InTransactionStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.inTransactionReturns
	return fakeReturns.result1
}

func (fake *FakeStorage) InTransactionCallCount() int {
	fake.inTransactionMutex.RLock()
	defer fake.inTransactionMutex.RUnlock()
	return len(fake.inTransactionArgsForCall)
}

func (fake *FakeStorage) InTransactionCalls(stub func(context.Context, func(ctx context.Context, storage storage.Repository) error) error) {
	fake.inTransactionMutex.Lock()
	defer fake.inTransactionMutex.Unlock()
	fake.InTransactionStub = stub
}

func (fake *FakeStorage) InTransactionArgsForCall(i int) (context.Context, func(ctx context.Context, storage storage.Repository) error) {
	fake.inTransactionMutex.RLock()
	defer fake.inTransactionMutex.RUnlock()
	argsForCall := fake.inTransactionArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeStorage) InTransactionReturns(result1 error) {
	fake.inTransactionMutex.Lock()
	defer fake.inTransactionMutex.Unlock()
	fake.InTransactionStub = nil
	fake.inTransactionReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeStorage) InTransactionReturnsOnCall(i int, result1 error) {
	fake.inTransactionMutex.Lock()
	defer fake.inTransactionMutex.Unlock()
	fake.InTransactionStub = nil
	if fake.inTransactionReturnsOnCall == nil {
		fake.inTransactionReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.inTransactionReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeStorage) Introduce(arg1 storage.Entity) {
	fake.introduceMutex.Lock()
	fake.introduceArgsForCall = append(fake.introduceArgsForCall, struct {
		arg1 storage.Entity
	}{arg1})
	fake.recordInvocation("Introduce", []interface{}{arg1})
	fake.introduceMutex.Unlock()
	if fake.IntroduceStub != nil {
		fake.IntroduceStub(arg1)
	}
}

func (fake *FakeStorage) IntroduceCallCount() int {
	fake.introduceMutex.RLock()
	defer fake.introduceMutex.RUnlock()
	return len(fake.introduceArgsForCall)
}

func (fake *FakeStorage) IntroduceCalls(stub func(storage.Entity)) {
	fake.introduceMutex.Lock()
	defer fake.introduceMutex.Unlock()
	fake.IntroduceStub = stub
}

func (fake *FakeStorage) IntroduceArgsForCall(i int) storage.Entity {
	fake.introduceMutex.RLock()
	defer fake.introduceMutex.RUnlock()
	argsForCall := fake.introduceArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeStorage) List(arg1 context.Context, arg2 types.ObjectType, arg3 ...query.Criterion) (types.ObjectList, error) {
	fake.listMutex.Lock()
	ret, specificReturn := fake.listReturnsOnCall[len(fake.listArgsForCall)]
	fake.listArgsForCall = append(fake.listArgsForCall, struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 []query.Criterion
	}{arg1, arg2, arg3})
	fake.recordInvocation("List", []interface{}{arg1, arg2, arg3})
	fake.listMutex.Unlock()
	if fake.ListStub != nil {
		return fake.ListStub(arg1, arg2, arg3...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.listReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeStorage) ListCallCount() int {
	fake.listMutex.RLock()
	defer fake.listMutex.RUnlock()
	return len(fake.listArgsForCall)
}

func (fake *FakeStorage) ListCalls(stub func(context.Context, types.ObjectType, ...query.Criterion) (types.ObjectList, error)) {
	fake.listMutex.Lock()
	defer fake.listMutex.Unlock()
	fake.ListStub = stub
}

func (fake *FakeStorage) ListArgsForCall(i int) (context.Context, types.ObjectType, []query.Criterion) {
	fake.listMutex.RLock()
	defer fake.listMutex.RUnlock()
	argsForCall := fake.listArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeStorage) ListReturns(result1 types.ObjectList, result2 error) {
	fake.listMutex.Lock()
	defer fake.listMutex.Unlock()
	fake.ListStub = nil
	fake.listReturns = struct {
		result1 types.ObjectList
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) ListReturnsOnCall(i int, result1 types.ObjectList, result2 error) {
	fake.listMutex.Lock()
	defer fake.listMutex.Unlock()
	fake.ListStub = nil
	if fake.listReturnsOnCall == nil {
		fake.listReturnsOnCall = make(map[int]struct {
			result1 types.ObjectList
			result2 error
		})
	}
	fake.listReturnsOnCall[i] = struct {
		result1 types.ObjectList
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) ListNoLabels(arg1 context.Context, arg2 types.ObjectType, arg3 ...query.Criterion) (types.ObjectList, error) {
	fake.listNoLabelsMutex.Lock()
	ret, specificReturn := fake.listNoLabelsReturnsOnCall[len(fake.listNoLabelsArgsForCall)]
	fake.listNoLabelsArgsForCall = append(fake.listNoLabelsArgsForCall, struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 []query.Criterion
	}{arg1, arg2, arg3})
	fake.recordInvocation("ListNoLabels", []interface{}{arg1, arg2, arg3})
	fake.listNoLabelsMutex.Unlock()
	if fake.ListNoLabelsStub != nil {
		return fake.ListNoLabelsStub(arg1, arg2, arg3...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.listNoLabelsReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeStorage) ListNoLabelsCallCount() int {
	fake.listNoLabelsMutex.RLock()
	defer fake.listNoLabelsMutex.RUnlock()
	return len(fake.listNoLabelsArgsForCall)
}

func (fake *FakeStorage) ListNoLabelsCalls(stub func(context.Context, types.ObjectType, ...query.Criterion) (types.ObjectList, error)) {
	fake.listNoLabelsMutex.Lock()
	defer fake.listNoLabelsMutex.Unlock()
	fake.ListNoLabelsStub = stub
}

func (fake *FakeStorage) ListNoLabelsArgsForCall(i int) (context.Context, types.ObjectType, []query.Criterion) {
	fake.listNoLabelsMutex.RLock()
	defer fake.listNoLabelsMutex.RUnlock()
	argsForCall := fake.listNoLabelsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeStorage) ListNoLabelsReturns(result1 types.ObjectList, result2 error) {
	fake.listNoLabelsMutex.Lock()
	defer fake.listNoLabelsMutex.Unlock()
	fake.ListNoLabelsStub = nil
	fake.listNoLabelsReturns = struct {
		result1 types.ObjectList
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) ListNoLabelsReturnsOnCall(i int, result1 types.ObjectList, result2 error) {
	fake.listNoLabelsMutex.Lock()
	defer fake.listNoLabelsMutex.Unlock()
	fake.ListNoLabelsStub = nil
	if fake.listNoLabelsReturnsOnCall == nil {
		fake.listNoLabelsReturnsOnCall = make(map[int]struct {
			result1 types.ObjectList
			result2 error
		})
	}
	fake.listNoLabelsReturnsOnCall[i] = struct {
		result1 types.ObjectList
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) Open(arg1 *storage.Settings) error {
	fake.openMutex.Lock()
	ret, specificReturn := fake.openReturnsOnCall[len(fake.openArgsForCall)]
	fake.openArgsForCall = append(fake.openArgsForCall, struct {
		arg1 *storage.Settings
	}{arg1})
	fake.recordInvocation("Open", []interface{}{arg1})
	fake.openMutex.Unlock()
	if fake.OpenStub != nil {
		return fake.OpenStub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.openReturns
	return fakeReturns.result1
}

func (fake *FakeStorage) OpenCallCount() int {
	fake.openMutex.RLock()
	defer fake.openMutex.RUnlock()
	return len(fake.openArgsForCall)
}

func (fake *FakeStorage) OpenCalls(stub func(*storage.Settings) error) {
	fake.openMutex.Lock()
	defer fake.openMutex.Unlock()
	fake.OpenStub = stub
}

func (fake *FakeStorage) OpenArgsForCall(i int) *storage.Settings {
	fake.openMutex.RLock()
	defer fake.openMutex.RUnlock()
	argsForCall := fake.openArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeStorage) OpenReturns(result1 error) {
	fake.openMutex.Lock()
	defer fake.openMutex.Unlock()
	fake.OpenStub = nil
	fake.openReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeStorage) OpenReturnsOnCall(i int, result1 error) {
	fake.openMutex.Lock()
	defer fake.openMutex.Unlock()
	fake.OpenStub = nil
	if fake.openReturnsOnCall == nil {
		fake.openReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.openReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeStorage) PingContext(arg1 context.Context) error {
	fake.pingContextMutex.Lock()
	ret, specificReturn := fake.pingContextReturnsOnCall[len(fake.pingContextArgsForCall)]
	fake.pingContextArgsForCall = append(fake.pingContextArgsForCall, struct {
		arg1 context.Context
	}{arg1})
	fake.recordInvocation("PingContext", []interface{}{arg1})
	fake.pingContextMutex.Unlock()
	if fake.PingContextStub != nil {
		return fake.PingContextStub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.pingContextReturns
	return fakeReturns.result1
}

func (fake *FakeStorage) PingContextCallCount() int {
	fake.pingContextMutex.RLock()
	defer fake.pingContextMutex.RUnlock()
	return len(fake.pingContextArgsForCall)
}

func (fake *FakeStorage) PingContextCalls(stub func(context.Context) error) {
	fake.pingContextMutex.Lock()
	defer fake.pingContextMutex.Unlock()
	fake.PingContextStub = stub
}

func (fake *FakeStorage) PingContextArgsForCall(i int) context.Context {
	fake.pingContextMutex.RLock()
	defer fake.pingContextMutex.RUnlock()
	argsForCall := fake.pingContextArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeStorage) PingContextReturns(result1 error) {
	fake.pingContextMutex.Lock()
	defer fake.pingContextMutex.Unlock()
	fake.PingContextStub = nil
	fake.pingContextReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeStorage) PingContextReturnsOnCall(i int, result1 error) {
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

func (fake *FakeStorage) QueryForList(arg1 context.Context, arg2 types.ObjectType, arg3 storage.NamedQuery, arg4 map[string]interface{}) (types.ObjectList, error) {
	fake.queryForListMutex.Lock()
	ret, specificReturn := fake.queryForListReturnsOnCall[len(fake.queryForListArgsForCall)]
	fake.queryForListArgsForCall = append(fake.queryForListArgsForCall, struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 storage.NamedQuery
		arg4 map[string]interface{}
	}{arg1, arg2, arg3, arg4})
	fake.recordInvocation("QueryForList", []interface{}{arg1, arg2, arg3, arg4})
	fake.queryForListMutex.Unlock()
	if fake.QueryForListStub != nil {
		return fake.QueryForListStub(arg1, arg2, arg3, arg4)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.queryForListReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeStorage) QueryForListCallCount() int {
	fake.queryForListMutex.RLock()
	defer fake.queryForListMutex.RUnlock()
	return len(fake.queryForListArgsForCall)
}

func (fake *FakeStorage) QueryForListCalls(stub func(context.Context, types.ObjectType, storage.NamedQuery, map[string]interface{}) (types.ObjectList, error)) {
	fake.queryForListMutex.Lock()
	defer fake.queryForListMutex.Unlock()
	fake.QueryForListStub = stub
}

func (fake *FakeStorage) QueryForListArgsForCall(i int) (context.Context, types.ObjectType, storage.NamedQuery, map[string]interface{}) {
	fake.queryForListMutex.RLock()
	defer fake.queryForListMutex.RUnlock()
	argsForCall := fake.queryForListArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4
}

func (fake *FakeStorage) QueryForListReturns(result1 types.ObjectList, result2 error) {
	fake.queryForListMutex.Lock()
	defer fake.queryForListMutex.Unlock()
	fake.QueryForListStub = nil
	fake.queryForListReturns = struct {
		result1 types.ObjectList
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) QueryForListReturnsOnCall(i int, result1 types.ObjectList, result2 error) {
	fake.queryForListMutex.Lock()
	defer fake.queryForListMutex.Unlock()
	fake.QueryForListStub = nil
	if fake.queryForListReturnsOnCall == nil {
		fake.queryForListReturnsOnCall = make(map[int]struct {
			result1 types.ObjectList
			result2 error
		})
	}
	fake.queryForListReturnsOnCall[i] = struct {
		result1 types.ObjectList
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) Update(arg1 context.Context, arg2 types.Object, arg3 types.LabelChanges, arg4 ...query.Criterion) (types.Object, error) {
	fake.updateMutex.Lock()
	ret, specificReturn := fake.updateReturnsOnCall[len(fake.updateArgsForCall)]
	fake.updateArgsForCall = append(fake.updateArgsForCall, struct {
		arg1 context.Context
		arg2 types.Object
		arg3 types.LabelChanges
		arg4 []query.Criterion
	}{arg1, arg2, arg3, arg4})
	fake.recordInvocation("Update", []interface{}{arg1, arg2, arg3, arg4})
	fake.updateMutex.Unlock()
	if fake.UpdateStub != nil {
		return fake.UpdateStub(arg1, arg2, arg3, arg4...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.updateReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeStorage) UpdateCallCount() int {
	fake.updateMutex.RLock()
	defer fake.updateMutex.RUnlock()
	return len(fake.updateArgsForCall)
}

func (fake *FakeStorage) UpdateCalls(stub func(context.Context, types.Object, types.LabelChanges, ...query.Criterion) (types.Object, error)) {
	fake.updateMutex.Lock()
	defer fake.updateMutex.Unlock()
	fake.UpdateStub = stub
}

func (fake *FakeStorage) UpdateArgsForCall(i int) (context.Context, types.Object, types.LabelChanges, []query.Criterion) {
	fake.updateMutex.RLock()
	defer fake.updateMutex.RUnlock()
	argsForCall := fake.updateArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4
}

func (fake *FakeStorage) UpdateReturns(result1 types.Object, result2 error) {
	fake.updateMutex.Lock()
	defer fake.updateMutex.Unlock()
	fake.UpdateStub = nil
	fake.updateReturns = struct {
		result1 types.Object
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) UpdateReturnsOnCall(i int, result1 types.Object, result2 error) {
	fake.updateMutex.Lock()
	defer fake.updateMutex.Unlock()
	fake.UpdateStub = nil
	if fake.updateReturnsOnCall == nil {
		fake.updateReturnsOnCall = make(map[int]struct {
			result1 types.Object
			result2 error
		})
	}
	fake.updateReturnsOnCall[i] = struct {
		result1 types.Object
		result2 error
	}{result1, result2}
}

func (fake *FakeStorage) UpdateLabels(arg1 context.Context, arg2 types.ObjectType, arg3 string, arg4 types.LabelChanges, arg5 ...query.Criterion) error {
	fake.updateLabelsMutex.Lock()
	ret, specificReturn := fake.updateLabelsReturnsOnCall[len(fake.updateLabelsArgsForCall)]
	fake.updateLabelsArgsForCall = append(fake.updateLabelsArgsForCall, struct {
		arg1 context.Context
		arg2 types.ObjectType
		arg3 string
		arg4 types.LabelChanges
		arg5 []query.Criterion
	}{arg1, arg2, arg3, arg4, arg5})
	fake.recordInvocation("UpdateLabels", []interface{}{arg1, arg2, arg3, arg4, arg5})
	fake.updateLabelsMutex.Unlock()
	if fake.UpdateLabelsStub != nil {
		return fake.UpdateLabelsStub(arg1, arg2, arg3, arg4, arg5...)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.updateLabelsReturns
	return fakeReturns.result1
}

func (fake *FakeStorage) UpdateLabelsCallCount() int {
	fake.updateLabelsMutex.RLock()
	defer fake.updateLabelsMutex.RUnlock()
	return len(fake.updateLabelsArgsForCall)
}

func (fake *FakeStorage) UpdateLabelsCalls(stub func(context.Context, types.ObjectType, string, types.LabelChanges, ...query.Criterion) error) {
	fake.updateLabelsMutex.Lock()
	defer fake.updateLabelsMutex.Unlock()
	fake.UpdateLabelsStub = stub
}

func (fake *FakeStorage) UpdateLabelsArgsForCall(i int) (context.Context, types.ObjectType, string, types.LabelChanges, []query.Criterion) {
	fake.updateLabelsMutex.RLock()
	defer fake.updateLabelsMutex.RUnlock()
	argsForCall := fake.updateLabelsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4, argsForCall.arg5
}

func (fake *FakeStorage) UpdateLabelsReturns(result1 error) {
	fake.updateLabelsMutex.Lock()
	defer fake.updateLabelsMutex.Unlock()
	fake.UpdateLabelsStub = nil
	fake.updateLabelsReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeStorage) UpdateLabelsReturnsOnCall(i int, result1 error) {
	fake.updateLabelsMutex.Lock()
	defer fake.updateLabelsMutex.Unlock()
	fake.UpdateLabelsStub = nil
	if fake.updateLabelsReturnsOnCall == nil {
		fake.updateLabelsReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.updateLabelsReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeStorage) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.closeMutex.RLock()
	defer fake.closeMutex.RUnlock()
	fake.countMutex.RLock()
	defer fake.countMutex.RUnlock()
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	fake.deleteMutex.RLock()
	defer fake.deleteMutex.RUnlock()
	fake.deleteReturningMutex.RLock()
	defer fake.deleteReturningMutex.RUnlock()
	fake.getMutex.RLock()
	defer fake.getMutex.RUnlock()
	fake.getForUpdateMutex.RLock()
	defer fake.getForUpdateMutex.RUnlock()
	fake.inTransactionMutex.RLock()
	defer fake.inTransactionMutex.RUnlock()
	fake.introduceMutex.RLock()
	defer fake.introduceMutex.RUnlock()
	fake.listMutex.RLock()
	defer fake.listMutex.RUnlock()
	fake.listNoLabelsMutex.RLock()
	defer fake.listNoLabelsMutex.RUnlock()
	fake.openMutex.RLock()
	defer fake.openMutex.RUnlock()
	fake.pingContextMutex.RLock()
	defer fake.pingContextMutex.RUnlock()
	fake.queryForListMutex.RLock()
	defer fake.queryForListMutex.RUnlock()
	fake.updateMutex.RLock()
	defer fake.updateMutex.RUnlock()
	fake.updateLabelsMutex.RLock()
	defer fake.updateLabelsMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeStorage) recordInvocation(key string, args []interface{}) {
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

var _ storage.Storage = new(FakeStorage)
