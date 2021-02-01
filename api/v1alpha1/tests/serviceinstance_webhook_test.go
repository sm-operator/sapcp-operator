package tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sm-operator/sapcp-operator/api/v1alpha1"
)

var _ = Describe("Service Binding Webhook Test", func() {
	var instance *v1alpha1.ServiceInstance
	BeforeEach(func() {
		instance = getInstance()
	})
	Context("Defaulter", func() {
		When("No external name provided", func() {
			BeforeEach(func() {
				instance.Spec.ExternalName = ""
			})
			It("should add default", func() {
				instance.Default()
				Expect(instance.Spec.ExternalName).To(Equal(instance.Name))
			})
		})
	})
})
