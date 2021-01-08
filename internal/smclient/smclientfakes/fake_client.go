// Code generated by counterfeiter. DO NOT EDIT.
package smclientfakes

import (
	"io"
	"net/http"
	"sync"

	"github.com/sm-operator/sapcp-operator/internal/smclient"
	"github.com/sm-operator/sapcp-operator/internal/smclient/types"
)

type FakeClient struct {
	BindStub        func(*types.ServiceBinding, *smclient.Parameters) (*types.ServiceBinding, string, error)
	bindMutex       sync.RWMutex
	bindArgsForCall []struct {
		arg1 *types.ServiceBinding
		arg2 *smclient.Parameters
	}
	bindReturns struct {
		result1 *types.ServiceBinding
		result2 string
		result3 error
	}
	bindReturnsOnCall map[int]struct {
		result1 *types.ServiceBinding
		result2 string
		result3 error
	}
	CallStub        func(string, string, io.Reader, *smclient.Parameters) (*http.Response, error)
	callMutex       sync.RWMutex
	callArgsForCall []struct {
		arg1 string
		arg2 string
		arg3 io.Reader
		arg4 *smclient.Parameters
	}
	callReturns struct {
		result1 *http.Response
		result2 error
	}
	callReturnsOnCall map[int]struct {
		result1 *http.Response
		result2 error
	}
	DeprovisionStub        func(string, *smclient.Parameters) (string, error)
	deprovisionMutex       sync.RWMutex
	deprovisionArgsForCall []struct {
		arg1 string
		arg2 *smclient.Parameters
	}
	deprovisionReturns struct {
		result1 string
		result2 error
	}
	deprovisionReturnsOnCall map[int]struct {
		result1 string
		result2 error
	}
	GetBindingByIDStub        func(string, *smclient.Parameters) (*types.ServiceBinding, error)
	getBindingByIDMutex       sync.RWMutex
	getBindingByIDArgsForCall []struct {
		arg1 string
		arg2 *smclient.Parameters
	}
	getBindingByIDReturns struct {
		result1 *types.ServiceBinding
		result2 error
	}
	getBindingByIDReturnsOnCall map[int]struct {
		result1 *types.ServiceBinding
		result2 error
	}
	GetInstanceByIDStub        func(string, *smclient.Parameters) (*types.ServiceInstance, error)
	getInstanceByIDMutex       sync.RWMutex
	getInstanceByIDArgsForCall []struct {
		arg1 string
		arg2 *smclient.Parameters
	}
	getInstanceByIDReturns struct {
		result1 *types.ServiceInstance
		result2 error
	}
	getInstanceByIDReturnsOnCall map[int]struct {
		result1 *types.ServiceInstance
		result2 error
	}
	ListBindingsStub        func(*smclient.Parameters) (*types.ServiceBindings, error)
	listBindingsMutex       sync.RWMutex
	listBindingsArgsForCall []struct {
		arg1 *smclient.Parameters
	}
	listBindingsReturns struct {
		result1 *types.ServiceBindings
		result2 error
	}
	listBindingsReturnsOnCall map[int]struct {
		result1 *types.ServiceBindings
		result2 error
	}
	ListInstancesStub        func(*smclient.Parameters) (*types.ServiceInstances, error)
	listInstancesMutex       sync.RWMutex
	listInstancesArgsForCall []struct {
		arg1 *smclient.Parameters
	}
	listInstancesReturns struct {
		result1 *types.ServiceInstances
		result2 error
	}
	listInstancesReturnsOnCall map[int]struct {
		result1 *types.ServiceInstances
		result2 error
	}
	ListOfferingsStub        func(*smclient.Parameters) (*types.ServiceOfferings, error)
	listOfferingsMutex       sync.RWMutex
	listOfferingsArgsForCall []struct {
		arg1 *smclient.Parameters
	}
	listOfferingsReturns struct {
		result1 *types.ServiceOfferings
		result2 error
	}
	listOfferingsReturnsOnCall map[int]struct {
		result1 *types.ServiceOfferings
		result2 error
	}
	ListPlansStub        func(*smclient.Parameters) (*types.ServicePlans, error)
	listPlansMutex       sync.RWMutex
	listPlansArgsForCall []struct {
		arg1 *smclient.Parameters
	}
	listPlansReturns struct {
		result1 *types.ServicePlans
		result2 error
	}
	listPlansReturnsOnCall map[int]struct {
		result1 *types.ServicePlans
		result2 error
	}
	ProvisionStub        func(*types.ServiceInstance, string, string, *smclient.Parameters) (string, string, error)
	provisionMutex       sync.RWMutex
	provisionArgsForCall []struct {
		arg1 *types.ServiceInstance
		arg2 string
		arg3 string
		arg4 *smclient.Parameters
	}
	provisionReturns struct {
		result1 string
		result2 string
		result3 error
	}
	provisionReturnsOnCall map[int]struct {
		result1 string
		result2 string
		result3 error
	}
	StatusStub        func(string, *smclient.Parameters) (*types.Operation, error)
	statusMutex       sync.RWMutex
	statusArgsForCall []struct {
		arg1 string
		arg2 *smclient.Parameters
	}
	statusReturns struct {
		result1 *types.Operation
		result2 error
	}
	statusReturnsOnCall map[int]struct {
		result1 *types.Operation
		result2 error
	}
	UnbindStub        func(string, *smclient.Parameters) (string, error)
	unbindMutex       sync.RWMutex
	unbindArgsForCall []struct {
		arg1 string
		arg2 *smclient.Parameters
	}
	unbindReturns struct {
		result1 string
		result2 error
	}
	unbindReturnsOnCall map[int]struct {
		result1 string
		result2 error
	}
	UpdateInstanceStub        func(string, *types.ServiceInstance, string, string, *smclient.Parameters) (*types.ServiceInstance, string, error)
	updateInstanceMutex       sync.RWMutex
	updateInstanceArgsForCall []struct {
		arg1 string
		arg2 *types.ServiceInstance
		arg3 string
		arg4 string
		arg5 *smclient.Parameters
	}
	updateInstanceReturns struct {
		result1 *types.ServiceInstance
		result2 string
		result3 error
	}
	updateInstanceReturnsOnCall map[int]struct {
		result1 *types.ServiceInstance
		result2 string
		result3 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeClient) Bind(arg1 *types.ServiceBinding, arg2 *smclient.Parameters) (*types.ServiceBinding, string, error) {
	fake.bindMutex.Lock()
	ret, specificReturn := fake.bindReturnsOnCall[len(fake.bindArgsForCall)]
	fake.bindArgsForCall = append(fake.bindArgsForCall, struct {
		arg1 *types.ServiceBinding
		arg2 *smclient.Parameters
	}{arg1, arg2})
	stub := fake.BindStub
	fakeReturns := fake.bindReturns
	fake.recordInvocation("Bind", []interface{}{arg1, arg2})
	fake.bindMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *FakeClient) BindCallCount() int {
	fake.bindMutex.RLock()
	defer fake.bindMutex.RUnlock()
	return len(fake.bindArgsForCall)
}

func (fake *FakeClient) BindCalls(stub func(*types.ServiceBinding, *smclient.Parameters) (*types.ServiceBinding, string, error)) {
	fake.bindMutex.Lock()
	defer fake.bindMutex.Unlock()
	fake.BindStub = stub
}

func (fake *FakeClient) BindArgsForCall(i int) (*types.ServiceBinding, *smclient.Parameters) {
	fake.bindMutex.RLock()
	defer fake.bindMutex.RUnlock()
	argsForCall := fake.bindArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeClient) BindReturns(result1 *types.ServiceBinding, result2 string, result3 error) {
	fake.bindMutex.Lock()
	defer fake.bindMutex.Unlock()
	fake.BindStub = nil
	fake.bindReturns = struct {
		result1 *types.ServiceBinding
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeClient) BindReturnsOnCall(i int, result1 *types.ServiceBinding, result2 string, result3 error) {
	fake.bindMutex.Lock()
	defer fake.bindMutex.Unlock()
	fake.BindStub = nil
	if fake.bindReturnsOnCall == nil {
		fake.bindReturnsOnCall = make(map[int]struct {
			result1 *types.ServiceBinding
			result2 string
			result3 error
		})
	}
	fake.bindReturnsOnCall[i] = struct {
		result1 *types.ServiceBinding
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeClient) Call(arg1 string, arg2 string, arg3 io.Reader, arg4 *smclient.Parameters) (*http.Response, error) {
	fake.callMutex.Lock()
	ret, specificReturn := fake.callReturnsOnCall[len(fake.callArgsForCall)]
	fake.callArgsForCall = append(fake.callArgsForCall, struct {
		arg1 string
		arg2 string
		arg3 io.Reader
		arg4 *smclient.Parameters
	}{arg1, arg2, arg3, arg4})
	stub := fake.CallStub
	fakeReturns := fake.callReturns
	fake.recordInvocation("Call", []interface{}{arg1, arg2, arg3, arg4})
	fake.callMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3, arg4)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) CallCallCount() int {
	fake.callMutex.RLock()
	defer fake.callMutex.RUnlock()
	return len(fake.callArgsForCall)
}

func (fake *FakeClient) CallCalls(stub func(string, string, io.Reader, *smclient.Parameters) (*http.Response, error)) {
	fake.callMutex.Lock()
	defer fake.callMutex.Unlock()
	fake.CallStub = stub
}

func (fake *FakeClient) CallArgsForCall(i int) (string, string, io.Reader, *smclient.Parameters) {
	fake.callMutex.RLock()
	defer fake.callMutex.RUnlock()
	argsForCall := fake.callArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4
}

func (fake *FakeClient) CallReturns(result1 *http.Response, result2 error) {
	fake.callMutex.Lock()
	defer fake.callMutex.Unlock()
	fake.CallStub = nil
	fake.callReturns = struct {
		result1 *http.Response
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) CallReturnsOnCall(i int, result1 *http.Response, result2 error) {
	fake.callMutex.Lock()
	defer fake.callMutex.Unlock()
	fake.CallStub = nil
	if fake.callReturnsOnCall == nil {
		fake.callReturnsOnCall = make(map[int]struct {
			result1 *http.Response
			result2 error
		})
	}
	fake.callReturnsOnCall[i] = struct {
		result1 *http.Response
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) Deprovision(arg1 string, arg2 *smclient.Parameters) (string, error) {
	fake.deprovisionMutex.Lock()
	ret, specificReturn := fake.deprovisionReturnsOnCall[len(fake.deprovisionArgsForCall)]
	fake.deprovisionArgsForCall = append(fake.deprovisionArgsForCall, struct {
		arg1 string
		arg2 *smclient.Parameters
	}{arg1, arg2})
	stub := fake.DeprovisionStub
	fakeReturns := fake.deprovisionReturns
	fake.recordInvocation("Deprovision", []interface{}{arg1, arg2})
	fake.deprovisionMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) DeprovisionCallCount() int {
	fake.deprovisionMutex.RLock()
	defer fake.deprovisionMutex.RUnlock()
	return len(fake.deprovisionArgsForCall)
}

func (fake *FakeClient) DeprovisionCalls(stub func(string, *smclient.Parameters) (string, error)) {
	fake.deprovisionMutex.Lock()
	defer fake.deprovisionMutex.Unlock()
	fake.DeprovisionStub = stub
}

func (fake *FakeClient) DeprovisionArgsForCall(i int) (string, *smclient.Parameters) {
	fake.deprovisionMutex.RLock()
	defer fake.deprovisionMutex.RUnlock()
	argsForCall := fake.deprovisionArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeClient) DeprovisionReturns(result1 string, result2 error) {
	fake.deprovisionMutex.Lock()
	defer fake.deprovisionMutex.Unlock()
	fake.DeprovisionStub = nil
	fake.deprovisionReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) DeprovisionReturnsOnCall(i int, result1 string, result2 error) {
	fake.deprovisionMutex.Lock()
	defer fake.deprovisionMutex.Unlock()
	fake.DeprovisionStub = nil
	if fake.deprovisionReturnsOnCall == nil {
		fake.deprovisionReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.deprovisionReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) GetBindingByID(arg1 string, arg2 *smclient.Parameters) (*types.ServiceBinding, error) {
	fake.getBindingByIDMutex.Lock()
	ret, specificReturn := fake.getBindingByIDReturnsOnCall[len(fake.getBindingByIDArgsForCall)]
	fake.getBindingByIDArgsForCall = append(fake.getBindingByIDArgsForCall, struct {
		arg1 string
		arg2 *smclient.Parameters
	}{arg1, arg2})
	stub := fake.GetBindingByIDStub
	fakeReturns := fake.getBindingByIDReturns
	fake.recordInvocation("GetBindingByID", []interface{}{arg1, arg2})
	fake.getBindingByIDMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) GetBindingByIDCallCount() int {
	fake.getBindingByIDMutex.RLock()
	defer fake.getBindingByIDMutex.RUnlock()
	return len(fake.getBindingByIDArgsForCall)
}

func (fake *FakeClient) GetBindingByIDCalls(stub func(string, *smclient.Parameters) (*types.ServiceBinding, error)) {
	fake.getBindingByIDMutex.Lock()
	defer fake.getBindingByIDMutex.Unlock()
	fake.GetBindingByIDStub = stub
}

func (fake *FakeClient) GetBindingByIDArgsForCall(i int) (string, *smclient.Parameters) {
	fake.getBindingByIDMutex.RLock()
	defer fake.getBindingByIDMutex.RUnlock()
	argsForCall := fake.getBindingByIDArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeClient) GetBindingByIDReturns(result1 *types.ServiceBinding, result2 error) {
	fake.getBindingByIDMutex.Lock()
	defer fake.getBindingByIDMutex.Unlock()
	fake.GetBindingByIDStub = nil
	fake.getBindingByIDReturns = struct {
		result1 *types.ServiceBinding
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) GetBindingByIDReturnsOnCall(i int, result1 *types.ServiceBinding, result2 error) {
	fake.getBindingByIDMutex.Lock()
	defer fake.getBindingByIDMutex.Unlock()
	fake.GetBindingByIDStub = nil
	if fake.getBindingByIDReturnsOnCall == nil {
		fake.getBindingByIDReturnsOnCall = make(map[int]struct {
			result1 *types.ServiceBinding
			result2 error
		})
	}
	fake.getBindingByIDReturnsOnCall[i] = struct {
		result1 *types.ServiceBinding
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) GetInstanceByID(arg1 string, arg2 *smclient.Parameters) (*types.ServiceInstance, error) {
	fake.getInstanceByIDMutex.Lock()
	ret, specificReturn := fake.getInstanceByIDReturnsOnCall[len(fake.getInstanceByIDArgsForCall)]
	fake.getInstanceByIDArgsForCall = append(fake.getInstanceByIDArgsForCall, struct {
		arg1 string
		arg2 *smclient.Parameters
	}{arg1, arg2})
	stub := fake.GetInstanceByIDStub
	fakeReturns := fake.getInstanceByIDReturns
	fake.recordInvocation("GetInstanceByID", []interface{}{arg1, arg2})
	fake.getInstanceByIDMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) GetInstanceByIDCallCount() int {
	fake.getInstanceByIDMutex.RLock()
	defer fake.getInstanceByIDMutex.RUnlock()
	return len(fake.getInstanceByIDArgsForCall)
}

func (fake *FakeClient) GetInstanceByIDCalls(stub func(string, *smclient.Parameters) (*types.ServiceInstance, error)) {
	fake.getInstanceByIDMutex.Lock()
	defer fake.getInstanceByIDMutex.Unlock()
	fake.GetInstanceByIDStub = stub
}

func (fake *FakeClient) GetInstanceByIDArgsForCall(i int) (string, *smclient.Parameters) {
	fake.getInstanceByIDMutex.RLock()
	defer fake.getInstanceByIDMutex.RUnlock()
	argsForCall := fake.getInstanceByIDArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeClient) GetInstanceByIDReturns(result1 *types.ServiceInstance, result2 error) {
	fake.getInstanceByIDMutex.Lock()
	defer fake.getInstanceByIDMutex.Unlock()
	fake.GetInstanceByIDStub = nil
	fake.getInstanceByIDReturns = struct {
		result1 *types.ServiceInstance
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) GetInstanceByIDReturnsOnCall(i int, result1 *types.ServiceInstance, result2 error) {
	fake.getInstanceByIDMutex.Lock()
	defer fake.getInstanceByIDMutex.Unlock()
	fake.GetInstanceByIDStub = nil
	if fake.getInstanceByIDReturnsOnCall == nil {
		fake.getInstanceByIDReturnsOnCall = make(map[int]struct {
			result1 *types.ServiceInstance
			result2 error
		})
	}
	fake.getInstanceByIDReturnsOnCall[i] = struct {
		result1 *types.ServiceInstance
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) ListBindings(arg1 *smclient.Parameters) (*types.ServiceBindings, error) {
	fake.listBindingsMutex.Lock()
	ret, specificReturn := fake.listBindingsReturnsOnCall[len(fake.listBindingsArgsForCall)]
	fake.listBindingsArgsForCall = append(fake.listBindingsArgsForCall, struct {
		arg1 *smclient.Parameters
	}{arg1})
	stub := fake.ListBindingsStub
	fakeReturns := fake.listBindingsReturns
	fake.recordInvocation("ListBindings", []interface{}{arg1})
	fake.listBindingsMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) ListBindingsCallCount() int {
	fake.listBindingsMutex.RLock()
	defer fake.listBindingsMutex.RUnlock()
	return len(fake.listBindingsArgsForCall)
}

func (fake *FakeClient) ListBindingsCalls(stub func(*smclient.Parameters) (*types.ServiceBindings, error)) {
	fake.listBindingsMutex.Lock()
	defer fake.listBindingsMutex.Unlock()
	fake.ListBindingsStub = stub
}

func (fake *FakeClient) ListBindingsArgsForCall(i int) *smclient.Parameters {
	fake.listBindingsMutex.RLock()
	defer fake.listBindingsMutex.RUnlock()
	argsForCall := fake.listBindingsArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeClient) ListBindingsReturns(result1 *types.ServiceBindings, result2 error) {
	fake.listBindingsMutex.Lock()
	defer fake.listBindingsMutex.Unlock()
	fake.ListBindingsStub = nil
	fake.listBindingsReturns = struct {
		result1 *types.ServiceBindings
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) ListBindingsReturnsOnCall(i int, result1 *types.ServiceBindings, result2 error) {
	fake.listBindingsMutex.Lock()
	defer fake.listBindingsMutex.Unlock()
	fake.ListBindingsStub = nil
	if fake.listBindingsReturnsOnCall == nil {
		fake.listBindingsReturnsOnCall = make(map[int]struct {
			result1 *types.ServiceBindings
			result2 error
		})
	}
	fake.listBindingsReturnsOnCall[i] = struct {
		result1 *types.ServiceBindings
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) ListInstances(arg1 *smclient.Parameters) (*types.ServiceInstances, error) {
	fake.listInstancesMutex.Lock()
	ret, specificReturn := fake.listInstancesReturnsOnCall[len(fake.listInstancesArgsForCall)]
	fake.listInstancesArgsForCall = append(fake.listInstancesArgsForCall, struct {
		arg1 *smclient.Parameters
	}{arg1})
	stub := fake.ListInstancesStub
	fakeReturns := fake.listInstancesReturns
	fake.recordInvocation("ListInstances", []interface{}{arg1})
	fake.listInstancesMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) ListInstancesCallCount() int {
	fake.listInstancesMutex.RLock()
	defer fake.listInstancesMutex.RUnlock()
	return len(fake.listInstancesArgsForCall)
}

func (fake *FakeClient) ListInstancesCalls(stub func(*smclient.Parameters) (*types.ServiceInstances, error)) {
	fake.listInstancesMutex.Lock()
	defer fake.listInstancesMutex.Unlock()
	fake.ListInstancesStub = stub
}

func (fake *FakeClient) ListInstancesArgsForCall(i int) *smclient.Parameters {
	fake.listInstancesMutex.RLock()
	defer fake.listInstancesMutex.RUnlock()
	argsForCall := fake.listInstancesArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeClient) ListInstancesReturns(result1 *types.ServiceInstances, result2 error) {
	fake.listInstancesMutex.Lock()
	defer fake.listInstancesMutex.Unlock()
	fake.ListInstancesStub = nil
	fake.listInstancesReturns = struct {
		result1 *types.ServiceInstances
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) ListInstancesReturnsOnCall(i int, result1 *types.ServiceInstances, result2 error) {
	fake.listInstancesMutex.Lock()
	defer fake.listInstancesMutex.Unlock()
	fake.ListInstancesStub = nil
	if fake.listInstancesReturnsOnCall == nil {
		fake.listInstancesReturnsOnCall = make(map[int]struct {
			result1 *types.ServiceInstances
			result2 error
		})
	}
	fake.listInstancesReturnsOnCall[i] = struct {
		result1 *types.ServiceInstances
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) ListOfferings(arg1 *smclient.Parameters) (*types.ServiceOfferings, error) {
	fake.listOfferingsMutex.Lock()
	ret, specificReturn := fake.listOfferingsReturnsOnCall[len(fake.listOfferingsArgsForCall)]
	fake.listOfferingsArgsForCall = append(fake.listOfferingsArgsForCall, struct {
		arg1 *smclient.Parameters
	}{arg1})
	stub := fake.ListOfferingsStub
	fakeReturns := fake.listOfferingsReturns
	fake.recordInvocation("ListOfferings", []interface{}{arg1})
	fake.listOfferingsMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) ListOfferingsCallCount() int {
	fake.listOfferingsMutex.RLock()
	defer fake.listOfferingsMutex.RUnlock()
	return len(fake.listOfferingsArgsForCall)
}

func (fake *FakeClient) ListOfferingsCalls(stub func(*smclient.Parameters) (*types.ServiceOfferings, error)) {
	fake.listOfferingsMutex.Lock()
	defer fake.listOfferingsMutex.Unlock()
	fake.ListOfferingsStub = stub
}

func (fake *FakeClient) ListOfferingsArgsForCall(i int) *smclient.Parameters {
	fake.listOfferingsMutex.RLock()
	defer fake.listOfferingsMutex.RUnlock()
	argsForCall := fake.listOfferingsArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeClient) ListOfferingsReturns(result1 *types.ServiceOfferings, result2 error) {
	fake.listOfferingsMutex.Lock()
	defer fake.listOfferingsMutex.Unlock()
	fake.ListOfferingsStub = nil
	fake.listOfferingsReturns = struct {
		result1 *types.ServiceOfferings
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) ListOfferingsReturnsOnCall(i int, result1 *types.ServiceOfferings, result2 error) {
	fake.listOfferingsMutex.Lock()
	defer fake.listOfferingsMutex.Unlock()
	fake.ListOfferingsStub = nil
	if fake.listOfferingsReturnsOnCall == nil {
		fake.listOfferingsReturnsOnCall = make(map[int]struct {
			result1 *types.ServiceOfferings
			result2 error
		})
	}
	fake.listOfferingsReturnsOnCall[i] = struct {
		result1 *types.ServiceOfferings
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) ListPlans(arg1 *smclient.Parameters) (*types.ServicePlans, error) {
	fake.listPlansMutex.Lock()
	ret, specificReturn := fake.listPlansReturnsOnCall[len(fake.listPlansArgsForCall)]
	fake.listPlansArgsForCall = append(fake.listPlansArgsForCall, struct {
		arg1 *smclient.Parameters
	}{arg1})
	stub := fake.ListPlansStub
	fakeReturns := fake.listPlansReturns
	fake.recordInvocation("ListPlans", []interface{}{arg1})
	fake.listPlansMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) ListPlansCallCount() int {
	fake.listPlansMutex.RLock()
	defer fake.listPlansMutex.RUnlock()
	return len(fake.listPlansArgsForCall)
}

func (fake *FakeClient) ListPlansCalls(stub func(*smclient.Parameters) (*types.ServicePlans, error)) {
	fake.listPlansMutex.Lock()
	defer fake.listPlansMutex.Unlock()
	fake.ListPlansStub = stub
}

func (fake *FakeClient) ListPlansArgsForCall(i int) *smclient.Parameters {
	fake.listPlansMutex.RLock()
	defer fake.listPlansMutex.RUnlock()
	argsForCall := fake.listPlansArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeClient) ListPlansReturns(result1 *types.ServicePlans, result2 error) {
	fake.listPlansMutex.Lock()
	defer fake.listPlansMutex.Unlock()
	fake.ListPlansStub = nil
	fake.listPlansReturns = struct {
		result1 *types.ServicePlans
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) ListPlansReturnsOnCall(i int, result1 *types.ServicePlans, result2 error) {
	fake.listPlansMutex.Lock()
	defer fake.listPlansMutex.Unlock()
	fake.ListPlansStub = nil
	if fake.listPlansReturnsOnCall == nil {
		fake.listPlansReturnsOnCall = make(map[int]struct {
			result1 *types.ServicePlans
			result2 error
		})
	}
	fake.listPlansReturnsOnCall[i] = struct {
		result1 *types.ServicePlans
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) Provision(arg1 *types.ServiceInstance, arg2 string, arg3 string, arg4 *smclient.Parameters) (string, string, error) {
	fake.provisionMutex.Lock()
	ret, specificReturn := fake.provisionReturnsOnCall[len(fake.provisionArgsForCall)]
	fake.provisionArgsForCall = append(fake.provisionArgsForCall, struct {
		arg1 *types.ServiceInstance
		arg2 string
		arg3 string
		arg4 *smclient.Parameters
	}{arg1, arg2, arg3, arg4})
	stub := fake.ProvisionStub
	fakeReturns := fake.provisionReturns
	fake.recordInvocation("Provision", []interface{}{arg1, arg2, arg3, arg4})
	fake.provisionMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3, arg4)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *FakeClient) ProvisionCallCount() int {
	fake.provisionMutex.RLock()
	defer fake.provisionMutex.RUnlock()
	return len(fake.provisionArgsForCall)
}

func (fake *FakeClient) ProvisionCalls(stub func(*types.ServiceInstance, string, string, *smclient.Parameters) (string, string, error)) {
	fake.provisionMutex.Lock()
	defer fake.provisionMutex.Unlock()
	fake.ProvisionStub = stub
}

func (fake *FakeClient) ProvisionArgsForCall(i int) (*types.ServiceInstance, string, string, *smclient.Parameters) {
	fake.provisionMutex.RLock()
	defer fake.provisionMutex.RUnlock()
	argsForCall := fake.provisionArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4
}

func (fake *FakeClient) ProvisionReturns(result1 string, result2 string, result3 error) {
	fake.provisionMutex.Lock()
	defer fake.provisionMutex.Unlock()
	fake.ProvisionStub = nil
	fake.provisionReturns = struct {
		result1 string
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeClient) ProvisionReturnsOnCall(i int, result1 string, result2 string, result3 error) {
	fake.provisionMutex.Lock()
	defer fake.provisionMutex.Unlock()
	fake.ProvisionStub = nil
	if fake.provisionReturnsOnCall == nil {
		fake.provisionReturnsOnCall = make(map[int]struct {
			result1 string
			result2 string
			result3 error
		})
	}
	fake.provisionReturnsOnCall[i] = struct {
		result1 string
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeClient) Status(arg1 string, arg2 *smclient.Parameters) (*types.Operation, error) {
	fake.statusMutex.Lock()
	ret, specificReturn := fake.statusReturnsOnCall[len(fake.statusArgsForCall)]
	fake.statusArgsForCall = append(fake.statusArgsForCall, struct {
		arg1 string
		arg2 *smclient.Parameters
	}{arg1, arg2})
	stub := fake.StatusStub
	fakeReturns := fake.statusReturns
	fake.recordInvocation("Status", []interface{}{arg1, arg2})
	fake.statusMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) StatusCallCount() int {
	fake.statusMutex.RLock()
	defer fake.statusMutex.RUnlock()
	return len(fake.statusArgsForCall)
}

func (fake *FakeClient) StatusCalls(stub func(string, *smclient.Parameters) (*types.Operation, error)) {
	fake.statusMutex.Lock()
	defer fake.statusMutex.Unlock()
	fake.StatusStub = stub
}

func (fake *FakeClient) StatusArgsForCall(i int) (string, *smclient.Parameters) {
	fake.statusMutex.RLock()
	defer fake.statusMutex.RUnlock()
	argsForCall := fake.statusArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeClient) StatusReturns(result1 *types.Operation, result2 error) {
	fake.statusMutex.Lock()
	defer fake.statusMutex.Unlock()
	fake.StatusStub = nil
	fake.statusReturns = struct {
		result1 *types.Operation
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) StatusReturnsOnCall(i int, result1 *types.Operation, result2 error) {
	fake.statusMutex.Lock()
	defer fake.statusMutex.Unlock()
	fake.StatusStub = nil
	if fake.statusReturnsOnCall == nil {
		fake.statusReturnsOnCall = make(map[int]struct {
			result1 *types.Operation
			result2 error
		})
	}
	fake.statusReturnsOnCall[i] = struct {
		result1 *types.Operation
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) Unbind(arg1 string, arg2 *smclient.Parameters) (string, error) {
	fake.unbindMutex.Lock()
	ret, specificReturn := fake.unbindReturnsOnCall[len(fake.unbindArgsForCall)]
	fake.unbindArgsForCall = append(fake.unbindArgsForCall, struct {
		arg1 string
		arg2 *smclient.Parameters
	}{arg1, arg2})
	stub := fake.UnbindStub
	fakeReturns := fake.unbindReturns
	fake.recordInvocation("Unbind", []interface{}{arg1, arg2})
	fake.unbindMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) UnbindCallCount() int {
	fake.unbindMutex.RLock()
	defer fake.unbindMutex.RUnlock()
	return len(fake.unbindArgsForCall)
}

func (fake *FakeClient) UnbindCalls(stub func(string, *smclient.Parameters) (string, error)) {
	fake.unbindMutex.Lock()
	defer fake.unbindMutex.Unlock()
	fake.UnbindStub = stub
}

func (fake *FakeClient) UnbindArgsForCall(i int) (string, *smclient.Parameters) {
	fake.unbindMutex.RLock()
	defer fake.unbindMutex.RUnlock()
	argsForCall := fake.unbindArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeClient) UnbindReturns(result1 string, result2 error) {
	fake.unbindMutex.Lock()
	defer fake.unbindMutex.Unlock()
	fake.UnbindStub = nil
	fake.unbindReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) UnbindReturnsOnCall(i int, result1 string, result2 error) {
	fake.unbindMutex.Lock()
	defer fake.unbindMutex.Unlock()
	fake.UnbindStub = nil
	if fake.unbindReturnsOnCall == nil {
		fake.unbindReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.unbindReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) UpdateInstance(arg1 string, arg2 *types.ServiceInstance, arg3 string, arg4 string, arg5 *smclient.Parameters) (*types.ServiceInstance, string, error) {
	fake.updateInstanceMutex.Lock()
	ret, specificReturn := fake.updateInstanceReturnsOnCall[len(fake.updateInstanceArgsForCall)]
	fake.updateInstanceArgsForCall = append(fake.updateInstanceArgsForCall, struct {
		arg1 string
		arg2 *types.ServiceInstance
		arg3 string
		arg4 string
		arg5 *smclient.Parameters
	}{arg1, arg2, arg3, arg4, arg5})
	stub := fake.UpdateInstanceStub
	fakeReturns := fake.updateInstanceReturns
	fake.recordInvocation("UpdateInstance", []interface{}{arg1, arg2, arg3, arg4, arg5})
	fake.updateInstanceMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3, arg4, arg5)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *FakeClient) UpdateInstanceCallCount() int {
	fake.updateInstanceMutex.RLock()
	defer fake.updateInstanceMutex.RUnlock()
	return len(fake.updateInstanceArgsForCall)
}

func (fake *FakeClient) UpdateInstanceCalls(stub func(string, *types.ServiceInstance, string, string, *smclient.Parameters) (*types.ServiceInstance, string, error)) {
	fake.updateInstanceMutex.Lock()
	defer fake.updateInstanceMutex.Unlock()
	fake.UpdateInstanceStub = stub
}

func (fake *FakeClient) UpdateInstanceArgsForCall(i int) (string, *types.ServiceInstance, string, string, *smclient.Parameters) {
	fake.updateInstanceMutex.RLock()
	defer fake.updateInstanceMutex.RUnlock()
	argsForCall := fake.updateInstanceArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4, argsForCall.arg5
}

func (fake *FakeClient) UpdateInstanceReturns(result1 *types.ServiceInstance, result2 string, result3 error) {
	fake.updateInstanceMutex.Lock()
	defer fake.updateInstanceMutex.Unlock()
	fake.UpdateInstanceStub = nil
	fake.updateInstanceReturns = struct {
		result1 *types.ServiceInstance
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeClient) UpdateInstanceReturnsOnCall(i int, result1 *types.ServiceInstance, result2 string, result3 error) {
	fake.updateInstanceMutex.Lock()
	defer fake.updateInstanceMutex.Unlock()
	fake.UpdateInstanceStub = nil
	if fake.updateInstanceReturnsOnCall == nil {
		fake.updateInstanceReturnsOnCall = make(map[int]struct {
			result1 *types.ServiceInstance
			result2 string
			result3 error
		})
	}
	fake.updateInstanceReturnsOnCall[i] = struct {
		result1 *types.ServiceInstance
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeClient) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.bindMutex.RLock()
	defer fake.bindMutex.RUnlock()
	fake.callMutex.RLock()
	defer fake.callMutex.RUnlock()
	fake.deprovisionMutex.RLock()
	defer fake.deprovisionMutex.RUnlock()
	fake.getBindingByIDMutex.RLock()
	defer fake.getBindingByIDMutex.RUnlock()
	fake.getInstanceByIDMutex.RLock()
	defer fake.getInstanceByIDMutex.RUnlock()
	fake.listBindingsMutex.RLock()
	defer fake.listBindingsMutex.RUnlock()
	fake.listInstancesMutex.RLock()
	defer fake.listInstancesMutex.RUnlock()
	fake.listOfferingsMutex.RLock()
	defer fake.listOfferingsMutex.RUnlock()
	fake.listPlansMutex.RLock()
	defer fake.listPlansMutex.RUnlock()
	fake.provisionMutex.RLock()
	defer fake.provisionMutex.RUnlock()
	fake.statusMutex.RLock()
	defer fake.statusMutex.RUnlock()
	fake.unbindMutex.RLock()
	defer fake.unbindMutex.RUnlock()
	fake.updateInstanceMutex.RLock()
	defer fake.updateInstanceMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeClient) recordInvocation(key string, args []interface{}) {
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

var _ smclient.Client = new(FakeClient)
