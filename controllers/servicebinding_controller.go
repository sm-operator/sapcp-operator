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
	"encoding/json"
	"fmt"
	smTypes "github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/go-logr/logr"
	"github.com/sm-operator/sapcp-operator/api/v1alpha1"
	"github.com/sm-operator/sapcp-operator/internal/config"
	"github.com/sm-operator/sapcp-operator/internal/smclient"
	smclientTypes "github.com/sm-operator/sapcp-operator/internal/smclient/types"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const bindingFinalizerName string = "storage.finalizers.peripli.io.service-manager.serviceBinding"

// ServiceBindingReconciler reconciles a ServiceBinding object
type ServiceBindingReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	SMClient smclient.Client
}

// +kubebuilder:rbac:groups=services.cloud.sap.com,resources=servicebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=services.cloud.sap.com,resources=servicebindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=services.cloud.sap.com,resources=serviceinstances,verbs=get;list
// +kubebuilder:rbac:groups=services.cloud.sap.com,resources=serviceinstances/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *ServiceBindingReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	//TODO configuration mechanism for the operator - env, yaml files, config maps?

	ctx := context.Background()
	//TODO optimize log - use withValue where possible
	log := r.Log.WithValues("servicebinding", req.NamespacedName)

	// your logic here

	serviceBinding := &v1alpha1.ServiceBinding{}
	if err := r.Get(ctx, req.NamespacedName, serviceBinding); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch ServiceBinding")
		}
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if len(serviceBinding.Status.OperationURL) > 0 {
		// ongoing operation - poll status from SM
		log.Info(fmt.Sprintf("resource is in progress, found operation url %s", serviceBinding.Status.OperationURL))
		//TODO set client config
		smClient, err := r.getSMClient(ctx, log)
		if err != nil {
			return ctrl.Result{}, err
		}

		status, err := smClient.Status(serviceBinding.Status.OperationURL, nil)
		if err != nil {
			log.Error(err, "failed to fetch operation", "operationURL", serviceBinding.Status.OperationURL)
			// TODO handle errors to fetch operation - should resync state from SM
			return ctrl.Result{}, err
		}

		switch status.State {
		case string(smTypes.IN_PROGRESS):
			fallthrough
		case string(smTypes.PENDING):
			return ctrl.Result{Requeue: true, RequeueAfter: config.Get().PollInterval}, nil
		case string(smTypes.FAILED):
			setFailureConditions(smTypes.OperationCategory(status.Type), status.Description, serviceBinding)
		case string(smTypes.SUCCEEDED):

			if serviceBinding.Status.OperationType == smTypes.CREATE {
				smBinding, err := smClient.GetBindingByID(serviceBinding.Status.BindingID, nil)
				if err != nil {
					log.Error(err, "Failed to get binding from SM")
				}

				if err := r.storeBindingSecret(ctx, serviceBinding, smBinding, log); err != nil {
					setFailureConditions(smTypes.CREATE, err.Error(), serviceBinding)
					return ctrl.Result{}, nil
				}
			}

			setSuccessConditions(smTypes.OperationCategory(status.Type), serviceBinding)
			if serviceBinding.Status.OperationType == smTypes.DELETE {
				// delete was successful - delete the secret
				// TODO extract deletion of the secert and removal of finalizer to common function
				if err = r.deleteBindingSecret(ctx, serviceBinding, log); err != nil {
					return ctrl.Result{}, err
				}

				//remove our finalizer from the list and update it.
				if err = r.removeFinalizer(ctx, serviceBinding, log); err != nil {
					return ctrl.Result{}, err
				}
			}
		}

		serviceBinding.Status.OperationURL = ""
		serviceBinding.Status.OperationType = ""

		if err := r.Status().Update(ctx, serviceBinding); err != nil {
			log.Error(err, "unable to update ServiceBinding status")
			return ctrl.Result{}, err
		}
	}

	if isDelete(serviceBinding.ObjectMeta) {
		if containsString(serviceBinding.Finalizers, bindingFinalizerName) {
			if len(serviceBinding.Status.BindingID) == 0 {
				// make sure there's no secret stored for the binding
				if err := r.deleteBindingSecret(ctx, serviceBinding, log); err != nil {
					return ctrl.Result{}, err
				}

				log.Info("Binding does not exists in SM, removing finalizer")
				if err := r.removeFinalizer(ctx, serviceBinding, log); err != nil {
					log.Error(err, "failed to remove finalizer")
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, nil
			}

			// our finalizer is present, so we need to delete the binding in SM
			smClient, err := r.getSMClient(ctx, log)
			if err != nil {
				return ctrl.Result{}, err
			}

			log.Info(fmt.Sprintf("Deleting binding with id %v from SM", serviceBinding.Status.BindingID))
			operationURL, err := smClient.Unbind(serviceBinding.Status.BindingID, nil)
			if err != nil {
				if smError, ok := err.(*smclient.ServiceManagerError); ok && smError.StatusCode == http.StatusNotFound {
					log.Info(fmt.Sprintf("Binding id %s not found in SM", serviceBinding.Status.BindingID))
					//if not found it means success
					serviceBinding.Status.BindingID = ""
					setSuccessConditions(smTypes.DELETE, serviceBinding)
					if err := r.Status().Update(ctx, serviceBinding); err != nil {
						return ctrl.Result{}, err
					}

					// delete binding secret if exist
					if err = r.deleteBindingSecret(ctx, serviceBinding, log); err != nil {
						return ctrl.Result{}, err
					}

					// remove our finalizer from the list and update it.
					if err := r.removeFinalizer(ctx, serviceBinding, log); err != nil {
						log.Error(err, "failed to remove finalizer")
						return ctrl.Result{}, err
					}

					// Stop reconciliation as the item is deleted
					return ctrl.Result{}, nil
				}

				//	//TODO handle non transient errors
				log.Error(err, "failed to delete binding")
				// if fail to delete the binding in SM, return with error
				// so that it can be retried
				if setFailureConditions(smTypes.DELETE, err.Error(), serviceBinding) {
					if err := r.Status().Update(ctx, serviceBinding); err != nil {
						return ctrl.Result{}, err
					}
				}

				return ctrl.Result{}, nil
			}

			if operationURL != "" {
				log.Info("Deleting binding async")
				serviceBinding.Status.OperationURL = operationURL
				serviceBinding.Status.OperationType = smTypes.DELETE
				setInProgressCondition(smTypes.DELETE, "", serviceBinding)

				if err := r.Status().Update(ctx, serviceBinding); err != nil {
					return ctrl.Result{}, err
				}

				return ctrl.Result{Requeue: true, RequeueAfter: config.Get().PollInterval}, nil
			} else {
				log.Info("Binding was deleted successfully")
				serviceBinding.Status.BindingID = ""
				setSuccessConditions(smTypes.DELETE, serviceBinding)
				if err := r.Status().Update(ctx, serviceBinding); err != nil {
					return ctrl.Result{}, err
				}

				if err = r.deleteBindingSecret(ctx, serviceBinding, log); err != nil {
					return ctrl.Result{}, err
				}

				// remove our finalizer from the list and update it.
				if err := r.removeFinalizer(ctx, serviceBinding, log); err != nil {
					log.Error(err, "failed to remove finalizer")
					return ctrl.Result{}, err
				}

				// Stop reconciliation as the item is being deleted
				return ctrl.Result{}, nil
			}
		}
	} else {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(serviceBinding.ObjectMeta.Finalizers, bindingFinalizerName) {
			log.Info("Binding has no finalizer, adding it...")
			if err := r.addFinalizer(ctx, serviceBinding); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if serviceBinding.Generation == serviceBinding.Status.ObservedGeneration {
		log.Info(fmt.Sprintf("Spec is not changed - ignoring... Generation is - %v", serviceBinding.Generation))
		return ctrl.Result{}, nil
	}

	log.Info(fmt.Sprintf("Spec is changed, current generation is %v and observed is %v", serviceBinding.Generation, serviceBinding.Status.ObservedGeneration))
	operationType := smTypes.CREATE

	log.Info("service instance name " + serviceBinding.Spec.ServiceInstanceName + " binding namespace " + serviceBinding.Namespace)
	serviceInstance, err := r.getServiceInstanceForBinding(ctx, serviceBinding)
	if err != nil {
		log.Error(err, fmt.Sprintf("Unable to find referenced service instance with k8s name %s", serviceBinding.Spec.ServiceInstanceName))

		setFailureConditions(operationType,
			fmt.Sprintf("Unable to find referenced service instance with k8s name %s in namespace %s", serviceBinding.Spec.ServiceInstanceName, serviceBinding.Namespace),
			serviceBinding)
		if err := r.Status().Update(ctx, serviceBinding); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	if serviceInProgress(serviceInstance) {
		log.Info(fmt.Sprintf("Service instance with k8s name %s is not ready for binding yet", serviceInstance.Name))

		setInProgressCondition(operationType,
			fmt.Sprintf("Referenced service instance with k8s name %s is not ready, waiting", serviceBinding.Spec.ServiceInstanceName),
			serviceBinding)
		if err := r.Status().Update(ctx, serviceBinding); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true, RequeueAfter: config.Get().SyncPeriod}, nil
	} else if serviceNotUsable(serviceInstance) {
		err := fmt.Errorf("service instance %s is not usable, unable to create binding %s", serviceBinding.Spec.ServiceInstanceName, serviceBinding.Name)
		log.Error(err, fmt.Sprintf("Unable to create binding for instance %s", serviceBinding.Spec.ServiceInstanceName))

		updated := setFailureConditions(operationType, err.Error(), serviceBinding)
		if updated {
			if err := r.Status().Update(ctx, serviceBinding); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true, RequeueAfter: config.Get().SyncPeriod}, nil
	}

	if serviceBinding.Status.BindingID == "" {
		log.Info("Binding ID is empty, checking if exist in SM")

		smClient, err := r.getSMClient(ctx, log)
		if err != nil {
			return ctrl.Result{}, err
		}
		parameters := smclient.Parameters{
			FieldQuery: []string{
				fmt.Sprintf("name eq '%s'", serviceBinding.Spec.ExternalName),
				fmt.Sprintf("service_instance_id eq '%s'", serviceInstance.Status.InstanceID)},
			LabelQuery: []string{
				fmt.Sprintf("_clusterid eq '%s'", "some-cluster-id"),
				fmt.Sprintf("_namespace eq '%s'", serviceBinding.Namespace),
				fmt.Sprintf("_k8sname eq '%s'", serviceBinding.Name)},
			GeneralParams: []string{"attach_last_operations=true"},
		}

		bindings, err := smClient.ListBindings(&parameters)
		if err != nil {
			log.Error(err, "failed to list bindings in SM")
			return ctrl.Result{Requeue: true, RequeueAfter: config.Get().SyncPeriod}, nil
		}
		if bindings != nil && len(bindings.ServiceBindings) == 1 {

			// Restore binding from SM

			log.Info(fmt.Sprintf("found existing smBinding in SM with id %s, updating status", bindings.ServiceBindings[0].ID))
			if err := r.SetOwner(ctx, serviceInstance, serviceBinding, log); err != nil {
				return ctrl.Result{}, err
			}
			if err := r.storeBindingSecret(ctx, serviceBinding, &bindings.ServiceBindings[0], log); err != nil {
				setFailureConditions(smTypes.CREATE, err.Error(), serviceBinding)
				return ctrl.Result{}, err
			}
			r.resyncBindingStatus(serviceBinding, bindings.ServiceBindings[0], serviceInstance)
			if err := r.Status().Update(ctx, serviceBinding); err != nil {
				log.Error(err, "unable to update ServiceBinding status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}

		log.Info("Creating smBinding in SM")
		labels := make(map[string][]string, 3)

		// add labels that can be used to construct OSB context in SM
		labels["_namespace"] = []string{serviceInstance.Namespace}
		labels["_k8sname"] = []string{serviceBinding.Name}
		labels["_clusterid"] = []string{"some-cluster-id"} //TODO maintain a stable cluster ID to pass in context

		bindingParameters, err := getBindingParameters(serviceBinding)
		if err != nil {
			log.Error(err, "failed to parse smBinding parameters")
			return ctrl.Result{}, err
		}

		smBinding, operationURL, err := smClient.Bind(&smclientTypes.ServiceBinding{
			Name:              serviceBinding.Spec.ExternalName,
			Labels:            labels,
			ServiceInstanceID: serviceInstance.Status.InstanceID,
			Parameters:        bindingParameters,
		}, nil)

		serviceBinding.Status.InstanceID = serviceInstance.Status.InstanceID
		log.Info(fmt.Sprintf("Updating observed generation (%v) to generation (%v)", serviceBinding.Status.ObservedGeneration, serviceBinding.Generation))
		serviceBinding.Status.ObservedGeneration = serviceBinding.Generation

		if err := r.SetOwner(ctx, serviceInstance, serviceBinding, log); err != nil {
			setFailureConditions(smTypes.CREATE, "", serviceBinding)
			if err := r.Status().Update(ctx, serviceBinding); err != nil {
				log.Error(err, "unable to update ServiceBinding status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}

		if err != nil {
			log.Error(err, "failed to create smBinding", "serviceInstanceID", serviceInstance.Status.InstanceID)
			setFailureConditions(smTypes.CREATE, err.Error(), serviceBinding)
			if err := r.Status().Update(ctx, serviceBinding); err != nil {
				log.Error(err, "unable to update ServiceBinding status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		if operationURL != "" {
			log.Info("Create smBinding request is async")
			serviceBinding.Status.OperationURL = operationURL
			serviceBinding.Status.OperationType = smTypes.CREATE
			setInProgressCondition(smTypes.CREATE, "", serviceBinding)
			serviceBinding.Status.BindingID = smclient.ExtractBindingID(operationURL)
			if err := r.Status().Update(ctx, serviceBinding); err != nil {
				log.Error(err, "unable to update ServiceBinding status")
				return ctrl.Result{}, err
			}
			if serviceBinding.Status.BindingID == "" {
				return ctrl.Result{}, fmt.Errorf("failed to extract smBinding ID from operation URL %s", operationURL)
			}

			return ctrl.Result{Requeue: true, RequeueAfter: config.Get().PollInterval}, nil
		} else {
			log.Info("Binding created successfully")

			if err := r.storeBindingSecret(ctx, serviceBinding, smBinding, log); err != nil {
				log.Error(err, "failed to create secret")
				setFailureConditions(smTypes.CREATE, err.Error(), serviceBinding)
				if err := r.Status().Update(ctx, serviceBinding); err != nil {
					log.Error(err, "unable to update ServiceBinding status")
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, err
			}

			setSuccessConditions(smTypes.CREATE, serviceBinding)
			serviceBinding.Status.BindingID = smBinding.ID
			log.Info("Updating binding", "bindingID", smBinding.ID)
			if err := r.Status().Update(ctx, serviceBinding); err != nil {
				log.Error(err, "unable to update ServiceBinding status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}

	}

	return ctrl.Result{}, nil
}

func (r *ServiceBindingReconciler) SetOwner(ctx context.Context, serviceInstance *v1alpha1.ServiceInstance, serviceBinding *v1alpha1.ServiceBinding, log logr.Logger) error {
	if err := controllerutil.SetControllerReference(serviceInstance, serviceBinding, r.Scheme); err != nil {
		log.Error(err, fmt.Sprintf("Could not update the smBinding %s owner instance reference", serviceBinding.Name))
		return err
	}
	if err := r.Update(ctx, serviceBinding); err != nil {
		log.Error(err, "Failed to set controller reference", "bindingName", serviceBinding.Name)
		return err
	}
	return nil
}

func serviceNotUsable(instance *v1alpha1.ServiceInstance) bool {
	return isDelete(instance.ObjectMeta) || instance.Status.Conditions[0].Reason == "createFailed"
}

func serviceInProgress(instance *v1alpha1.ServiceInstance) bool {
	if instance.Status.InstanceID != "" && len(instance.Status.Conditions) == 1 && instance.Status.Conditions[0].Status == v1alpha1.ConditionFalse {
		// instance is in progress
		return true
	}

	if instance.Status.InstanceID == "" && len(instance.Status.Conditions) == 0 {
		// instance waiting for recovery
		return true
	}

	return false
}

func (r *ServiceBindingReconciler) getServiceInstanceForBinding(ctx context.Context, binding *v1alpha1.ServiceBinding) (*v1alpha1.ServiceInstance, error) {
	serviceInstance := &v1alpha1.ServiceInstance{}
	if err := r.Get(ctx, types.NamespacedName{Name: binding.Spec.ServiceInstanceName, Namespace: binding.Namespace}, serviceInstance); err != nil {
		return nil, err
	}

	return serviceInstance, nil
}

func (r *ServiceBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ServiceBinding{}).
		Complete(r)
}

func (r *ServiceBindingReconciler) getSMClient(ctx context.Context, log logr.Logger) (smclient.Client, error) {
	if r.SMClient != nil {
		return r.SMClient, nil
	} else {
		return getSMClient(ctx, r, log)
	}
}

func (r *ServiceBindingReconciler) removeFinalizer(ctx context.Context, binding *v1alpha1.ServiceBinding, log logr.Logger) error {
	log.Info("removing finalizer")
	binding.Finalizers = removeString(binding.Finalizers, bindingFinalizerName)
	if err := r.Update(ctx, binding); err != nil {
		return err
	}
	return nil
}

func (r *ServiceBindingReconciler) addFinalizer(ctx context.Context, binding *v1alpha1.ServiceBinding) error {
	binding.Finalizers = append(binding.Finalizers, bindingFinalizerName)
	if err := r.Update(ctx, binding); err != nil {
		return err
	}
	return nil
}

func (r *ServiceBindingReconciler) resyncBindingStatus(k8sBinding *v1alpha1.ServiceBinding, smBinding smclientTypes.ServiceBinding, serviceInstance *v1alpha1.ServiceInstance) {
	//set observed generation to 0 because we dont know which generation the current state in SM represents
	k8sBinding.Status.ObservedGeneration = 0
	k8sBinding.Status.BindingID = smBinding.ID
	k8sBinding.Status.InstanceID = serviceInstance.Status.InstanceID
	switch smBinding.LastOperation.State {
	case smTypes.PENDING:
		fallthrough
	case smTypes.IN_PROGRESS:
		k8sBinding.Status.OperationURL = buildOperationURL(smBinding.LastOperation.ID, smBinding.ID, web.ServiceInstancesURL)
		k8sBinding.Status.OperationType = smBinding.LastOperation.Type
		setInProgressCondition(smBinding.LastOperation.Type, smBinding.LastOperation.Description, k8sBinding)
	case smTypes.SUCCEEDED:
		setSuccessConditions(smBinding.LastOperation.Type, k8sBinding)
	case smTypes.FAILED:
		setFailureConditions(smBinding.LastOperation.Type, smBinding.LastOperation.Description, k8sBinding)
	}
}

func (r *ServiceBindingReconciler) storeBindingSecret(ctx context.Context, k8sBinding *v1alpha1.ServiceBinding, smBinding *smclientTypes.ServiceBinding, log logr.Logger) error {
	logger := log.WithValues("bindingName", k8sBinding.Name, "secretName", k8sBinding.Name)

	credentialsMap, err := normalizeCredentials(smBinding.Credentials)
	if err != nil {
		logger.Error(err, "Failed to store binding secret")
		return fmt.Errorf("failed to store binding secret: %s", err.Error())
	}

	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name: k8sBinding.Name,
			// TODO annotations? labels?
			Namespace: k8sBinding.Namespace,
		},
		Data: credentialsMap,
	}
	if err := controllerutil.SetControllerReference(k8sBinding, secret, r.Scheme); err != nil {
		logger.Error(err, "Failed to set secret owner")
		return err
	}
	if err := r.Create(ctx, secret); err != nil {
		logger.Error(err, "Failed to store binding secret")
		return err
	}
	k8sBinding.Status.SecretName = k8sBinding.Name
	return nil
}

func (r *ServiceBindingReconciler) deleteBindingSecret(ctx context.Context, binding *v1alpha1.ServiceBinding, log logr.Logger) error {
	bindingSecert := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: binding.Namespace,
		Name:      binding.Status.SecretName,
	}, bindingSecert); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch binding secret")
			return err
		} else {
			// secret not found, nothing more to do
			return nil
		}
	}

	if err := r.Delete(ctx, bindingSecert); err != nil {
		log.Error(err, "Failed to delete binding secret")
		return err
	}

	return nil
}

func getBindingParameters(binding *v1alpha1.ServiceBinding) (json.RawMessage, error) {
	var instanceParameters json.RawMessage
	if binding.Spec.Parameters != nil {
		parametersJSON, err := binding.Spec.Parameters.MarshalJSON()
		if err != nil {
			return nil, err
		}
		instanceParameters = parametersJSON
	}
	return instanceParameters, nil
}
