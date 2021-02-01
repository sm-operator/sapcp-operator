package tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sm-operator/sapcp-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Service Binding Type Test", func() {
	var binding *v1alpha1.ServiceBinding
	BeforeEach(func() {
		binding = getBinding()
		conditions := binding.GetConditions()
		readyCondition := metav1.Condition{Type: v1alpha1.ConditionReady, Status: metav1.ConditionTrue, Reason: "reason", Message: "message"}
		meta.SetStatusCondition(&conditions, readyCondition)
		binding.SetConditions(conditions)
	})

	It("should clone correctly", func() {
		clonedBinding := binding.DeepClone()
		Expect(binding).To(Equal(clonedBinding))

		clonedStatus := binding.Status.DeepCopy()
		Expect(&binding.Status).To(Equal(clonedStatus))

		clonedSpec := binding.Spec.DeepCopy()
		Expect(&binding.Spec).To(Equal(clonedSpec))

		list := &v1alpha1.ServiceBindingList{Items: []v1alpha1.ServiceBinding{*binding}}
		clonedList := list.DeepCopy()
		Expect(list).To(Equal(clonedList))
	})

	It("should clone into correctly", func() {
		clonedBinding := &v1alpha1.ServiceBinding{}
		binding.DeepCopyInto(clonedBinding)
		Expect(binding).To(Equal(clonedBinding))

		clonedStatus := &v1alpha1.ServiceBindingStatus{}
		binding.Status.DeepCopyInto(clonedStatus)
		Expect(&binding.Status).To(Equal(clonedStatus))

		clonedSpec := &v1alpha1.ServiceBindingSpec{}
		binding.Spec.DeepCopyInto(clonedSpec)
		Expect(&binding.Spec).To(Equal(clonedSpec))

		list := &v1alpha1.ServiceBindingList{Items: []v1alpha1.ServiceBinding{*binding}}
		clonedList := &v1alpha1.ServiceBindingList{}
		list.DeepCopyInto(clonedList)
		Expect(list).To(Equal(clonedList))
	})

})
