/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"github.com/sm-operator/sapcp-operator/internal/config"
	"github.com/sm-operator/sapcp-operator/internal/smclient"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/sm-operator/sapcp-operator/internal/smclient/smclientfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	servicesv1alpha1 "github.com/sm-operator/sapcp-operator/api/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

const (
	timeout      = time.Second * 20
	interval     = time.Millisecond * 10
	syncPeriod   = time.Millisecond * 250
	pollInterval = time.Millisecond * 250
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var fakeClient *smclientfakes.FakeClient

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = servicesv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	fakeClient = &smclientfakes.FakeClient{}
	testConfig := config.Config{SyncPeriod: syncPeriod, PollInterval: pollInterval}
	err = (&ServiceInstanceReconciler{
		BaseReconciler: &BaseReconciler{
			Client:   k8sManager.GetClient(),
			Scheme:   k8sManager.GetScheme(),
			Log:      ctrl.Log.WithName("controllers").WithName("ServiceInstance"),
			SMClient: func() smclient.Client { return fakeClient },
			Config:   testConfig,
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&ServiceBindingReconciler{
		BaseReconciler: &BaseReconciler{
			Client:   k8sManager.GetClient(),
			Scheme:   k8sManager.GetScheme(),
			Log:      ctrl.Log.WithName("controllers").WithName("ServiceBinding"),
			SMClient: func() smclient.Client { return fakeClient },
			Config:   testConfig,
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	nsSpec := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
	err = k8sClient.Create(context.Background(), nsSpec)
	Expect(err).ToNot(HaveOccurred())

	nsSpec = &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: bindingTestNamespace}}
	err = k8sClient.Create(context.Background(), nsSpec)
	Expect(err).ToNot(HaveOccurred())

}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func isReady(resource servicesv1alpha1.SAPCPResource) bool {
	return len(resource.GetConditions()) == 1 && resource.GetConditions()[0].Status == metav1.ConditionTrue
}

func isFailed(resource servicesv1alpha1.SAPCPResource) bool {
	return (len(resource.GetConditions()) == 2 && resource.GetConditions()[1].Status == metav1.ConditionTrue) ||
		len(resource.GetConditions()) == 1 && resource.GetConditions()[0].Status == metav1.ConditionFalse && resource.GetConditions()[0].Reason == Blocked
}
