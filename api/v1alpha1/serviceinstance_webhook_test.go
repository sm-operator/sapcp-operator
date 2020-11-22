package v1alpha1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Service Binding Webhook Test", func() {
	var instance *ServiceInstance
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

	Context("Validator", func() {
		Context("Validate create", func() {
			It("should succeed", func() {
				err := instance.ValidateCreate()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("Validate update", func() {
			var newInstance *ServiceInstance
			BeforeEach(func() {
				newInstance = getInstance()
			})
			When("Spec changed", func() {
				When("Spec changed", func() {
					It("should succeed", func() {
						newInstance.Spec.ExternalName = "new-service-instance"
						err := newInstance.ValidateUpdate(instance)
						Expect(err).ToNot(HaveOccurred())
					})
				})

			})

			When("Metadata changed", func() {
				It("should succeed", func() {
					newInstance.Finalizers = append(newInstance.Finalizers, "newFinalizer")
					err := newInstance.ValidateUpdate(instance)
					Expect(err).ToNot(HaveOccurred())
				})
			})

			When("Status changed", func() {
				It("should succeed", func() {
					newInstance.Status.InstanceID = "12345"
					err := newInstance.ValidateUpdate(instance)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Context("Validate delete", func() {
			It("should succeed", func() {
				err := instance.ValidateDelete()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
