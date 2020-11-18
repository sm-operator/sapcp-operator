package v1alpha1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Binding Webhook Suite")
}

var binding *ServiceBinding

var _ = Describe("Service Binding Webhook Test", func() {
	BeforeEach(func() {
		binding = getBinding()
	})
	Context("Defaulter", func() {
		When("No external name provided", func() {
			BeforeEach(func() {
				binding.Spec.ExternalName = ""
			})
			It("should add default", func() {
				binding.Default()
				Expect(binding.Spec.ExternalName).To(Equal(binding.Name))
			})
		})
	})

	Context("Validator", func() {
		Context("Validate create", func() {
			It("should succeed", func() {
				err := binding.ValidateCreate()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("Validate update", func() {
			var newBinding *ServiceBinding
			BeforeEach(func() {
				newBinding = getBinding()
			})
			When("Spec changed", func() {
				When("Service instance name changed", func() {
					It("should fail", func() {
						newBinding.Spec.ServiceInstanceName = "new-service-instance"
						err := newBinding.ValidateUpdate(binding)
						Expect(err).To(HaveOccurred())
					})
				})

				When("External name changed", func() {
					It("should fail", func() {
						newBinding.Spec.ExternalName = "new-external-instance"
						err := newBinding.ValidateUpdate(binding)
						Expect(err).To(HaveOccurred())
					})
				})

				When("Parameters were changed", func() {
					It("should fail", func() {
						newBinding.Spec.Parameters = &runtime.RawExtension{
							Raw: []byte("params"),
						}
						err := newBinding.ValidateUpdate(binding)
						Expect(err).To(HaveOccurred())
					})
				})
			})

			When("Metadata changed", func() {
				It("should succeed", func() {
					newBinding.Finalizers = append(newBinding.Finalizers, "newFinalizer")
					err := newBinding.ValidateUpdate(binding)
					Expect(err).ToNot(HaveOccurred())
				})
			})

			When("Status changed", func() {
				It("should succeed", func() {
					newBinding.Status.BindingID = "12345"
					err := newBinding.ValidateUpdate(binding)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Context("Validate delete", func() {
			It("should succeed", func() {
				err := binding.ValidateDelete()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})

func getBinding() *ServiceBinding {
	return &ServiceBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "services.cloud.sap.com/v1alpha1",
			Kind:       "ServiceBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service-binding-1",
			Namespace: "namespace-1",
		},
		Spec: ServiceBindingSpec{
			ServiceInstanceName: "service-instance-1",
			ExternalName:        "my-service-binding-1",
		},

		Status: ServiceBindingStatus{},
	}
}
