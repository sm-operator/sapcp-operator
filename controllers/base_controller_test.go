package controllers

import (
	"context"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sm-operator/sapcp-operator/api/v1alpha1"
	"github.com/sm-operator/sapcp-operator/internal/secrets"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const managementNamespace = "test-management-namespace"

var _ = Describe("Base controller", func() {
	var serviceInstance *v1alpha1.ServiceInstance
	var fakeInstanceName string
	var ctx context.Context
	//var defaultLookupKey types.NamespacedName
	var controller *BaseReconciler

	BeforeEach(func() {
		ctx = context.Background()
		fakeInstanceName = "ic-test-" + uuid.New().String()
		//defaultLookupKey = types.NamespacedName{Name: fakeInstanceName, Namespace: testNamespace}

		resolver := &secrets.SecretResolver{
			ManagementNamespace: managementNamespace,
			Log:                 logf.Log.WithName("SecretResolver"),
			Client:              k8sClient,
		}
		controller = &BaseReconciler{
			SecretResolver: resolver,
			Log:            logf.Log.WithName("reconciler"),
			Client:         k8sClient,
		}

	})

	When("SM secret not exists", func() {
		It("Should fail with failure condition", func() {
			serviceInstance = &v1alpha1.ServiceInstance{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "services.cloud.sap.com/v1alpha1",
					Kind:       "ServiceInstance",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fakeInstanceName,
					Namespace: testNamespace,
				},
				Spec: v1alpha1.ServiceInstanceSpec{
					ExternalName:        fakeInstanceExternalName,
					ServicePlanName:     fakePlanName,
					ServiceOfferingName: fakeOfferingName,
				},
			}
			controller.getSMClient(ctx, controller.Log, serviceInstance)

			Expect(serviceInstance.Status.Conditions[0].Reason).To(Equal(Blocked))
			Expect(len(serviceInstance.Status.Conditions)).To(Equal(1))
		})
	})
})
