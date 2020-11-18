package controllers

import (
	"context"
	"fmt"
	smTypes "github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sm-operator/sapcp-operator/api/v1alpha1"
	"github.com/sm-operator/sapcp-operator/internal/smclient"
	types2 "github.com/sm-operator/sapcp-operator/internal/smclient/types"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
)

// +kubebuilder:docs-gen:collapse=Imports

const (
	fakeInstanceID           = "ic-fake-instance-id"
	fakeInstanceName         = "ic-test-instance"
	fakeInstanceExternalName = "ic-test-instance-external-name"
	testNamespace            = "ic-test-namespace"
)

var _ = Describe("ServiceInstance controller", func() {

	var createdInstance *v1alpha1.ServiceInstance

	createInstance := func(ctx context.Context, name, namespace, externalName string) *v1alpha1.ServiceInstance {
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
			return len(createdInstance.Status.Conditions) > 0
		}, timeout, interval).Should(BeTrue())
		//Expect(createdInstance.Status.InstanceID).ToNot(BeEmpty())

		return createdInstance
	}

	deleteInstance := func(ctx context.Context, instanceToDelete *v1alpha1.ServiceInstance) {

		instanceLookupKey := types.NamespacedName{Name: instanceToDelete.Name, Namespace: instanceToDelete.Namespace}
		k8sClient.Delete(ctx, instanceToDelete)

		Eventually(func() bool {
			err := k8sClient.Get(ctx, instanceLookupKey, &v1alpha1.ServiceInstance{})
			return errors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	}

	BeforeEach(func() {
		fakeClient.ProvisionReturns(fakeInstanceID, "", nil)

	})

	AfterEach(func() {
		if createdInstance != nil {
			deleteInstance(context.Background(), createdInstance)
		}
	})

	Context("Create", func() {
		XContext("Invalid parameters", func() {
			Context("service plan id not provided", func() {
				When("service offering name and service plan name are not provided", func() {
					It("provisioning should fail", func() {
						//TODO
					})
				})
				When("service offering name is provided and service plan name is not provided", func() {
					It("provisioning should fail", func() {
						//TODO
					})
				})
				When("service offering name not provided and service plan name is provided", func() {
					It("provisioning should fail", func() {
						//TODO
					})
				})
			})
			Context("service plan id is provided", func() {
				When("plan id does not exists in SM", func() {
					It("provisioning should fail", func() {
						//TODO
					})
				})
				When("plan id does not match the provided plan name", func() {
					It("provisioning should fail", func() {
						//TODO
					})
				})
				When("plan id does match provided plan name but not match provided offering name", func() {
					It("provisioning should fail", func() {
						//TODO
					})
				})
			})
		})

		Context("Valid parameters", func() {
			Context("Sync", func() {
				When("service offering and service plan name are provided and service plan id not provided", func() {
					It("should provision instance of the provided offering and plan name successfully", func() {
						createdInstance = createInstance(context.Background(), fakeInstanceName, testNamespace, fakeInstanceExternalName)
						Expect(createdInstance.Status.InstanceID).To(Equal(fakeInstanceID))
						Expect(createdInstance.Spec.ExternalName).To(Equal(fakeInstanceExternalName))
						Expect(createdInstance.Name).To(Equal(fakeInstanceName))
						//Expect(fakeClient.ProvisionCallCount()).To(Equal(1))
					})
				})

				When("provision request to SM fails", func() {
					var errMessage string
					BeforeEach(func() {
						errMessage = "failed to provision instance"
						fakeClient.ProvisionReturns("", "", &smclient.ServiceManagerError{
							StatusCode: http.StatusBadRequest,
							Message:    errMessage,
						})
					})

					It("should have failure condition", func() {
						createdInstance := createInstance(context.Background(), testNamespace, fakeInstanceName, fakeInstanceExternalName)
						Expect(len(createdInstance.Status.Conditions)).To(Equal(2))
						Expect(createdInstance.Status.Conditions[0].Status).To(Equal(v1alpha1.ConditionFalse))
						Expect(createdInstance.Status.Conditions[0].Message).To(ContainSubstring(errMessage))
						//TODO should have instance ID?
					})
				})
			})
		})

		XContext("Async", func() {
			When("service offering and service plan name are provided and service plan id not provided", func() {
				It("should update in progress condition and provision the instance successfully", func() {
					//TODO
				})
			})
			When("provision request to SM fails", func() {
				It("should update in progress condition and afterwards failure condition", func() {
					//TODO
				})
			})
		})

		XWhen("service offering and service plan name and matching service plan id are provided", func() {
			It("should provision instance of the provided offering and plan name successfully", func() {
				//TODO
			})
		})

		When("external name is not provided", func() {
			It("succeeds and uses the k8s name as external name", func() {
				createdInstance := createInstance(context.Background(), fakeInstanceName, testNamespace, "")
				Expect(createdInstance.Status.InstanceID).To(Equal(fakeInstanceID))
				Expect(createdInstance.Spec.ExternalName).To(Equal(fakeInstanceName))
				Expect(createdInstance.Name).To(Equal(fakeInstanceName))
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
				createdInstance = createInstance(context.Background(), fakeInstanceName, testNamespace, fakeInstanceExternalName)
				Expect(createdInstance.Status.InstanceID).To(Equal(fakeInstanceID))
				//Expect(fakeClient.ListInstancesCallCount()).To(Equal(1))
				//Expect(fakeClient.ProvisionCallCount()).To(Equal(0))
			})
		})
	})

	XContext("Update", func() {
		XWhen("spec is changed", func() {
			It("should succeed", func() {
				//TODO
			})
		})
	})

	XContext("Delete", func() {
		XWhen("delete in SM succeeds", func() {
			It("should delete the k8s instance", func() {
				//TODO
			})
		})

		XWhen("delete in SM is async", func() {
			When("polling ends with success", func() {
				It("should delete the k8s instance", func() {
					//TODO
				})
			})
			When("polling ends with failure", func() {
				It("should not delete the k8s instance and condition is updated with failure", func() {
					//TODO
				})
			})
		})
	})

})
