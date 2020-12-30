package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	ctrl "sigs.k8s.io/controller-runtime"

	"k8s.io/apimachinery/pkg/api/meta"
	types2 "k8s.io/apimachinery/pkg/types"

	smTypes "github.com/Peripli/service-manager/pkg/types"
	"github.com/go-logr/logr"
	servicesv1alpha1 "github.com/sm-operator/sapcp-operator/api/v1alpha1"
	"github.com/sm-operator/sapcp-operator/internal/config"
	"github.com/sm-operator/sapcp-operator/internal/secrets"
	"github.com/sm-operator/sapcp-operator/internal/smclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	subaccountIDLabel = "subaccount_id"
	namespaceLabel    = "_namespace"
	k8sNameLabel      = "_k8sname"
	clusterIDLabel    = "_clusterid"

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

type BaseReconciler struct {
	client.Client
	Log            logr.Logger
	Scheme         *runtime.Scheme
	SMClient       func() smclient.Client
	Config         config.Config
	SecretResolver *secrets.SecretResolver
}

func getParameters(sapResource servicesv1alpha1.SAPCPResource) (json.RawMessage, error) {
	var instanceParameters json.RawMessage
	if sapResource.GetParameters() != nil {
		parametersJSON, err := sapResource.GetParameters().MarshalJSON()
		if err != nil {
			return nil, err
		}
		instanceParameters = parametersJSON
	}
	return instanceParameters, nil
}

func (r *BaseReconciler) getSMClient(ctx context.Context, log logr.Logger, object servicesv1alpha1.SAPCPResource) (smclient.Client, error) {
	if r.SMClient != nil {
		return r.SMClient(), nil
	}

	secret, err := r.SecretResolver.GetSecretForResource(ctx, object.GetNamespace())
	if err != nil {
		setFailureConditions(smTypes.CREATE, err.Error(), object)
		return nil, err
	}

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
		setFailureConditions(smTypes.CREATE, fmt.Sprintf("failed to create service-manager client: %s", err.Error()), object)
		if err := r.updateStatus(ctx, object, log); err != nil {
			return nil, err
		}
		return nil, err
	}
	return cl, nil
}

func (r *BaseReconciler) removeFinalizer(ctx context.Context, object servicesv1alpha1.SAPCPResource, finalizerName string) error {
	if containsString(object.GetFinalizers(), finalizerName) {
		finalizers := object.GetFinalizers()
		finalizers = removeString(finalizers, finalizerName)
		object.SetFinalizers(finalizers)
		if err := r.Update(ctx, object); err != nil {
			if err := r.Get(ctx, types2.NamespacedName{Name: object.GetName(), Namespace: object.GetNamespace()}, object); err != nil {
				return client.IgnoreNotFound(err)
			}
			if err := r.Update(ctx, object); err != nil {
				return fmt.Errorf("failed to remove finalizer %s : %v", finalizerName, err)
			}
		}
	}
	return nil
}

func (r *BaseReconciler) addFinalizer(ctx context.Context, object servicesv1alpha1.SAPCPResource, finalizerName string) error {
	if !containsString(object.GetFinalizers(), finalizerName) {
		finalizers := object.GetFinalizers()
		finalizers = append(finalizers, finalizerName)
		object.SetFinalizers(finalizers)
		if err := r.Update(ctx, object); err != nil {
			if err := r.Get(ctx, types2.NamespacedName{Name: object.GetName(), Namespace: object.GetNamespace()}, object); err != nil {
				return fmt.Errorf("failed to fetch latest %s : %v", object.GetControllerName(), err)
			}
			if err := r.Update(ctx, object); err != nil {
				return fmt.Errorf("failed to add finalizer %s : %v", finalizerName, err)
			}
		}
	}
	return nil
}

func (r *BaseReconciler) updateStatus(ctx context.Context, object servicesv1alpha1.SAPCPResource, log logr.Logger) error {
	log.Info(fmt.Sprintf("updating %s status", object.GetControllerName()))
	if err := r.Status().Update(ctx, object); err != nil {
		status := object.GetStatus()
		log.Info(fmt.Sprintf("failed to update status - %s, fetching latest %s and trying again", err.Error(), object.GetControllerName()))
		if err := r.Get(ctx, types2.NamespacedName{Name: object.GetName(), Namespace: object.GetNamespace()}, object); err != nil {
			log.Error(err, fmt.Sprintf("failed to fetch latest %s", object.GetControllerName()))
			return err
		}

		object.SetStatus(status)
		if err := r.Status().Update(ctx, object); err != nil {
			log.Error(err, fmt.Sprintf("unable to update %s status", object.GetControllerName()))
			return err
		}
	}
	log.Info(fmt.Sprintf("updated %s status in k8s", object.GetControllerName()))
	return nil
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

func setInProgressCondition(operationType smTypes.OperationCategory, message string, object servicesv1alpha1.SAPCPResource) {
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

	conditions := object.GetConditions()
	if len(conditions) > 0 {
		meta.RemoveStatusCondition(&conditions, servicesv1alpha1.ConditionFailed)
	}
	readyCondition := metav1.Condition{Type: servicesv1alpha1.ConditionReady, Status: metav1.ConditionFalse, Reason: getConditionReason(operationType, smTypes.IN_PROGRESS), Message: message}
	meta.SetStatusCondition(&conditions, readyCondition)
	object.SetConditions(conditions)
}

func setSuccessConditions(operationType smTypes.OperationCategory, object servicesv1alpha1.SAPCPResource) {
	var message string
	if operationType == smTypes.CREATE {
		message = fmt.Sprintf("%s provisioned successfully", object.GetControllerName())
	} else if operationType == smTypes.UPDATE {
		message = fmt.Sprintf("%s updated successfully", object.GetControllerName())
	} else if operationType == smTypes.DELETE {
		message = fmt.Sprintf("%s deleted successfully", object.GetControllerName())
	}

	conditions := object.GetConditions()
	if len(conditions) > 0 {
		meta.RemoveStatusCondition(&conditions, servicesv1alpha1.ConditionFailed)
	}
	readyCondition := metav1.Condition{Type: servicesv1alpha1.ConditionReady, Status: metav1.ConditionTrue, Reason: getConditionReason(operationType, smTypes.SUCCEEDED), Message: message}
	meta.SetStatusCondition(&conditions, readyCondition)
	object.SetConditions(conditions)
}

func setFailureConditions(operationType smTypes.OperationCategory, errorMessage string, object servicesv1alpha1.SAPCPResource) {
	var message string
	if operationType == smTypes.CREATE {
		message = fmt.Sprintf("%s create failed: %s", object.GetControllerName(), errorMessage)
	} else if operationType == smTypes.UPDATE {
		message = fmt.Sprintf("%s update failed: %s", object.GetControllerName(), errorMessage)
	} else if operationType == smTypes.DELETE {
		message = fmt.Sprintf("%s deletion failed: %s", object.GetControllerName(), errorMessage)
	}

	var reason string
	if operationType != Unknown {
		reason = getConditionReason(operationType, smTypes.FAILED)
	} else {
		reason = object.GetConditions()[0].Reason
	}

	conditions := object.GetConditions()
	readyCondition := metav1.Condition{Type: servicesv1alpha1.ConditionReady, Status: metav1.ConditionFalse, Reason: reason, Message: message}
	meta.SetStatusCondition(&conditions, readyCondition)

	failedCondition := metav1.Condition{Type: servicesv1alpha1.ConditionFailed, Status: metav1.ConditionTrue, Reason: reason, Message: message}
	meta.SetStatusCondition(&conditions, failedCondition)
	object.SetConditions(conditions)
}

func isDelete(object metav1.ObjectMeta) bool {
	return !object.DeletionTimestamp.IsZero()
}

func isTransientError(err error) bool {
	if smError, ok := err.(*smclient.ServiceManagerError); ok {
		if smError.StatusCode == http.StatusTooManyRequests || smError.StatusCode == http.StatusServiceUnavailable {
			return true
		}
	}
	//if error is final we ignore it (no point to retry)
	return false
}

func (r *BaseReconciler) markAsNonTransientError(ctx context.Context, operationType smTypes.OperationCategory, message string, object servicesv1alpha1.SAPCPResource, log logr.Logger) (ctrl.Result, error) {
	setFailureConditions(operationType, message, object)
	log.Info(fmt.Sprintf("operation %s of %s encountered a non transient error, giving up operation :(", operationType, object.GetControllerName()))
	object.SetObservedGeneration(object.GetGeneration())
	err := r.updateStatus(ctx, object, log)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *BaseReconciler) markAsTransientError(ctx context.Context, operationType smTypes.OperationCategory, message string, object servicesv1alpha1.SAPCPResource, log logr.Logger) (ctrl.Result, error) {
	setInProgressCondition(operationType, message, object)
	if err := r.updateStatus(ctx, object, log); err != nil {
		return ctrl.Result{}, err
	}

	log.Info(fmt.Sprintf("operation %s of %s encountered a transient error, will try again :)", operationType, object.GetControllerName()))
	return ctrl.Result{Requeue: true, RequeueAfter: r.Config.LongPollInterval}, nil
}
