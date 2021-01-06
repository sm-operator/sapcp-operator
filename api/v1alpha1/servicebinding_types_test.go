package v1alpha1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Service Binding Type Test", func() {
	var binding *ServiceBinding
	BeforeEach(func() {
		binding = getBinding()
		conditions := binding.GetConditions()
		readyCondition := metav1.Condition{Type: ConditionReady, Status: metav1.ConditionTrue, Reason: "reason", Message: "message"}
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
	})

})
