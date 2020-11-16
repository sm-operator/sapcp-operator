package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	smTypes "github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/go-logr/logr"
	servicesv1alpha1 "github.wdf.sap.corp/i042428/sapcp-operator/api/v1alpha1"
	"github.wdf.sap.corp/i042428/sapcp-operator/internal"
	"github.wdf.sap.corp/i042428/sapcp-operator/internal/smclient"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func conditionChanged(condition servicesv1alpha1.Condition, otherCondition *servicesv1alpha1.Condition) bool {
	return condition.Message != otherCondition.Message ||
		condition.Status != otherCondition.Status ||
		condition.Reason != otherCondition.Reason
}

func buildOperationURL(operationID, resourceID, resourceUrl string) string {
	return fmt.Sprintf("%s/%s%s/%s", resourceUrl, resourceID, web.ResourceOperationsURL, operationID)
}

func isDelete(object v1.ObjectMeta) bool {
	return !object.DeletionTimestamp.IsZero()
}

func normalizeCredentials(credentialsJSON json.RawMessage) (map[string][]byte, error) {
	var credentialsMap map[string]interface{}
	err := json.Unmarshal(credentialsJSON, &credentialsMap)
	if err != nil {
		return nil, err
	}

	normalized := make(map[string][]byte)
	for propertyName, value := range credentialsMap {
		keyString := strings.Replace(propertyName, " ", "_", -1)
		// need to re-marshal as json might have complex types, which need to be flattened in strings
		jString, err := json.Marshal(value)
		if err != nil {
			return normalized, err
		}
		// need to remove quotes from flattened objects
		strVal := strings.TrimPrefix(string(jString), "\"")
		strVal = strings.TrimSuffix(strVal, "\"")
		normalized[keyString] = []byte(strVal)
	}
	return normalized, nil
}

func setInProgressCondition(operationType smTypes.OperationCategory, message string, object internal.SapCPController) {
	conditions := make([]*servicesv1alpha1.Condition, 0)

	var defaultMessage, reason string
	if operationType == smTypes.CREATE {
		defaultMessage = fmt.Sprintf("%s is being created", object.GetControllerName())
		reason = "CreateInProgress"
	} else if operationType == smTypes.UPDATE {
		defaultMessage = fmt.Sprintf("%s is being created", object.GetControllerName())
		reason = "UpdateInProgress"
	} else if operationType == smTypes.DELETE {
		defaultMessage = fmt.Sprintf("%s is being updated", object.GetControllerName())
		reason = "DeleteInProgress"
	}

	if len(message) == 0 {
		message = defaultMessage
	}

	conditions = append(conditions, &servicesv1alpha1.Condition{
		Type:               servicesv1alpha1.ConditionReady,
		Status:             servicesv1alpha1.ConditionFalse,
		LastTransitionTime: v1.Now(),
		Reason:             reason,
		Message:            message,
	})

	object.SetConditions(conditions)
}

func setSuccessConditions(operationType smTypes.OperationCategory, object internal.SapCPController) {
	conditions := make([]*servicesv1alpha1.Condition, 0)

	var message, reason string
	if operationType == smTypes.CREATE {
		message = fmt.Sprintf("%s provisioned successfully", object.GetControllerName())
		reason = "Created"
	} else if operationType == smTypes.UPDATE {
		message = fmt.Sprintf("%s updated successfully", object.GetControllerName())
		reason = "Updated"
	} else if operationType == smTypes.DELETE {
		message = fmt.Sprintf("%s deleted successfully", object.GetControllerName())
		reason = "Deleted"
	}

	conditions = append(conditions, &servicesv1alpha1.Condition{
		Type:               servicesv1alpha1.ConditionReady,
		Status:             servicesv1alpha1.ConditionTrue,
		LastTransitionTime: v1.Now(),
		Reason:             reason,
		Message:            message,
	})
	object.SetConditions(conditions)
}

func setFailureConditions(operationType smTypes.OperationCategory, errorMessage string, object internal.SapCPController) bool {
	var message, reason string
	if operationType == smTypes.CREATE {
		message = fmt.Sprintf("%s create failed: %s",object.GetControllerName(), errorMessage)
		reason = "createFailed"
	} else if operationType == smTypes.UPDATE {
		message = fmt.Sprintf("%s update failed: %s",object.GetControllerName(), errorMessage)
		reason = "updateFailed"
	} else if operationType == smTypes.DELETE {
		message = fmt.Sprintf("%s deletion failed: %s", object.GetControllerName(), errorMessage)
		reason = "deleteFailed"
	}

	readyCondition := servicesv1alpha1.Condition{
		Type:               servicesv1alpha1.ConditionReady,
		Status:             servicesv1alpha1.ConditionFalse,
		LastTransitionTime: v1.Now(),
		Reason:             reason,
		Message:            message,
	}

	failedCondition := servicesv1alpha1.Condition{
		Type:               servicesv1alpha1.ConditionFailed,
		Status:             servicesv1alpha1.ConditionTrue,
		LastTransitionTime: v1.Now(),
		Reason:             reason,
		Message:            message,
	}

	if len(object.GetConditions()) != 2 {
		object.SetConditions([]*servicesv1alpha1.Condition{&readyCondition, &failedCondition})
		return true
	}

	for _, condition := range object.GetConditions() {
		switch condition.Type {
		case servicesv1alpha1.ConditionReady:
			if conditionChanged(readyCondition, condition) {
				object.SetConditions([]*servicesv1alpha1.Condition{&readyCondition, &failedCondition})
				return true
			}
			break
		case servicesv1alpha1.ConditionFailed:
			if conditionChanged(failedCondition, condition) {
				object.SetConditions([]*servicesv1alpha1.Condition{&readyCondition, &failedCondition})
				return true
			}
			break
		}
	}
	return false
}

func getSMClient(ctx context.Context, r client.Client, log logr.Logger) (smclient.Client, error) {
	secretData, err  := getSMSecret(ctx, r, log, "default")
	if err != nil {
		return nil, err
	}
	if cl, err := smclient.NewClient(string(secretData["subdomain"]), &smclient.ClientConfig{
		ClientID:     string(secretData["clientid"]),
		ClientSecret: string(secretData["clientsecret"]),
		URL:          string(secretData["url"]),
		SSLDisabled:  false,
	}); err != nil {
		log.Error(err, "Failed to initialize SM client")
		return nil, err
	} else {
		return cl, nil
	}
}

func getSMSecret(ctx context.Context, r client.Client, log logr.Logger, namespace string) (map[string][]byte, error) {
	log.Info("getting SM secret sapcp-operator")

	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: "sapcp-operator"}, &secret); err != nil {
		log.Error(err, "secret not found")
	}

	return secret.Data, nil
}




