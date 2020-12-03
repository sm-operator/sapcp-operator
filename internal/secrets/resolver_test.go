package secrets_test

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sm-operator/sapcp-operator/api/v1alpha1"
	"github.com/sm-operator/sapcp-operator/internal/secrets"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// +kubebuilder:docs-gen:collapse=Imports

const (
	managementNamespace = "test-management-namespace"
	testNamespace       = "test-namespace"
)

var _ = Describe("Secrets Resolver", func() {

	var ctx context.Context
	var resolver *secrets.SecretResolver
	var expectedClientID string
	var secret *corev1.Secret

	createSecret := func(namePrefix string, namespace string) *corev1.Secret {
		var name string
		if namePrefix == "" {
			name = secrets.SAPCPOperatorSecretName
		} else {
			name = fmt.Sprintf("%s-%s", namePrefix, secrets.SAPCPOperatorSecretName)
		}

		expectedClientID = uuid.New().String()
		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"clientid":     []byte(expectedClientID),
				"clientsecret": []byte("client-secret"),
				"url":          []byte("https://some.url"),
				"subdomain":    []byte("sub-domain"),
			},
		}

		err := k8sClient.Create(ctx, newSecret)
		Expect(err).ToNot(HaveOccurred())

		return newSecret
	}

	validateSecretResolved := func() {
		resolvedSecret, err := resolver.GetSecretForResource(ctx, &v1alpha1.ServiceInstance{
			ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(resolvedSecret).ToNot(BeNil())
		Expect(string(resolvedSecret.Data["clientid"])).To(Equal(expectedClientID))
	}

	validateSecretNotResolved := func() {
		_, err := resolver.GetSecretForResource(ctx, &v1alpha1.ServiceInstance{
			ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: "test"},
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cannot find sapcp operator secret"))
	}

	BeforeEach(func() {
		ctx = context.Background()
		resolver = &secrets.SecretResolver{
			ManagementNamespace: managementNamespace,
			Log:                 logf.Log.WithName("SecretResolver"),
			Client:              k8sClient,
		}
	})

	AfterEach(func() {
		if secret != nil {
			err := k8sClient.Delete(ctx, secret)
			Expect(client.IgnoreNotFound(err)).ToNot(HaveOccurred())
		}
	})

	Context("Secret doesn't exist", func() {
		It("should fail to resolve the secret", func() {
			validateSecretNotResolved()
		})
	})

	Context("Secret in resource namespace", func() {
		BeforeEach(func() {
			secret = createSecret("", testNamespace)
		})
		Context("Namespace secrets enabled", func() {
			BeforeEach(func() {
				resolver.EnableNamespaceSecrets = true
			})
			It("should resolve the secret", func() {
				fmt.Printf("secret %v", secret)
				validateSecretResolved()
			})
		})

		Context("Namespace secrets disabled", func() {
			It("should fail to resolve the secret", func() {
				validateSecretNotResolved()
			})

			When("secret for resource namespace exists in management namespace", func() {
				var anotherSecret *corev1.Secret

				BeforeEach(func() {
					anotherSecret = createSecret(testNamespace, managementNamespace)
				})

				AfterEach(func() {
					err := k8sClient.Delete(ctx, anotherSecret)
					Expect(client.IgnoreNotFound(err)).ToNot(HaveOccurred())
				})

				It("should resolve the secret", func() {
					validateSecretResolved()
				})
			})

			When("cluster exists in management namespace", func() {
				var anotherSecret *corev1.Secret

				BeforeEach(func() {
					anotherSecret = createSecret("", managementNamespace)
				})

				AfterEach(func() {
					err := k8sClient.Delete(ctx, anotherSecret)
					Expect(client.IgnoreNotFound(err)).ToNot(HaveOccurred())
				})

				It("should resolve the secret", func() {
					validateSecretResolved()
				})
			})
		})
	})

	Context("Secret for resource namespace is in management namespace", func() {
		BeforeEach(func() {
			secret = createSecret(testNamespace, managementNamespace)
		})

		It("should resolve the secret", func() {
			validateSecretResolved()
		})
	})

	Context("Cluster secret is in management namespace", func() {
		BeforeEach(func() {
			secret = createSecret("", managementNamespace)
		})

		It("should resolve the secret", func() {
			validateSecretResolved()
		})
	})

})
