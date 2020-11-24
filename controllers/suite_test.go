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
	"crypto/tls"
	"fmt"
	"github.com/sm-operator/sapcp-operator/internal"
	"github.com/sm-operator/sapcp-operator/internal/smclient"
	"net"
	"path/filepath"
	"testing"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/sm-operator/sapcp-operator/internal/smclient/smclientfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/sm-operator/sapcp-operator/api/v1alpha1"
	servicesv1alpha1 "github.com/sm-operator/sapcp-operator/api/v1alpha1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

const (
	timeout  = time.Second * 10
	interval = time.Millisecond * 250
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

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			DirectoryPaths: []string{filepath.Join("..", "config", "webhook")},
		},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = v1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = servicesv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	webhookInstallOptions := &testEnv.WebhookInstallOptions

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		Host:               webhookInstallOptions.LocalServingHost,
		Port:               webhookInstallOptions.LocalServingPort,
		CertDir:            webhookInstallOptions.LocalServingCertDir,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	Expect(err).ToNot(HaveOccurred())

	fakeClient = &smclientfakes.FakeClient{}

	err = (&ServiceInstanceReconciler{
		Client:   k8sManager.GetClient(),
		Scheme:   k8sManager.GetScheme(),
		Log:      ctrl.Log.WithName("controllers").WithName("ServiceInstance"),
		SMClient: func() smclient.Client { return fakeClient },
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&servicesv1alpha1.ServiceInstance{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&ServiceBindingReconciler{
		Client:   k8sManager.GetClient(),
		Scheme:   k8sManager.GetScheme(),
		Log:      ctrl.Log.WithName("controllers").WithName("ServiceBinding"),
		SMClient: func() smclient.Client { return fakeClient },
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&servicesv1alpha1.ServiceBinding{}).SetupWebhookWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// +kubebuilder:scaffold:webhook

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

	// wait for the webhook server to get ready
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}, timeout, interval).Should(Succeed())

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func isReady(resource internal.SAPCPResource) bool {
	return len(resource.GetConditions()) == 1 && resource.GetConditions()[0].Status == v1alpha1.ConditionTrue
}

func isFailed(resource internal.SAPCPResource) bool {
	return len(resource.GetConditions()) == 2 && resource.GetConditions()[1].Status == v1alpha1.ConditionTrue
}
