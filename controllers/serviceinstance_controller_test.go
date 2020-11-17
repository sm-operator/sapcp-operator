package controllers

import (
	"context"
	"github.com/sm-operator/sapcp-operator/api/v1alpha1"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// +kubebuilder:docs-gen:collapse=Imports

var _ = XDescribe("ServiceInstance controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("Provisioning new service instance", func() {
		It("Should succeed", func() {
			ctx := context.Background()
			instance := &v1alpha1.ServiceInstance{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "services.cloud.sap.com/v1alpha1",
					Kind:       "ServiceInstance",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-instance",
					Namespace: "some-namespace",
				},
				Spec: v1alpha1.ServiceInstanceSpec{
					ServicePlanID:       "1111",
					ServicePlanName:     "planName",
					ServiceOfferingName: "offeringName",
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			instanceLookupKey := types.NamespacedName{Name: "new-instance", Namespace: "some-namespace"}
			createdInstance := &v1alpha1.ServiceInstance{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, instanceLookupKey, createdInstance)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			// Let's make sure our Schedule string value was properly converted/handled.
			Expect(createdInstance.Status.InstanceID).ToNot(BeEmpty())
			Expect(createdInstance.Spec.ExternalName).To(Equal(createdInstance.Name))
		})
	})
})
