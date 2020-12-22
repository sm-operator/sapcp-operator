package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	smTypes "github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sm-operator/sapcp-operator/api/v1alpha1"
	"github.com/sm-operator/sapcp-operator/internal/smclient"
	"github.com/sm-operator/sapcp-operator/internal/smclient/smclientfakes"
	smclientTypes "github.com/sm-operator/sapcp-operator/internal/smclient/types"
	"github.com/sm-operator/sapcp-operator/internal/util"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"time"
)

// +kubebuilder:docs-gen:collapse=Imports

const (
	fakeBindingID = "fake-binding-id"
)

var _ = Describe("ServiceBinding controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	instanceName := "test-instance"
	namespace := "test-namespace"
	bindingName := "test-binding-" + uuid.New().String()

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
		binding.Spec.Parameters = &runtime.RawExtension{
			Raw: []byte(`{"key": "value"}`),
		}

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
		fakeClient = &smclientfakes.FakeClient{}
		fakeClient.ProvisionReturns("12345678", "", nil)
		fakeClient.BindReturns(&smclientTypes.ServiceBinding{ID: fakeBindingID, Credentials: json.RawMessage("{\"secret_key\": \"secret_value\"}")}, "", nil)
	})

	AfterEach(func() {
		if createdBinding != nil {
			secretName := createdBinding.Status.SecretName
			fakeClient.UnbindReturns("", nil)
			k8sClient.Delete(context.Background(), createdBinding)
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: namespace}, createdBinding)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
			if len(secretName) > 0 {
				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: namespace}, &v1.Secret{})
					return apierrors.IsNotFound(err)
				}, timeout, interval).Should(BeTrue())
			}

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
			Context("Sync", func() {
				It("Should create binding and store the binding credentials in a secret", func() {
					ctx := context.Background()
					createdBinding = createBinding(ctx, bindingName, namespace, instanceName, "binding-external-name")
					Expect(createdBinding.Spec.ExternalName).To(Equal("binding-external-name"))

					By("Verify binding secret created")
					bindingSecret := getSecret(ctx, createdBinding.Status.SecretName, createdBinding.Namespace)
					validateSecretData(bindingSecret, "secret_key", "secret_value")
				})
				When("bind call to SM returns error", func() {
					errorMessage := "no binding for you"

					BeforeEach(func() {
						fakeClient.BindReturns(nil, "", errors.New(errorMessage))
					})

					It("should fail with the error returned from SM", func() {
						createBindingWithError(context.Background(), bindingName, namespace, instanceName, "binding-external-name",
							errorMessage)
					})
				})
			})

			Context("Async", func() {
				JustBeforeEach(func() {
					fakeClient.BindReturns(
						nil,
						fmt.Sprintf("/v1/service_bindings/%s/operations/an-operation-id", fakeBindingID),
						nil)
				})

				When("bind polling returns success", func() {
					JustBeforeEach(func() {
						fakeClient.StatusReturns(&smclientTypes.Operation{ResourceID: fakeBindingID, State: string(smTypes.SUCCEEDED)}, nil)
						fakeClient.GetBindingByIDReturns(&smclientTypes.ServiceBinding{ID: fakeBindingID, Credentials: json.RawMessage("{\"secret_key\": \"secret_value\"}")}, nil)
					})

					It("Should create binding and store the binding credentials in a secret", func() {
						ctx := context.Background()
						createdBinding = createBinding(ctx, bindingName, namespace, instanceName, "")
					})
				})

				When("bind polling returns error different from 404", func() {
					errorMessage := "no binding for you"

					JustBeforeEach(func() {
						fakeClient.StatusReturns(&smclientTypes.Operation{
							Type:        string(smTypes.DELETE),
							State:       string(smTypes.FAILED),
							Description: errorMessage,
						}, nil)
					})

					It("should fail with the error returned from SM", func() {
						createBindingWithError(context.Background(), bindingName, namespace, instanceName, "existing-name",
							errorMessage)
					})
				})
				When("bind polling returns 404", func() {
					JustBeforeEach(func() {
						fakeClient.GetBindingByIDReturns(&smclientTypes.ServiceBinding{ID: fakeBindingID, LastOperation: &smTypes.Operation{State: smTypes.SUCCEEDED, Type: smTypes.CREATE}}, nil)
						fakeClient.StatusReturns(nil, &smclient.ServiceManagerError{StatusCode: http.StatusNotFound})
						fakeClient.BindReturns(nil, "/v1/service_bindings/id/operations/1234", nil)
					})
					It("should not fail", func() {
						createdBinding, err := createBindingWithoutAssertions(context.Background(), bindingName, namespace, instanceName, "")
						Expect(err).ToNot(HaveOccurred())
						Eventually(func() bool {
							err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: namespace}, createdBinding)
							if err != nil || len(createdBinding.Status.Conditions) != 1 || createdBinding.Status.Conditions[0].Reason != Created {
								return false
							}
							return true
						}, timeout, interval).Should(BeTrue())
					})
				})
			})

			When("external name is not provided", func() {
				It("succeeds and uses the k8s name as external name", func() {
					createdBinding = createBinding(context.Background(), bindingName, namespace, instanceName, "")
					Expect(createdBinding.Spec.ExternalName).To(Equal(createdBinding.Name))
				})
			})

			When("referenced service instance is failed", func() {
				JustBeforeEach(func() {
					setFailureConditions(smTypes.CREATE, "Failed to create instance (test)", createdInstance)
					err := k8sClient.Status().Update(context.Background(), createdInstance)
					Expect(err).ToNot(HaveOccurred())
				})
				It("should retry and succeed once the instance is ready", func() {
					// verify create fail with appropriate message
					createBindingWithError(context.Background(), bindingName, namespace, instanceName, "binding-external-name",
						"is not usable")

					// verify creation is retired and succeeds after instance is ready
					setSuccessConditions(smTypes.CREATE, createdInstance)
					err := k8sClient.Status().Update(context.Background(), createdInstance)
					Expect(err).ToNot(HaveOccurred())

					Eventually(func() bool {
						err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: namespace}, createdBinding)
						return err == nil && isReady(createdBinding)
					}, syncPeriod+time.Second, interval).Should(BeTrue())
				})
			})

			When("referenced service instance is not ready", func() {
				JustBeforeEach(func() {
					setInProgressCondition(smTypes.CREATE, "", createdInstance)
					err := k8sClient.Status().Update(context.Background(), createdInstance)
					Expect(err).ToNot(HaveOccurred())
				})
				It("should retry and succeed once the instance is ready", func() {
					var err error
					createdBinding, err = createBindingWithoutAssertionsAndWait(context.Background(), bindingName, namespace, instanceName, "binding-external-name", false)
					Expect(err).ToNot(HaveOccurred())
					Expect(isInProgress(createdBinding)).To(BeTrue())

					// verify creation is retired and succeeds after instance is ready
					setSuccessConditions(smTypes.CREATE, createdInstance)
					err = k8sClient.Status().Update(context.Background(), createdInstance)
					Expect(err).ToNot(HaveOccurred())

					Eventually(func() bool {
						err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: namespace}, createdBinding)
						return err == nil && isReady(createdBinding)
					}, pollInterval+time.Second, interval).Should(BeTrue())
				})
			})
		})

		Context("Recovery", func() {
			type recoveryTestCase struct {
				lastOpType  smTypes.OperationCategory
				lastOpState smTypes.OperationState
			}
			executeTestCase := func(testCase recoveryTestCase) {
				fakeBinding := func(state smTypes.OperationState) *smclientTypes.ServiceBinding {
					return &smclientTypes.ServiceBinding{
						ID:          fakeBindingID,
						Name:        "fake-binding-external-name",
						Credentials: json.RawMessage("{\"secret_key\": \"secret_value\"}"),
						LastOperation: &smTypes.Operation{
							Type:        testCase.lastOpType,
							State:       state,
							Description: "fake-description",
						},
					}
				}

				When("binding exists in SM", func() {
					BeforeEach(func() {
						fakeClient.ListBindingsReturns(
							&smclientTypes.ServiceBindings{
								ServiceBindings: []smclientTypes.ServiceBinding{*fakeBinding(testCase.lastOpState)},
							}, nil)
						fakeClient.StatusReturns(&smclientTypes.Operation{ResourceID: fakeBindingID, State: string(smTypes.IN_PROGRESS)}, nil)
					})
					AfterEach(func() {
						fakeClient.StatusReturns(&smclientTypes.Operation{ResourceID: fakeBindingID, State: string(smTypes.SUCCEEDED)}, nil)
						fakeClient.GetBindingByIDReturns(fakeBinding(smTypes.SUCCEEDED), nil)
					})
					When(fmt.Sprintf("last operation is %s %s", testCase.lastOpType, testCase.lastOpState), func() {
						It("should resync status", func() {
							var err error
							createdBinding, err = createBindingWithoutAssertionsAndWait(context.Background(), bindingName, namespace, instanceName, "fake-binding-external-name", false)
							Expect(err).ToNot(HaveOccurred())
							smCallArgs := fakeClient.ListBindingsArgsForCall(0)
							Expect(smCallArgs.LabelQuery).To(HaveLen(3))
							Expect(smCallArgs.FieldQuery).To(HaveLen(1))
							//TODO verify correct parameters used to find binding in SM are correct

							if testCase.lastOpState == smTypes.FAILED {
								Expect(createdBinding.Status.Conditions).To(HaveLen(2))
								Expect(createdBinding.Status.Conditions[0].Status).To(Equal(v1alpha1.ConditionFalse))
								Expect(createdBinding.Status.Conditions[0].Reason).To(Equal(getConditionReason(testCase.lastOpType, testCase.lastOpState)))

								Expect(createdBinding.Status.Conditions[1].Status).To(Equal(v1alpha1.ConditionTrue))
								Expect(createdBinding.Status.Conditions[1].Reason).To(Equal(getConditionReason(testCase.lastOpType, testCase.lastOpState)))
								Expect(createdBinding.Status.Conditions[1].Message).To(ContainSubstring("fake-description"))
							} else {
								Expect(createdBinding.Status.Conditions).To(HaveLen(1))
								if testCase.lastOpState == smTypes.SUCCEEDED {
									Expect(createdBinding.Status.Conditions[0].Status).To(Equal(v1alpha1.ConditionTrue))
									Expect(createdBinding.Status.Conditions[0].Reason).To(Equal(getConditionReason(testCase.lastOpType, testCase.lastOpState)))

									if testCase.lastOpType == smTypes.CREATE || testCase.lastOpType == smTypes.UPDATE {

										validateSecretData(getSecret(context.Background(), createdBinding.Status.SecretName, createdBinding.Namespace), "secret_key", "secret_value")
									}
								} else {
									Expect(createdBinding.Status.Conditions[0].Status).To(Equal(v1alpha1.ConditionFalse))
									Expect(createdBinding.Status.Conditions[0].Reason).To(Equal(getConditionReason(testCase.lastOpType, testCase.lastOpState)))
								}
							}
						})
					})
				})
			}

			for _, testCase := range []recoveryTestCase{
				{lastOpType: smTypes.CREATE, lastOpState: smTypes.SUCCEEDED},
				{lastOpType: smTypes.CREATE, lastOpState: smTypes.IN_PROGRESS},
				{lastOpType: smTypes.CREATE, lastOpState: smTypes.FAILED},
				{lastOpType: smTypes.DELETE, lastOpState: smTypes.SUCCEEDED},
				{lastOpType: smTypes.DELETE, lastOpState: smTypes.IN_PROGRESS},
				{lastOpType: smTypes.DELETE, lastOpState: smTypes.FAILED},
			} {
				executeTestCase(testCase)
			}
		})
	})

	Context("Update", func() {
		JustBeforeEach(func() {
			createdBinding = createBinding(context.Background(), bindingName, namespace, instanceName, "binding-external-name")
			Expect(isReady(createdBinding)).To(BeTrue())
		})

		When("external name is changed", func() {
			It("should fail", func() {
				createdBinding.Spec.ExternalName = "new-external-name"
				err := k8sClient.Update(context.Background(), createdBinding)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("service binding spec cannot be modified after creation"))
			})
		})

		When("service instance name is changed", func() {
			It("should fail", func() {
				createdBinding.Spec.ServiceInstanceName = "new-instance-name"
				err := k8sClient.Update(context.Background(), createdBinding)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("service binding spec cannot be modified after creation"))
			})
		})

		When("parameters are changed", func() {
			It("should fail", func() {
				createdBinding.Spec.Parameters = &runtime.RawExtension{
					Raw: []byte(`{"new-key": "new-value"}`),
				}
				err := k8sClient.Update(context.Background(), createdBinding)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("service binding spec cannot be modified after creation"))
			})
		})

		XWhen("labels are changed", func() {
			It("should fail", func() {
				// TODO labels
			})
		})
	})

	Context("Delete", func() {
		JustBeforeEach(func() {
			createdBinding = createBinding(context.Background(), bindingName, namespace, instanceName, "binding-external-name")
			Expect(isReady(createdBinding)).To(BeTrue())
		})

		Context("Sync", func() {
			When("delete in SM succeeds", func() {
				JustBeforeEach(func() {
					fakeClient.UnbindReturns("", nil)
				})
				It("should delete the k8s binding and secret", func() {
					secretName := createdBinding.Status.SecretName
					Expect(secretName).ToNot(BeEmpty())
					Expect(k8sClient.Delete(context.Background(), createdBinding)).To(Succeed())
					Eventually(func() bool {
						err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: namespace}, createdBinding)
						if apierrors.IsNotFound(err) {
							err := k8sClient.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: namespace}, &v1.Secret{})
							return apierrors.IsNotFound(err)
						}
						return false
					}, timeout, interval).Should(BeTrue())
				})
			})

			When("delete in SM fails", func() {
				JustBeforeEach(func() {
					fakeClient.UnbindReturns("", fmt.Errorf("some-error"))
				})

				It("should not remove finalizer and keep the secret", func() {
					secretName := createdBinding.Status.SecretName
					Expect(secretName).ToNot(BeEmpty())
					Expect(k8sClient.Delete(context.Background(), createdBinding)).To(Succeed())

					Eventually(func() bool {
						err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: namespace}, createdBinding)
						if err != nil {
							return false
						}
						return len(createdBinding.Status.Conditions) == 2
					}, timeout, interval).Should(BeTrue())

					Expect(createdBinding.Status.Conditions[0].Reason).To(Equal(getConditionReason(smTypes.DELETE, smTypes.FAILED)))
					Expect(createdBinding.Status.Conditions[0].Status).To(Equal(v1alpha1.ConditionFalse))
					Expect(createdBinding.Status.Conditions[1].Reason).To(Equal(getConditionReason(smTypes.DELETE, smTypes.FAILED)))
					Expect(createdBinding.Status.Conditions[1].Status).To(Equal(v1alpha1.ConditionTrue))
					Expect(createdBinding.Status.Conditions[1].Message).To(ContainSubstring("some-error"))

					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: namespace}, &v1.Secret{})
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Context("Async", func() {
			JustBeforeEach(func() {
				fakeClient.UnbindReturns(util.BuildOperationURL("an-operation-id", fakeBindingID, web.ServiceBindingsURL), nil)
			})

			When("polling ends with success", func() {
				JustBeforeEach(func() {
					fakeClient.StatusReturns(&smclientTypes.Operation{ResourceID: fakeBindingID, State: string(smTypes.SUCCEEDED)}, nil)
				})

				It("should delete the k8s binding and secret", func() {
					secretName := createdBinding.Status.SecretName
					Expect(secretName).ToNot(BeEmpty())
					Expect(k8sClient.Delete(context.Background(), createdBinding)).To(Succeed())

					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: namespace}, createdBinding)
					Expect(err).ToNot(HaveOccurred())

					Eventually(func() bool {
						err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: namespace}, createdBinding)
						if apierrors.IsNotFound(err) {
							err = k8sClient.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: namespace}, &v1.Secret{})
							return apierrors.IsNotFound(err)
						}

						return false
					}, timeout, interval).Should(BeTrue())
				})
			})

			When("polling ends with failure", func() {
				errorMessage := "delete-binding-async-error"
				JustBeforeEach(func() {
					fakeClient.StatusReturns(&smclientTypes.Operation{
						Type:        string(smTypes.DELETE),
						State:       string(smTypes.FAILED),
						Description: errorMessage,
					}, nil)
				})
				It("should not delete the k8s binding and secret", func() {
					secretName := createdBinding.Status.SecretName
					Expect(secretName).ToNot(BeEmpty())
					Expect(k8sClient.Delete(context.Background(), createdBinding)).To(Succeed())

					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: namespace}, createdBinding)
					Expect(err).ToNot(HaveOccurred())

					Eventually(func() bool {
						err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: namespace}, createdBinding)
						if err != nil {
							return false
						}
						return len(createdBinding.Status.Conditions) == 2
					}, timeout, interval).Should(BeTrue())

					Expect(createdBinding.Status.Conditions[0].Reason).To(Equal(getConditionReason(smTypes.DELETE, smTypes.FAILED)))
					Expect(createdBinding.Status.Conditions[0].Status).To(Equal(v1alpha1.ConditionFalse))
					Expect(createdBinding.Status.Conditions[1].Reason).To(Equal(getConditionReason(smTypes.DELETE, smTypes.FAILED)))
					Expect(createdBinding.Status.Conditions[1].Status).To(Equal(v1alpha1.ConditionTrue))
					Expect(createdBinding.Status.Conditions[1].Message).To(ContainSubstring(errorMessage))

					err = k8sClient.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: namespace}, &v1.Secret{})
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})
	})

})

func validateSecretData(secret *v1.Secret, expectedKey string, expectedValue string) {
	Expect(secret.Data).ToNot(BeNil())
	Expect(len(secret.Data[expectedKey])).ToNot(BeNil())
	Expect(string(secret.Data[expectedKey])).To(Equal(expectedValue))
}

func getSecret(ctx context.Context, name, namespace string) *v1.Secret {
	secret := &v1.Secret{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, secret)
	Expect(err).ToNot(HaveOccurred())

	return secret
}
