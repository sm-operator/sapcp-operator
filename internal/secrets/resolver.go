package secrets

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type SecretResolver struct {
	ManagementNamespace    string
	EnableNamespaceSecrets bool
}

func (sr *SecretResolver) GetSecretForResource(ctx context.Context, client client.Client, resource controllerutil.Object) (*v1.Secret, error) {
	var secretForResource *v1.Secret
	var err error

	if sr.EnableNamespaceSecrets {
		secretForResource, err = sr.getSecretFromNamespace(ctx, client, resource.GetNamespace())
		if err != nil {
			return nil, err
		}
	}

	if secretForResource == nil {
		// secret not found in resource namespace, search for namespace-specific secret in management namespace
		secretForResource, err = sr.getSecretForNamespace(ctx, client, resource.GetNamespace())
		if err != nil {
			return nil, err
		}
	}

	if secretForResource == nil {
		// namespace-specific secret not found in management namespace, fallback to central cluster secret
		secretForResource, err = sr.getClusterSecret(ctx, client)
		if err != nil {
			return nil, err
		}
	}

	if secretForResource == nil {
		return nil, fmt.Errorf("cannot find secret associated to %s %s", resource.GetObjectKind().GroupVersionKind().Kind, resource.GetName())
	}

	return secretForResource, nil
}

func (sr *SecretResolver) getSecretFromNamespace(ctx context.Context, c client.Client, namespace string) (*v1.Secret, error) {
	var secret *v1.Secret
	err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "sapcp-operator-secret"}, secret)
	return secret, client.IgnoreNotFound(err)
}

func (sr *SecretResolver) getSecretForNamespace(ctx context.Context, c client.Client, namespace string) (*v1.Secret, error) {
	var secret *v1.Secret
	err := c.Get(ctx, types.NamespacedName{Namespace: sr.ManagementNamespace, Name: fmt.Sprintf("%s-sapcp-operator-secret", namespace)}, secret)
	return secret, client.IgnoreNotFound(err)
}

func (sr *SecretResolver) getClusterSecret(ctx context.Context, c client.Client) (*v1.Secret, error) {
	var secret *v1.Secret
	err := c.Get(ctx, types.NamespacedName{Namespace: sr.ManagementNamespace, Name: "sapcp-operator-secret"}, secret)
	return secret, client.IgnoreNotFound(err)
}
