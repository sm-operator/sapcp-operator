package tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sm-operator/sapcp-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SAPCP Resource Suite")
}

func getBinding() *v1alpha1.ServiceBinding {
	return &v1alpha1.ServiceBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "services.cloud.sap.com/v1alpha1",
			Kind:       "ServiceBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service-binding-1",
			Namespace: "namespace-1",
		},
		Spec: v1alpha1.ServiceBindingSpec{
			ServiceInstanceName: "service-instance-1",
			ExternalName:        "my-service-binding-1",
			Parameters:          &runtime.RawExtension{Raw: []byte(`{"key":"val"}`)},
		},

		Status: v1alpha1.ServiceBindingStatus{},
	}
}

func getInstance() *v1alpha1.ServiceInstance {
	return &v1alpha1.ServiceInstance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "services.cloud.sap.com/v1alpha1",
			Kind:       "ServiceInstance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service-instance-1",
			Namespace: "namespace-1",
		},
		Spec: v1alpha1.ServiceInstanceSpec{
			ServiceOfferingName: "service-offering-1",
			ServicePlanName:     "service-plan-name-1",
			ServicePlanID:       "service-plan-id-1",
			ExternalName:        "my-service-instance-1",
			Parameters:          &runtime.RawExtension{Raw: []byte(`{"key":"val"}`)},
		},

		Status: v1alpha1.ServiceInstanceStatus{},
	}
}
