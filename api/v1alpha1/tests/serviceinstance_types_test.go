package tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sm-operator/sapcp-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Service Instance Type Test", func() {
	var instance *v1alpha1.ServiceInstance
	BeforeEach(func() {
		instance = getInstance()
		conditions := instance.GetConditions()
		readyCondition := metav1.Condition{Type: v1alpha1.ConditionReady, Status: metav1.ConditionTrue, Reason: "reason", Message: "message"}
		meta.SetStatusCondition(&conditions, readyCondition)
		instance.SetConditions(conditions)
	})

	It("should clone correctly", func() {
		clonedInstance := instance.DeepClone()
		Expect(instance).To(Equal(clonedInstance))

		clonedStatus := instance.Status.DeepCopy()
		Expect(&instance.Status).To(Equal(clonedStatus))

		clonedSpec := instance.Spec.DeepCopy()
		Expect(&instance.Spec).To(Equal(clonedSpec))

		list := &v1alpha1.ServiceInstanceList{Items: []v1alpha1.ServiceInstance{*instance}}
		clonedList := list.DeepCopy()
		Expect(list).To(Equal(clonedList))
	})

	It("should clone into correctly", func() {
		clonedInstance := &v1alpha1.ServiceInstance{}
		instance.DeepCopyInto(clonedInstance)
		Expect(instance).To(Equal(clonedInstance))

		clonedStatus := &v1alpha1.ServiceInstanceStatus{}
		instance.Status.DeepCopyInto(clonedStatus)
		Expect(&instance.Status).To(Equal(clonedStatus))

		clonedSpec := &v1alpha1.ServiceInstanceSpec{}
		instance.Spec.DeepCopyInto(clonedSpec)
		Expect(&instance.Spec).To(Equal(clonedSpec))

		list := &v1alpha1.ServiceInstanceList{Items: []v1alpha1.ServiceInstance{*instance}}
		clonedList := &v1alpha1.ServiceInstanceList{}
		list.DeepCopyInto(clonedList)
		Expect(list).To(Equal(clonedList))
	})
})
