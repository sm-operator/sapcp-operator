package secrets

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//TODO + revisit the name based approach for managed secret, replace with label based mechanism + admission webhook for secrets to avoid duplications

const (
	SAPCPOperatorSecretName = "sapcp-operator-secret"
)

type SecretResolver struct {
	ManagementNamespace    string
	EnableNamespaceSecrets bool
	Client                 client.Client
	Log                    logr.Logger
}

func (sr *SecretResolver) GetSecretForResource(ctx context.Context, resource controllerutil.Object) (*v1.Secret, error) {
	var secretForResource *v1.Secret
	var err error
	found := false

	if sr.EnableNamespaceSecrets {
		sr.Log.Info("Searching for secret in resource namespace", "namespace", resource.GetNamespace())
		secretForResource, err = sr.getSecretFromNamespace(ctx, resource.GetNamespace())
		if client.IgnoreNotFound(err) != nil {
			return nil, err
		}

		found = !apierrors.IsNotFound(err)
	}

	if !found {
		// secret not found in resource namespace, search for namespace-specific secret in management namespace
		sr.Log.Info("Searching for namespace secret in management namespace", "namespace", resource.GetNamespace(), "managementNamespace", sr.ManagementNamespace)
		secretForResource, err = sr.getSecretForNamespace(ctx, resource.GetNamespace())
		if client.IgnoreNotFound(err) != nil {
			return nil, err
		}

		found = !apierrors.IsNotFound(err)
	}

	if !found {
		// namespace-specific secret not found in management namespace, fallback to central cluster secret
		sr.Log.Info("Searching for cluster secret", "managementNamespace", sr.ManagementNamespace)
		secretForResource, err = sr.getClusterSecret(ctx)
		if client.IgnoreNotFound(err) != nil {
			return nil, err
		}

		found = !apierrors.IsNotFound(err)
	}

	if !found {
		// secret not found anywhere
		return nil, fmt.Errorf("cannot find sapcp operator secret")
	}

	if err := validateSAPCPOperatorSecret(secretForResource); err != nil {
		sr.Log.Error(err, "failed to validate secret", "secretName", secretForResource.Name, "secretNamespace", secretForResource.Namespace)
		return nil, err
	}

	return secretForResource, nil
}

func validateSAPCPOperatorSecret(secret *v1.Secret) error {
	secretData := secret.Data
	if secretData == nil {
		return fmt.Errorf("invalid secret - secret data is missing")
	}

	for _, field := range []string{"clientid", "clientsecret", "url", "subdomain"} {
		if len(secretData[field]) == 0 {
			return fmt.Errorf("invalid secret - %s is missing", field)
		}
	}

	return nil
}

func (sr *SecretResolver) getSecretFromNamespace(ctx context.Context, namespace string) (*v1.Secret, error) {
	secret := &v1.Secret{}
	err := sr.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: SAPCPOperatorSecretName}, secret)
	return secret, err
}

func (sr *SecretResolver) getSecretForNamespace(ctx context.Context, namespace string) (*v1.Secret, error) {
	secret := &v1.Secret{}
	err := sr.Client.Get(ctx, types.NamespacedName{Namespace: sr.ManagementNamespace, Name: fmt.Sprintf("%s-%s", namespace, SAPCPOperatorSecretName)}, secret)
	return secret, err
}

func (sr *SecretResolver) getClusterSecret(ctx context.Context) (*v1.Secret, error) {
	secret := &v1.Secret{}
	err := sr.Client.Get(ctx, types.NamespacedName{Namespace: sr.ManagementNamespace, Name: SAPCPOperatorSecretName}, secret)
	return secret, err
}
