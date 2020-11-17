package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sm-operator/sapcp-operator/api/v1alpha1"
	smclientTypes "github.com/sm-operator/sapcp-operator/internal/smclient/types"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// +kubebuilder:docs-gen:collapse=Imports

const (
	timeout       = time.Second * 10
	interval      = time.Millisecond * 250
	fakeBindingID = "fake-binding-id"
)

var _ = Describe("ServiceBinding controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	instanceName := "test-instance"
	namespace := "test-namespace"
	bindingName := "test-binding"

	var createdInstance *v1alpha1.ServiceInstance
	var createdBinding *v1alpha1.ServiceBinding

	newBinding := func(name, namespace string) *v1alpha1.ServiceBinding {
		return &v1alpha1.ServiceBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "services.cloud.sap.com/v1alpha1",
				Kind:       "ServiceBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
	}

	createBindingWithoutAssertionsAndWait := func(ctx context.Context, name string, namespace string, instanceName string, externalName string, wait bool) (*v1alpha1.ServiceBinding, error) {
		binding := newBinding(name, namespace)
		binding.Spec.ServiceInstanceName = instanceName
		binding.Spec.ExternalName = externalName

		if err := k8sClient.Create(ctx, binding); err != nil {
			return nil, err
		}

		bindingLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
		createdBinding = &v1alpha1.ServiceBinding{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, bindingLookupKey, createdBinding)
			if err != nil {
				return false
			}

			if wait {
				return isReady(createdBinding) || isFailed(createdBinding)
			} else {
				return len(createdBinding.Status.Conditions) > 0
			}
		}, timeout, interval).Should(BeTrue())
		return createdBinding, nil
	}

	createBindingWithoutAssertions := func(ctx context.Context, name string, namespace string, instanceName string, externalName string) (*v1alpha1.ServiceBinding, error) {
		return createBindingWithoutAssertionsAndWait(ctx, name, namespace, instanceName, externalName, true)
	}

	createBindingWithError := func(ctx context.Context, name, namespace, instanceName, externalName, failureMessage string) *v1alpha1.ServiceBinding {
		createdBinding, err := createBindingWithoutAssertions(ctx, name, namespace, instanceName, externalName)
		if err != nil {
			Expect(err.Error()).To(ContainSubstring(failureMessage))
		} else {
			Expect(createdBinding.Status.BindingID).To(BeEmpty())
			Expect(createdBinding.Status.SecretName).To(BeEmpty())
			Expect(len(createdBinding.Status.Conditions)).To(Equal(2))
			Expect(createdBinding.Status.Conditions[1].Status).To(Equal(v1alpha1.ConditionTrue))
			Expect(createdBinding.Status.Conditions[1].Message).To(ContainSubstring(failureMessage))
		}

		return nil
	}

	createBinding := func(ctx context.Context, name, namespace, instanceName, externalName string) *v1alpha1.ServiceBinding {
		createdBinding, err := createBindingWithoutAssertions(ctx, name, namespace, instanceName, externalName)
		Expect(err).ToNot(HaveOccurred())

		Expect(createdBinding.Status.InstanceID).ToNot(BeEmpty())
		Expect(createdBinding.Status.BindingID).To(Equal(fakeBindingID))
		Expect(createdBinding.Status.SecretName).To(Not(BeEmpty()))
		Expect(int(createdBinding.Status.ObservedGeneration)).To(Equal(1))

		return createdBinding
	}

	createInstance := func(ctx context.Context, name, namespace, externalName string, wait bool) *v1alpha1.ServiceInstance {
		instance := &v1alpha1.ServiceInstance{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "services.cloud.sap.com/v1alpha1",
				Kind:       "ServiceInstance",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: v1alpha1.ServiceInstanceSpec{
				ExternalName:        externalName,
				ServicePlanID:       "a-plan-id",
				ServicePlanName:     "a-plan-name",
				ServiceOfferingName: "an-offering-name",
			},
		}
		Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

		instanceLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
		createdInstance := &v1alpha1.ServiceInstance{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, instanceLookupKey, createdInstance)
			if err != nil {
				return false
			}
			if wait {
				return isReady(createdInstance)
			} else {
				return len(createdInstance.Status.Conditions) > 0
			}
		}, timeout, interval).Should(BeTrue())
		Expect(createdInstance.Status.InstanceID).ToNot(BeEmpty())

		return createdInstance
	}

	JustBeforeEach(func() {
		createdInstance = createInstance(context.Background(), instanceName, namespace, "", true)
	})

	BeforeEach(func() {
		fakeClient.BindReturns(&smclientTypes.ServiceBinding{ID: fakeBindingID, Credentials: json.RawMessage("{\"secret_key\": \"secret_value\"}")}, "", nil)
	})

	AfterEach(func() {
		if createdBinding != nil {
			Expect(k8sClient.Delete(context.Background(), createdBinding)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: namespace}, createdBinding)
				if err != nil {
					if apierrors.IsNotFound(err) {
						return true
					}
				}

				return false
			}, timeout, interval).Should(BeTrue())

			createdBinding = nil
		}

		Expect(k8sClient.Delete(context.Background(), createdInstance)).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: instanceName, Namespace: namespace}, createdInstance)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return true
				}
			}

			return false
		}, timeout, interval).Should(BeTrue())

		createdInstance = nil
	})

	Context("Create", func() {
		Context("Invalid parameters", func() {
			When("service instance name is not provided", func() {
				It("should fail", func() {
					createBindingWithError(context.Background(), bindingName, namespace, "", "",
						"spec.serviceInstanceName in body should be at least 1 chars long")
				})
			})

			When("referenced service instance does not exist", func() {
				It("should fail", func() {
					createBindingWithError(context.Background(), bindingName, namespace, "no-such-instance", "",
						"Unable to find referenced service instance with k8s name no-such-instance")
				})
			})

			When("referenced service instance exist in another namespace", func() {
				It("should fail", func() {
					createBindingWithError(context.Background(), bindingName, "other-"+namespace, instanceName, "",
						fmt.Sprintf("Unable to find referenced service instance with k8s name %s", instanceName))
				})
			})
		})

		Context("Valid parameters", func() {
			It("Should create binding and store the binding credentials in a secret", func() {
				ctx := context.Background()
				createdBinding = createBinding(ctx, bindingName, namespace, instanceName, "binding-external-name")
				Expect(createdBinding.Spec.ExternalName).To(Equal("binding-external-name"))

				By("Verify binding secret created")
				bindingSecret := &v1.Secret{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: createdBinding.Status.SecretName, Namespace: namespace}, bindingSecret)
				Expect(err).ToNot(HaveOccurred())

				Expect(bindingSecret.Data).ToNot(BeNil())
				Expect(len(bindingSecret.Data["secret_key"])).ToNot(BeNil())

				Expect(string(bindingSecret.Data["secret_key"])).To(Equal("secret_value"))
			})

			When("external name is not provided", func() {
				It("succeeds and uses the k8s name as external name", func() {
					createdBinding = createBinding(context.Background(), bindingName, namespace, instanceName, "")
					Expect(createdBinding.Spec.ExternalName).To(Equal(createdBinding.Name))
				})
			})

			XWhen("referenced service instance is failed", func() {
				It("should retry and succeed once the instance is fixed?", func() {
					// TODO what is the expected behaviour?
				})
			})

			XWhen("referenced service instance is not ready", func() {
				It("should retry and succeed once the instance is ready?", func() {
					// TODO what is the expected behaviour?
				})
			})

			When("bind call to SM returns error", func() {
				errorMessage := "no binding for you"

				BeforeEach(func() {
					fakeClient.BindReturns(nil, "", errors.New(errorMessage))
				})

				It("should fail with the error returned from SM", func() {
					createBindingWithError(context.Background(), bindingName, namespace, instanceName, "existing-name",
						errorMessage)
				})
			})
		})
	})

	Context("Update", func() {
		XWhen("spec is changed", func() {
			It("should fail", func() {
				//TODO
			})
		})
	})

	Context("Delete", func() {
		XWhen("delete in SM succeeds", func() {
			It("should delete the k8s binding and secret", func() {
				//TODO
			})
		})

		XWhen("delete in SM is async", func() {
			When("polling ends with success", func() {
				It("should delete the k8s binding and secret", func() {
					//TODO
				})
			})
			When("polling ends with failure", func() {
				It("should not delete the k8s binding and secret", func() {
					//TODO
				})
			})
		})
	})
})
