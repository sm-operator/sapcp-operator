package controllers

import (
	"context"
	"fmt"
	smTypes "github.com/Peripli/service-manager/pkg/types"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sm-operator/sapcp-operator/api/v1alpha1"
	"github.com/sm-operator/sapcp-operator/internal/smclient"
	"github.com/sm-operator/sapcp-operator/internal/smclient/smclientfakes"
	types2 "github.com/sm-operator/sapcp-operator/internal/smclient/types"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"strings"
)

// +kubebuilder:docs-gen:collapse=Imports

const (
	fakeInstanceID           = "ic-fake-instance-id"
	fakeInstanceExternalName = "ic-test-instance-external-name"
	testNamespace            = "ic-test-namespace"
	fakeOfferingName         = "offering-a"
	fakePlanName             = "plan-a"
)

var _ = Describe("ServiceInstance controller", func() {

	var serviceInstance *v1alpha1.ServiceInstance
	var fakeInstanceName string
	var ctx context.Context
	var defaultLookupKey types.NamespacedName
	instanceSpec := v1alpha1.ServiceInstanceSpec{
		ExternalName:        fakeInstanceExternalName,
		ServicePlanName:     fakePlanName,
		ServiceOfferingName: fakeOfferingName,
	}

	createInstance := func(ctx context.Context, instanceSpec v1alpha1.ServiceInstanceSpec) *v1alpha1.ServiceInstance {
		instance := &v1alpha1.ServiceInstance{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "services.cloud.sap.com/v1alpha1",
				Kind:       "ServiceInstance",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fakeInstanceName,
				Namespace: testNamespace,
			},
			Spec: instanceSpec,
		}
		Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

		createdInstance := &v1alpha1.ServiceInstance{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, defaultLookupKey, createdInstance)
			if err != nil {
				return false
			}
			return len(createdInstance.Status.Conditions) > 0
		}, timeout, interval).Should(BeTrue())

		return createdInstance
	}

	deleteInstance := func(ctx context.Context, instanceToDelete *v1alpha1.ServiceInstance, wait bool) {
		err := k8sClient.Get(ctx, types.NamespacedName{Name: instanceToDelete.Name, Namespace: instanceToDelete.Namespace}, &v1alpha1.ServiceInstance{})
		if err != nil {
			Expect(errors.IsNotFound(err)).To(Equal(true))
			return
		}

		Expect(k8sClient.Delete(ctx, instanceToDelete)).Should(Succeed())

		if wait {
			Eventually(func() bool {
				a := &v1alpha1.ServiceInstance{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: instanceToDelete.Name, Namespace: instanceToDelete.Namespace}, a)
				return errors.IsNotFound(err)
			}, timeout*2, interval).Should(BeTrue())
		}
	}

	BeforeEach(func() {
		ctx = context.Background()
		fakeInstanceName = "ic-test-" + uuid.New().String()
		defaultLookupKey = types.NamespacedName{Name: fakeInstanceName, Namespace: testNamespace}

		fakeClient = &smclientfakes.FakeClient{}
		fakeClient.ProvisionReturns(fakeInstanceID, "", nil)
		fakeClient.DeprovisionReturns("", nil)
	})

	AfterEach(func() {
		if serviceInstance != nil {
			deleteInstance(ctx, serviceInstance, true)
		}
	})

	Describe("Create", func() {
		Context("Invalid parameters", func() {
			createInstanceWithFailure := func(spec v1alpha1.ServiceInstanceSpec) {
				instance := &v1alpha1.ServiceInstance{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "services.cloud.sap.com/v1alpha1",
						Kind:       "ServiceInstance",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      fakeInstanceName,
						Namespace: testNamespace,
					},
					Spec: spec,
				}
				Expect(k8sClient.Create(ctx, instance)).ShouldNot(Succeed())
			}
			Describe("service plan id not provided", func() {
				When("service offering name and service plan name are not provided", func() {
					It("provisioning should fail", func() {
						createInstanceWithFailure(v1alpha1.ServiceInstanceSpec{})
					})
				})
				When("service offering name is provided and service plan name is not provided", func() {
					It("provisioning should fail", func() {
						createInstanceWithFailure(v1alpha1.ServiceInstanceSpec{ServiceOfferingName: "fake-offering"})
					})
				})
				When("service offering name not provided and service plan name is provided", func() {
					It("provisioning should fail", func() {
						createInstanceWithFailure(v1alpha1.ServiceInstanceSpec{ServicePlanID: "fake-plan"})
					})
				})
			})
			Describe("service plan id is provided", func() {
				When("service offering name and service plan name are not provided", func() {
					It("provision should fail", func() {
						createInstanceWithFailure(v1alpha1.ServiceInstanceSpec{ServicePlanID: "fake-plan-id"})
					})
				})
				When("plan id does not match the provided offering name and plan name", func() {
					instanceSpec := v1alpha1.ServiceInstanceSpec{
						ServiceOfferingName: fakeOfferingName,
						ServicePlanName:     fakePlanName,
						ServicePlanID:       "wrong-id",
					}
					BeforeEach(func() {
						fakeClient.ProvisionReturns("", "", fmt.Errorf("provided plan id does not match the provided offeing name and plan name"))
					})
					It("provisioning should fail", func() {
						serviceInstance = createInstance(ctx, instanceSpec)
						Expect(serviceInstance.Status.Conditions[0].Message).To(ContainSubstring("provided plan id does not match"))
					})
				})
			})
		})

		Context("Sync", func() {
			When("provision request to SM succeeds", func() {
				It("should provision instance of the provided offering and plan name successfully", func() {
					serviceInstance = createInstance(ctx, instanceSpec)
					Expect(serviceInstance.Status.InstanceID).To(Equal(fakeInstanceID))
					Expect(serviceInstance.Spec.ExternalName).To(Equal(fakeInstanceExternalName))
					Expect(serviceInstance.Name).To(Equal(fakeInstanceName))
					Expect(fakeClient.ProvisionCallCount()).To(Equal(1))
				})
			})

			When("provision request to SM fails", func() {
				var errMessage string
				JustBeforeEach(func() {
					errMessage = "failed to provision instance"
					fakeClient.ProvisionReturns("", "", &smclient.ServiceManagerError{
						StatusCode: http.StatusBadRequest,
						Message:    errMessage,
					})
				})
				JustAfterEach(func() {
					fakeClient.ProvisionReturns(fakeInstanceID, "", nil)
				})

				It("should have failure condition", func() {
					serviceInstance = createInstance(ctx, instanceSpec)
					Expect(len(serviceInstance.Status.Conditions)).To(Equal(2))
					Expect(serviceInstance.Status.Conditions[0].Status).To(Equal(v1alpha1.ConditionFalse))
					Expect(serviceInstance.Status.Conditions[0].Message).To(ContainSubstring(errMessage))
				})
			})
		})

		Context("Async", func() {
			BeforeEach(func() {
				fakeClient.ProvisionReturns(fakeInstanceID, "/v1/service_instances/fakeid/operation/1234", nil)
				fakeClient.StatusReturns(&types2.Operation{
					ID:    "1234",
					Type:  string(smTypes.CREATE),
					State: string(smTypes.IN_PROGRESS),
				}, nil)
			})
			JustAfterEach(func() {
				fakeClient.ProvisionReturns(fakeInstanceID, "", nil)
			})
			createInstanceAsync := func() {
				serviceInstance = createInstance(ctx, instanceSpec)
				Expect(serviceInstance.Status.OperationURL).To(Not(BeEmpty()))
				Expect(len(serviceInstance.Status.Conditions)).To(Equal(1))
				Expect(serviceInstance.Status.Conditions[0].Reason).To(Equal(CreateInProgress))
			}
			When("polling ends with success", func() {
				It("should update in progress condition and provision the instance successfully", func() {
					createInstanceAsync()
					fakeClient.StatusReturns(&types2.Operation{
						ID:    "1234",
						Type:  string(smTypes.CREATE),
						State: string(smTypes.SUCCEEDED),
					}, nil)
					Eventually(func() bool {
						err := k8sClient.Get(ctx, defaultLookupKey, serviceInstance)
						if err != nil || len(serviceInstance.Status.Conditions) != 1 || serviceInstance.Status.Conditions[0].Reason != Created {
							return false
						}
						return true
					}, timeout*2, interval).Should(BeTrue())
				})
			})
			When("polling ends with failure", func() {
				It("should update in progress condition and afterwards failure condition", func() {
					createInstanceAsync()
					fakeClient.StatusReturns(&types2.Operation{
						ID:    "1234",
						Type:  string(smTypes.CREATE),
						State: string(smTypes.FAILED),
					}, nil)
					Eventually(func() bool {
						err := k8sClient.Get(ctx, defaultLookupKey, serviceInstance)
						if err != nil || len(serviceInstance.Status.Conditions) != 2 || serviceInstance.Status.Conditions[0].Reason != CreateFailed {
							return false
						}
						return true
					}, timeout*2, interval).Should(BeTrue())
				})
			})
		})

		Context("Recovery", func() {
			When("instance exists in SM", func() {
				BeforeEach(func() {
					fakeClient.ProvisionReturns("", "", fmt.Errorf("ERROR"))
					fakeClient.ListInstancesReturns(&types2.ServiceInstances{
						ServiceInstances: []types2.ServiceInstance{
							{
								ID:            fakeInstanceID,
								Name:          fakeInstanceName,
								LastOperation: &smTypes.Operation{State: smTypes.SUCCEEDED, Type: smTypes.CREATE},
							},
						},
					}, nil)
				})
				AfterEach(func() {
					fakeClient.ListInstancesReturns(&types2.ServiceInstances{ServiceInstances: []types2.ServiceInstance{}}, nil)
				})
				It("should point to the existing instance and not create a new one", func() {
					serviceInstance = createInstance(ctx, instanceSpec)
					Expect(serviceInstance.Status.InstanceID).To(Equal(fakeInstanceID))
					Expect(fakeClient.ProvisionCallCount()).To(Equal(0))
				})
			})
		})

		When("external name is not provided", func() {
			It("succeeds and uses the k8s name as external name", func() {
				withoutExternal := v1alpha1.ServiceInstanceSpec{
					ServicePlanName:     "a-plan-name",
					ServiceOfferingName: "an-offering-name",
				}
				serviceInstance = createInstance(ctx, withoutExternal)
				Expect(serviceInstance.Status.InstanceID).To(Equal(fakeInstanceID))
				Expect(serviceInstance.Spec.ExternalName).To(Equal(fakeInstanceName))
				Expect(serviceInstance.Name).To(Equal(fakeInstanceName))
			})
		})
	})

	Describe("Update", func() {
		newExternalName := "my-new-external-name"
		updateSpec := v1alpha1.ServiceInstanceSpec{
			ExternalName:        newExternalName,
			ServicePlanName:     fakePlanName,
			ServiceOfferingName: fakeOfferingName,
		}
		isConditionRefersUpdateOp := func(instance *v1alpha1.ServiceInstance) bool {
			conditionReason := instance.Status.Conditions[0].Reason
			return strings.Contains(conditionReason, Updated) || strings.Contains(conditionReason, UpdateInProgress) || strings.Contains(conditionReason, UpdateFailed)

		}

		updateInstance := func(serviceInstance *v1alpha1.ServiceInstance) *v1alpha1.ServiceInstance {
			Expect(k8sClient.Update(ctx, serviceInstance)).Should(Succeed())
			updatedInstance := &v1alpha1.ServiceInstance{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, defaultLookupKey, updatedInstance)
				if err != nil {
					return false
				}
				return len(updatedInstance.Status.Conditions) > 0 && isConditionRefersUpdateOp(updatedInstance)
			}, timeout*2, interval).Should(BeTrue())

			return updatedInstance
		}

		JustBeforeEach(func() {
			serviceInstance.Spec = instanceSpec
			serviceInstance = createInstance(ctx, instanceSpec)
			Expect(serviceInstance.Spec.ExternalName).To(Equal(fakeInstanceExternalName))
		})
		Context("When update call to SM succeed", func() {
			Context("Sync", func() {
				When("spec is changed", func() {
					BeforeEach(func() {
						fakeClient.UpdateInstanceReturns(nil, "", nil)
					})
					It("condition should be Updated", func() {
						serviceInstance.Spec = updateSpec
						serviceInstance = updateInstance(serviceInstance)
						Expect(serviceInstance.Spec.ExternalName).To(Equal(newExternalName))
						Expect(serviceInstance.Status.Conditions[0].Reason).To(Equal(Updated))
					})
				})
			})
			Context("Async", func() {
				When("spec is changed", func() {
					BeforeEach(func() {
						fakeClient.UpdateInstanceReturns(nil, "/v1/service_instances/id/operation/1234", nil)
						fakeClient.StatusReturns(&types2.Operation{
							ID:    "1234",
							Type:  string(smTypes.UPDATE),
							State: string(smTypes.IN_PROGRESS),
						}, nil)
					})
					It("condition should be updated from in progress to Updated", func() {
						serviceInstance.Spec = updateSpec
						updatedInstance := updateInstance(serviceInstance)
						Expect(updatedInstance.Status.Conditions[0].Reason).To(Equal(UpdateInProgress))
						fakeClient.StatusReturns(&types2.Operation{
							ID:    "1234",
							Type:  string(smTypes.UPDATE),
							State: string(smTypes.SUCCEEDED),
						}, nil)
						Eventually(func() bool {
							err := k8sClient.Get(ctx, defaultLookupKey, updatedInstance)
							if err != nil || len(updatedInstance.Status.Conditions) != 1 || updatedInstance.Status.Conditions[0].Reason != Updated {
								return false
							}
							return true
						}, timeout*2, interval).Should(BeTrue())
						Expect(updatedInstance.Spec.ExternalName).To(Equal(newExternalName))
					})
				})
			})
		})

		Context("When update call to SM fails", func() {
			Context("Sync", func() {
				When("spec is changed", func() {
					BeforeEach(func() {
						fakeClient.UpdateInstanceReturns(nil, "", fmt.Errorf("failed to update instance"))
					})
					It("condition should be Updated", func() {
						serviceInstance.Spec = updateSpec
						updatedInstance := updateInstance(serviceInstance)
						Expect(updatedInstance.Status.Conditions[0].Reason).To(Equal(UpdateFailed))
					})
				})
			})
			Context("Async", func() {
				When("spec is changed", func() {
					BeforeEach(func() {
						fakeClient.UpdateInstanceReturns(nil, "/v1/service_instances/id/operation/1234", nil)
						fakeClient.StatusReturns(&types2.Operation{
							ID:    "1234",
							Type:  string(smTypes.UPDATE),
							State: string(smTypes.IN_PROGRESS),
						}, nil)
					})
					It("condition should be updated from in progress to Updated", func() {
						serviceInstance.Spec = updateSpec
						updatedInstance := updateInstance(serviceInstance)
						Expect(updatedInstance.Status.Conditions[0].Reason).To(Equal(UpdateInProgress))
						fakeClient.StatusReturns(&types2.Operation{
							ID:    "1234",
							Type:  string(smTypes.UPDATE),
							State: string(smTypes.FAILED),
						}, nil)
						Eventually(func() bool {
							err := k8sClient.Get(ctx, defaultLookupKey, updatedInstance)
							if err != nil || len(updatedInstance.Status.Conditions) != 2 || updatedInstance.Status.Conditions[0].Reason != UpdateFailed {
								return false
							}
							return true
						}, timeout*2, interval).Should(BeTrue())
					})
				})
			})
		})
	})

	Describe("Delete", func() {
		BeforeEach(func() {
			serviceInstance = createInstance(ctx, instanceSpec)
		})

		When("delete in SM succeeds", func() {
			BeforeEach(func() {
				fakeClient.DeprovisionReturns("", nil)
			})
			It("should delete the k8s instance", func() {
				deleteInstance(ctx, serviceInstance, true)
			})
		})

		When("delete in SM fails", func() {
			JustBeforeEach(func() {
				fakeClient.DeprovisionReturns("", fmt.Errorf("failed to delete instance"))
			})
			JustAfterEach(func() {
				fakeClient.DeprovisionReturns("", nil)
			})
			It("should not delete the k8s instance and should update the condition", func() {
				deleteInstance(ctx, serviceInstance, false)
				Eventually(func() bool {
					err := k8sClient.Get(ctx, defaultLookupKey, serviceInstance)
					if err != nil {
						return false
					}
					return len(serviceInstance.Status.Conditions) == 2 && serviceInstance.Status.Conditions[0].Reason == DeleteFailed
				}, timeout, interval).Should(BeTrue())
			})
		})

		When("delete in SM is async", func() {
			JustBeforeEach(func() {
				fakeClient.DeprovisionReturns("/v1/service_instances/id/operation/1234", nil)
				fakeClient.StatusReturns(&types2.Operation{
					ID:    "1234",
					Type:  string(smTypes.DELETE),
					State: string(smTypes.IN_PROGRESS),
				}, nil)
				deleteInstance(ctx, serviceInstance, false)
				Eventually(func() bool {
					err := k8sClient.Get(ctx, defaultLookupKey, serviceInstance)
					if err != nil {
						return false
					}
					return len(serviceInstance.Status.Conditions) == 1 && serviceInstance.Status.Conditions[0].Reason == DeleteInProgress
				}, timeout, interval).Should(BeTrue())
			})
			When("polling ends with success", func() {
				JustBeforeEach(func() {
					fakeClient.StatusReturns(&types2.Operation{
						ID:    "1234",
						Type:  string(smTypes.DELETE),
						State: string(smTypes.SUCCEEDED),
					}, nil)
				})
				It("should delete the k8s instance", func() {
					deleteInstance(ctx, serviceInstance, true)
				})
			})
			When("polling ends with failure", func() {
				JustBeforeEach(func() {
					fakeClient.StatusReturns(&types2.Operation{
						ID:    "1234",
						Type:  string(smTypes.DELETE),
						State: string(smTypes.FAILED),
					}, nil)
				})
				AfterEach(func() {
					fakeClient.DeprovisionReturns("", nil)
				})
				It("should not delete the k8s instance and condition is updated with failure", func() {
					deleteInstance(ctx, serviceInstance, false)
					Eventually(func() bool {
						err := k8sClient.Get(ctx, defaultLookupKey, serviceInstance)
						if errors.IsNotFound(err) {
							return false
						}
						return len(serviceInstance.Status.Conditions) == 2 && serviceInstance.Status.Conditions[0].Reason == DeleteFailed
					}, timeout*2, interval).Should(BeTrue())
				})
			})
		})
	})
})
