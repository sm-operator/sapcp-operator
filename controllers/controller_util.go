package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	smTypes "github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/go-logr/logr"
	servicesv1alpha1 "github.com/sm-operator/sapcp-operator/api/v1alpha1"
	"github.com/sm-operator/sapcp-operator/internal"
	"github.com/sm-operator/sapcp-operator/internal/smclient"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Created = "Created"
	Updated = "Updated"
	Deleted = "Deleted"

	CreateInProgress = "CreateInProgress"
	UpdateInProgress = "UpdateInProgress"
	DeleteInProgress = "DeleteInProgress"

	CreateFailed = "CreateFailed"
	UpdateFailed = "UpdateFailed"
	DeleteFailed = "DeleteFailed"

	Unknown = "Unknown"
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

func buildOperationURL(operationID, resourceID, resourceURL string) string {
	return fmt.Sprintf("%s/%s%s/%s", resourceURL, resourceID, web.ResourceOperationsURL, operationID)
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

func setInProgressCondition(operationType smTypes.OperationCategory, message string, object internal.SAPCPResource) {
	conditions := make([]*servicesv1alpha1.Condition, 0)

	var defaultMessage string
	if operationType == smTypes.CREATE {
		defaultMessage = fmt.Sprintf("%s is being created", object.GetControllerName())
	} else if operationType == smTypes.UPDATE {
		defaultMessage = fmt.Sprintf("%s is being updated", object.GetControllerName())
	} else if operationType == smTypes.DELETE {
		defaultMessage = fmt.Sprintf("%s is being deleted", object.GetControllerName())
	}

	if len(message) == 0 {
		message = defaultMessage
	}

	conditions = append(conditions, &servicesv1alpha1.Condition{
		Type:               servicesv1alpha1.ConditionReady,
		Status:             servicesv1alpha1.ConditionFalse,
		LastTransitionTime: v1.Now(),
		Reason:             getConditionReason(operationType, smTypes.IN_PROGRESS),
		Message:            message,
	})

	object.SetConditions(conditions)
}

func setSuccessConditions(operationType smTypes.OperationCategory, object internal.SAPCPResource) {
	conditions := make([]*servicesv1alpha1.Condition, 0)

	var message string
	if operationType == smTypes.CREATE {
		message = fmt.Sprintf("%s provisioned successfully", object.GetControllerName())
	} else if operationType == smTypes.UPDATE {
		message = fmt.Sprintf("%s updated successfully", object.GetControllerName())
	} else if operationType == smTypes.DELETE {
		message = fmt.Sprintf("%s deleted successfully", object.GetControllerName())
	}

	conditions = append(conditions, &servicesv1alpha1.Condition{
		Type:               servicesv1alpha1.ConditionReady,
		Status:             servicesv1alpha1.ConditionTrue,
		LastTransitionTime: v1.Now(),
		Reason:             getConditionReason(operationType, smTypes.SUCCEEDED),
		Message:            message,
	})
	object.SetConditions(conditions)
}

func setFailureConditions(operationType smTypes.OperationCategory, errorMessage string, object internal.SAPCPResource) bool {
	var message string
	if operationType == smTypes.CREATE {
		message = fmt.Sprintf("%s create failed: %s", object.GetControllerName(), errorMessage)
	} else if operationType == smTypes.UPDATE {
		message = fmt.Sprintf("%s update failed: %s", object.GetControllerName(), errorMessage)
	} else if operationType == smTypes.DELETE {
		message = fmt.Sprintf("%s deletion failed: %s", object.GetControllerName(), errorMessage)
	}

	reason := getConditionReason(operationType, smTypes.FAILED)

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
		case servicesv1alpha1.ConditionFailed:
			if conditionChanged(failedCondition, condition) {
				object.SetConditions([]*servicesv1alpha1.Condition{&readyCondition, &failedCondition})
				return true
			}
		}
	}
	return false
}

func getSMClient(ctx context.Context, secret *corev1.Secret, log logr.Logger) (smclient.Client, error) {
	if secret == nil {
		return nil, fmt.Errorf("cannot create SM client - secret is missing")
	}
	secretData := secret.Data
	cl, err := smclient.NewClient(ctx, &smclient.ClientConfig{
		ClientID:     string(secretData["clientid"]),
		ClientSecret: string(secretData["clientsecret"]),
		URL:          string(secretData["url"]),
		Subdomain:    string(secretData["subdomain"]),
		SSLDisabled:  false,
	}, nil)

	if err != nil {
		log.Error(err, "Failed to initialize SM client")
		return nil, err
	}
	return cl, nil

}

func getConditionReason(opType smTypes.OperationCategory, state smTypes.OperationState) string {
	switch state {
	case smTypes.SUCCEEDED:
		if opType == smTypes.CREATE {
			return Created
		} else if opType == smTypes.UPDATE {
			return Updated
		} else if opType == smTypes.DELETE {
			return Deleted
		}
	case smTypes.IN_PROGRESS, smTypes.PENDING:
		if opType == smTypes.CREATE {
			return CreateInProgress
		} else if opType == smTypes.UPDATE {
			return UpdateInProgress
		} else if opType == smTypes.DELETE {
			return DeleteInProgress
		}
	case smTypes.FAILED:
		if opType == smTypes.CREATE {
			return CreateFailed
		} else if opType == smTypes.UPDATE {
			return UpdateFailed
		} else if opType == smTypes.DELETE {
			return DeleteFailed
		}
	}

	return Unknown
}
