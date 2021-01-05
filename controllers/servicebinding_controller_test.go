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
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
)

// +kubebuilder:docs-gen:collapse=Imports

const (
	fakeBindingID        = "fake-binding-id"
	bindingTestNamespace = "test-namespace"
)

var _ = Describe("ServiceBinding controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.

	var createdInstance *v1alpha1.ServiceInstance
	var createdBinding *v1alpha1.ServiceBinding
	var instanceName string
	var bindingName string

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
		}, timeout*2, interval).Should(BeTrue())
		return createdBinding, nil
	}

	createBindingWithoutAssertions := func(ctx context.Context, name string, namespace string, instanceName string, externalName string) (*v1alpha1.ServiceBinding, error) {
		return createBindingWithoutAssertionsAndWait(ctx, name, namespace, instanceName, externalName, true)
	}

	createBindingWithError := func(ctx context.Context, name, namespace, instanceName, externalName, failureMessage string) {
		createdBinding, err := createBindingWithoutAssertions(ctx, name, namespace, instanceName, externalName)
		if err != nil {
			Expect(err.Error()).To(ContainSubstring(failureMessage))
		} else {
			Expect(createdBinding.Status.SecretName).To(BeEmpty())
			Expect(len(createdBinding.Status.Conditions)).To(Equal(2))
			Expect(createdBinding.Status.Conditions[1].Status).To(Equal(metav1.ConditionTrue))
			Expect(createdBinding.Status.Conditions[1].Message).To(ContainSubstring(failureMessage))
		}
	}

	createBindingWithBlockedError := func(ctx context.Context, name, namespace, instanceName, externalName, failureMessage string) *v1alpha1.ServiceBinding {
		createdBinding, err := createBindingWithoutAssertions(ctx, name, namespace, instanceName, externalName)
		if err != nil {
			Expect(err.Error()).To(ContainSubstring(failureMessage))
		} else {
			Expect(createdBinding.Status.SecretName).To(BeEmpty())
			Expect(len(createdBinding.Status.Conditions)).To(Equal(1))
			Expect(createdBinding.Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(createdBinding.Status.Conditions[0].Message).To(ContainSubstring(failureMessage))
			Expect(createdBinding.Status.Conditions[0].Reason).To(Equal(Blocked))
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
		createdInstance = createInstance(context.Background(), instanceName, bindingTestNamespace, "", true)
	})

	BeforeEach(func() {
		guid := uuid.New().String()
		instanceName = "test-instance-" + guid
		bindingName = "test-binding-" + guid
		fakeClient = &smclientfakes.FakeClient{}
		fakeClient.ProvisionReturns("12345678", "", nil)
		fakeClient.BindReturns(&smclientTypes.ServiceBinding{ID: fakeBindingID, Credentials: json.RawMessage("{\"secret_key\": \"secret_value\"}")}, "", nil)
		fakeClient.ListBindingsReturnsOnCall(0, nil, nil)
		fakeClient.ListBindingsReturnsOnCall(1, &smclientTypes.ServiceBindings{
			ServiceBindings: []smclientTypes.ServiceBinding{
				{
					ID:          fakeBindingID,
					Name:        "fake-binding-external-name",
					Credentials: json.RawMessage("{\"secret_key\": \"secret_value\"}"),
					LastOperation: &smTypes.Operation{
						Type:        smTypes.CREATE,
						State:       smTypes.SUCCEEDED,
						Description: "fake-description",
					},
				},
			},
		}, nil)

		smInstance := &smclientTypes.ServiceInstance{ServiceInstanceBase: smclientTypes.ServiceInstanceBase{ID: fakeInstanceID, Ready: true, LastOperation: &smTypes.Operation{State: smTypes.SUCCEEDED, Type: smTypes.UPDATE}}}
		fakeClient.GetInstanceByIDReturns(smInstance, nil)
	})

	AfterEach(func() {
		if createdBinding != nil {
			secretName := createdBinding.Status.SecretName
			fakeClient.UnbindReturns("", nil)
			_ = k8sClient.Delete(context.Background(), createdBinding)
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: bindingTestNamespace}, createdBinding)
				return apierrors.IsNotFound(err)
			}, timeout*2, interval).Should(BeTrue())
			if len(secretName) > 0 {
				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: bindingTestNamespace}, &v1.Secret{})
					return apierrors.IsNotFound(err)
				}, timeout, interval).Should(BeTrue())
			}

			createdBinding = nil
		}

		Expect(k8sClient.Delete(context.Background(), createdInstance)).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: instanceName, Namespace: bindingTestNamespace}, createdInstance)
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
					createBindingWithError(context.Background(), bindingName, bindingTestNamespace, "", "",
						"spec.serviceInstanceName in body should be at least 1 chars long")
				})
			})

			When("referenced service instance does not exist", func() {
				It("should fail", func() {
					createBindingWithBlockedError(context.Background(), bindingName, bindingTestNamespace, "no-such-instance", "",
						"unable to find service instance")
				})
			})

			When("referenced service instance exist in another namespace", func() {
				var otherNamespace = "other-" + bindingTestNamespace
				BeforeEach(func() {
					nsSpec := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: otherNamespace}}
					err := k8sClient.Create(context.Background(), nsSpec)
					Expect(err).ToNot(HaveOccurred())
				})
				It("should fail", func() {
					createBindingWithBlockedError(context.Background(), bindingName, otherNamespace, instanceName, "",
						fmt.Sprintf("unable to find service instance"))
				})
			})
		})

		Context("Valid parameters", func() {
			Context("Sync", func() {
				It("Should create binding and store the binding credentials in a secret", func() {
					ctx := context.Background()
					createdBinding = createBinding(ctx, bindingName, bindingTestNamespace, instanceName, "binding-external-name")
					Expect(createdBinding.Spec.ExternalName).To(Equal("binding-external-name"))

					By("Verify binding secret created")
					bindingSecret := getSecret(ctx, createdBinding.Status.SecretName, createdBinding.Namespace)
					validateSecretData(bindingSecret, "secret_key", "secret_value")
				})

				When("bind call to SM returns error", func() {
					var errorMessage string

					When("general error occurred", func() {
						errorMessage = "no binding for you"
						BeforeEach(func() {
							fakeClient.BindReturns(nil, "", errors.New(errorMessage))
						})

						It("should fail with the error returned from SM", func() {
							createBindingWithError(context.Background(), bindingName, bindingTestNamespace, instanceName, "binding-external-name",
								errorMessage)
						})
					})

					When("SM returned transient error(429)", func() {
						BeforeEach(func() {
							errorMessage = "too many requests"
							fakeClient.BindReturnsOnCall(0, nil, "", &smclient.ServiceManagerError{
								StatusCode: http.StatusTooManyRequests,
								Message:    errorMessage,
							})
							fakeClient.BindReturnsOnCall(1, &smclientTypes.ServiceBinding{ID: fakeBindingID, Credentials: json.RawMessage("{\"secret_key\": \"secret_value\"}")}, "", nil)
						})

						It("should eventually succeed", func() {
							b, err := createBindingWithoutAssertionsAndWait(context.Background(), bindingName, bindingTestNamespace, instanceName, "binding-external-name", true)
							Expect(err).ToNot(HaveOccurred())
							Expect(isReady(b)).To(BeTrue())
						})
					})

					When("SM returned non transient error(400)", func() {
						BeforeEach(func() {
							errorMessage = "very bad request"
							fakeClient.BindReturnsOnCall(0, nil, "", &smclient.ServiceManagerError{
								StatusCode: http.StatusBadRequest,
								Message:    errorMessage,
							})
						})

						It("should fail", func() {
							createBindingWithError(context.Background(), bindingName, bindingTestNamespace, instanceName, "binding-external-name", errorMessage)
						})
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
						createdBinding = createBinding(ctx, bindingName, bindingTestNamespace, instanceName, "")
					})
				})

				When("bind polling returns FAILED state", func() {
					errorMessage := "no binding for you"

					JustBeforeEach(func() {
						fakeClient.StatusReturns(&smclientTypes.Operation{
							Type:        string(smTypes.CREATE),
							State:       string(smTypes.FAILED),
							Description: errorMessage,
						}, nil)
					})

					It("should fail with the error returned from SM", func() {
						createBindingWithError(context.Background(), bindingName, bindingTestNamespace, instanceName, "existing-name",
							errorMessage)
					})
				})

				When("bind polling returns error", func() {
					JustBeforeEach(func() {
						fakeClient.GetBindingByIDReturns(&smclientTypes.ServiceBinding{ID: fakeBindingID, LastOperation: &smTypes.Operation{State: smTypes.SUCCEEDED, Type: smTypes.CREATE}}, nil)
						fakeClient.StatusReturns(nil, fmt.Errorf("no polling for you"))
						fakeClient.BindReturns(nil, "/v1/service_bindings/id/operations/1234", nil)
					})
					It("should eventually succeed", func() {
						createdBinding, err := createBindingWithoutAssertions(context.Background(), bindingName, bindingTestNamespace, instanceName, "")
						Expect(err).ToNot(HaveOccurred())
						Eventually(func() bool {
							err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: bindingTestNamespace}, createdBinding)
							Expect(err).ToNot(HaveOccurred())
							return isReady(createdBinding)
						}, timeout, interval).Should(BeTrue())
					})
				})
			})

			When("external name is not provided", func() {
				It("succeeds and uses the k8s name as external name", func() {
					createdBinding = createBinding(context.Background(), bindingName, bindingTestNamespace, instanceName, "")
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
					createBindingWithBlockedError(context.Background(), bindingName, bindingTestNamespace, instanceName, "binding-external-name",
						"is not usable")

					// verify creation is retired and succeeds after instance is ready
					setSuccessConditions(smTypes.CREATE, createdInstance)
					err := k8sClient.Status().Update(context.Background(), createdInstance)
					Expect(err).ToNot(HaveOccurred())

					Eventually(func() bool {
						err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: bindingTestNamespace}, createdBinding)
						return err == nil && isReady(createdBinding)
					}, timeout, interval).Should(BeTrue())
				})
			})

			When("referenced service instance is not ready", func() {
				JustBeforeEach(func() {
					fakeClient.StatusReturns(&smclientTypes.Operation{ResourceID: fakeInstanceID, State: string(smTypes.IN_PROGRESS)}, nil)
					setInProgressCondition(smTypes.CREATE, "", createdInstance)
					createdInstance.Status.OperationURL = "/1234"
					createdInstance.Status.OperationType = smTypes.CREATE
					err := k8sClient.Status().Update(context.Background(), createdInstance)
					Expect(err).ToNot(HaveOccurred())
				})

				It("should retry and succeed once the instance is ready", func() {
					var err error

					createdBinding, err = createBindingWithoutAssertionsAndWait(context.Background(), bindingName, bindingTestNamespace, instanceName, "binding-external-name", false)
					Expect(err).ToNot(HaveOccurred())
					Expect(isInProgress(createdBinding)).To(BeTrue())

					// verify creation is retired and succeeds after instance is ready
					setSuccessConditions(smTypes.CREATE, createdInstance)
					createdInstance.Status.OperationType = ""
					createdInstance.Status.OperationURL = ""
					err = k8sClient.Status().Update(context.Background(), createdInstance)
					Expect(err).ToNot(HaveOccurred())

					Eventually(func() bool {
						err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: bindingTestNamespace}, createdBinding)
						return err == nil && isReady(createdBinding)
					}, timeout, interval).Should(BeTrue())
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
					JustBeforeEach(func() {
						fakeClient.ListBindingsReturnsOnCall(0,
							&smclientTypes.ServiceBindings{
								ServiceBindings: []smclientTypes.ServiceBinding{*fakeBinding(testCase.lastOpState)},
							}, nil)
						fakeClient.StatusReturns(&smclientTypes.Operation{ResourceID: fakeBindingID, State: string(smTypes.IN_PROGRESS)}, nil)
					})
					JustAfterEach(func() {
						fakeClient.StatusReturns(&smclientTypes.Operation{ResourceID: fakeBindingID, State: string(smTypes.SUCCEEDED)}, nil)
						fakeClient.GetBindingByIDReturns(fakeBinding(smTypes.SUCCEEDED), nil)
					})
					When(fmt.Sprintf("last operation is %s %s", testCase.lastOpType, testCase.lastOpState), func() {
						It("should resync status", func() {
							var err error
							createdBinding, err = createBindingWithoutAssertionsAndWait(context.Background(), bindingName, bindingTestNamespace, instanceName, "fake-binding-external-name", false)
							Expect(err).ToNot(HaveOccurred())
							smCallArgs := fakeClient.ListBindingsArgsForCall(0)
							Expect(smCallArgs.LabelQuery).To(HaveLen(3))
							Expect(smCallArgs.FieldQuery).To(HaveLen(1))
							Expect(createdBinding.Status.Conditions[0].Reason).To(Equal(getConditionReason(testCase.lastOpType, testCase.lastOpState)))
							//TODO verify correct parameters used to find binding in SM are correct

							switch testCase.lastOpState {
							case smTypes.FAILED:
								Expect(isFailed(createdBinding))
							case smTypes.IN_PROGRESS:
								Expect(isInProgress(createdBinding))
							case smTypes.SUCCEEDED:
								Expect(isReady(createdBinding))
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
			createdBinding = createBinding(context.Background(), bindingName, bindingTestNamespace, instanceName, "binding-external-name")
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

	})

	Context("Delete", func() {

		validateBindingDeletion := func(binding *v1alpha1.ServiceBinding) {
			secretName := binding.Status.SecretName
			Expect(secretName).ToNot(BeEmpty())
			Expect(k8sClient.Delete(context.Background(), binding)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: bindingTestNamespace}, binding)
				if apierrors.IsNotFound(err) {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: bindingTestNamespace}, &v1.Secret{})
					return apierrors.IsNotFound(err)
				}
				return false
			}, timeout, interval).Should(BeTrue())
		}

		validateBindingNotDeleted := func(binding *v1alpha1.ServiceBinding, errorMessage string) {
			secretName := createdBinding.Status.SecretName
			Expect(secretName).ToNot(BeEmpty())
			Expect(k8sClient.Delete(context.Background(), createdBinding)).To(Succeed())

			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: bindingTestNamespace}, createdBinding)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: bindingName, Namespace: bindingTestNamespace}, createdBinding)
				if err != nil {
					return false
				}
				return len(createdBinding.Status.Conditions) == 2
			}, timeout, interval).Should(BeTrue())

			Expect(createdBinding.Status.Conditions[0].Reason).To(Equal(getConditionReason(smTypes.DELETE, smTypes.FAILED)))
			Expect(createdBinding.Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(createdBinding.Status.Conditions[1].Reason).To(Equal(getConditionReason(smTypes.DELETE, smTypes.FAILED)))
			Expect(createdBinding.Status.Conditions[1].Status).To(Equal(metav1.ConditionTrue))
			Expect(createdBinding.Status.Conditions[1].Message).To(ContainSubstring(errorMessage))

			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: bindingTestNamespace}, &v1.Secret{})
			Expect(err).ToNot(HaveOccurred())
		}

		JustBeforeEach(func() {
			createdBinding = createBinding(context.Background(), bindingName, bindingTestNamespace, instanceName, "binding-external-name")
			Expect(isReady(createdBinding)).To(BeTrue())
		})

		Context("Sync", func() {
			When("delete in SM succeeds", func() {
				JustBeforeEach(func() {
					fakeClient.UnbindReturns("", nil)
				})
				It("should delete the k8s binding and secret", func() {
					validateBindingDeletion(createdBinding)
				})
			})

			When("delete in SM fails with general error", func() {
				errorMessage := "some-error"
				JustBeforeEach(func() {
					fakeClient.UnbindReturns("", fmt.Errorf(errorMessage))
				})

				It("should not remove finalizer and keep the secret", func() {
					validateBindingNotDeleted(createdBinding, errorMessage)
				})
			})

			When("delete in SM fails with transient error", func() {
				JustBeforeEach(func() {
					fakeClient.UnbindReturnsOnCall(0, "", &smclient.ServiceManagerError{StatusCode: http.StatusTooManyRequests})
					fakeClient.UnbindReturnsOnCall(1, "", nil)
				})

				It("should eventually succeed", func() {
					validateBindingDeletion(createdBinding)
				})
			})
		})

		Context("Async", func() {
			JustBeforeEach(func() {
				fakeClient.UnbindReturns(smclient.BuildOperationURL("an-operation-id", fakeBindingID, web.ServiceBindingsURL), nil)
			})

			When("polling ends with success", func() {
				JustBeforeEach(func() {
					fakeClient.StatusReturns(&smclientTypes.Operation{ResourceID: fakeBindingID, State: string(smTypes.SUCCEEDED)}, nil)
				})

				It("should delete the k8s binding and secret", func() {
					validateBindingDeletion(createdBinding)
				})
			})

			When("polling ends with FAILED state", func() {
				errorMessage := "delete-binding-async-error"
				JustBeforeEach(func() {
					fakeClient.StatusReturns(&smclientTypes.Operation{
						Type:        string(smTypes.DELETE),
						State:       string(smTypes.FAILED),
						Description: errorMessage,
					}, nil)
				})

				It("should not delete the k8s binding and secret", func() {
					validateBindingNotDeleted(createdBinding, errorMessage)
				})
			})

			When("polling returns error", func() {

				JustBeforeEach(func() {
					fakeClient.UnbindReturnsOnCall(0, smclient.BuildOperationURL("an-operation-id", fakeBindingID, web.ServiceBindingsURL), nil)
					fakeClient.StatusReturns(nil, fmt.Errorf("no polling for you"))
					fakeClient.GetBindingByIDReturns(&smclientTypes.ServiceBinding{ID: fakeBindingID, LastOperation: &smTypes.Operation{State: smTypes.SUCCEEDED, Type: smTypes.CREATE}}, nil)
					fakeClient.UnbindReturnsOnCall(1, "", nil)
				})

				It("should recover and eventually succeed", func() {
					validateBindingDeletion(createdBinding)
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
