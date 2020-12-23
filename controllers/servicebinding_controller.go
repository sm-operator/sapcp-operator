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
	"fmt"
	"net/http"

	smTypes "github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/go-logr/logr"
	"github.com/sm-operator/sapcp-operator/api/v1alpha1"
	"github.com/sm-operator/sapcp-operator/internal/smclient"
	smclientTypes "github.com/sm-operator/sapcp-operator/internal/smclient/types"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const bindingFinalizerName string = "storage.finalizers.peripli.io.service-manager.serviceBinding"

// ServiceBindingReconciler reconciles a ServiceBinding object
type ServiceBindingReconciler struct {
	*BaseReconciler
}

// +kubebuilder:rbac:groups=services.cloud.sap.com,resources=servicebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=services.cloud.sap.com,resources=servicebindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=services.cloud.sap.com,resources=serviceinstances,verbs=get;list
// +kubebuilder:rbac:groups=services.cloud.sap.com,resources=serviceinstances/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *ServiceBindingReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	// TODO optimize log - use withValue where possible
	log := r.Log.WithValues("servicebinding", req.NamespacedName)

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
		return r.poll(ctx, serviceBinding, log)
	}

	if isDelete(serviceBinding.ObjectMeta) {
		return r.delete(ctx, serviceBinding, log)
	}
	// The object is not being deleted, so if it does not have our finalizer,
	// then lets add the finalizer and update the object. This is equivalent
	// registering our finalizer.
	if !containsString(serviceBinding.ObjectMeta.Finalizers, bindingFinalizerName) {
		log.Info("Binding has no finalizer, adding it...")
		if err := r.addFinalizer(ctx, serviceBinding); err != nil {
			return ctrl.Result{}, err
		}
	}

	if serviceBinding.Generation == serviceBinding.Status.ObservedGeneration {
		log.Info(fmt.Sprintf("Spec is not changed - ignoring... Generation is - %v", serviceBinding.Generation))
		return ctrl.Result{}, nil
	}

	log.Info(fmt.Sprintf("Spec is changed, current generation is %v and observed is %v", serviceBinding.Generation, serviceBinding.Status.ObservedGeneration))

	log.Info("service instance name " + serviceBinding.Spec.ServiceInstanceName + " binding namespace " + serviceBinding.Namespace)
	serviceInstance, err := r.getServiceInstanceForBinding(ctx, serviceBinding)
	if err != nil {
		log.Error(err, fmt.Sprintf("Unable to find referenced service instance with k8s name %s", serviceBinding.Spec.ServiceInstanceName))

		setFailureConditions(smTypes.CREATE,
			fmt.Sprintf("Unable to find referenced service instance with k8s name %s in namespace %s", serviceBinding.Spec.ServiceInstanceName, serviceBinding.Namespace),
			serviceBinding)
		if err := r.Status().Update(ctx, serviceBinding); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	if serviceInProgress(serviceInstance) {
		log.Info(fmt.Sprintf("Service instance with k8s name %s is not ready for binding yet", serviceInstance.Name))

		setInProgressCondition(smTypes.CREATE,
			fmt.Sprintf("Referenced service instance with k8s name %s is not ready, cannot create binding yet", serviceBinding.Spec.ServiceInstanceName),
			serviceBinding)
		if err := r.Status().Update(ctx, serviceBinding); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true, RequeueAfter: r.Config.PollInterval}, nil
	}

	if serviceNotUsable(serviceInstance) {
		err := fmt.Errorf("service instance %s is not usable, unable to create binding %s. Will retry after %s", serviceBinding.Spec.ServiceInstanceName, serviceBinding.Name, r.Config.SyncPeriod.String())
		log.Error(err, fmt.Sprintf("Unable to create binding for instance %s", serviceBinding.Spec.ServiceInstanceName))

		updated := setFailureConditions(smTypes.CREATE, err.Error(), serviceBinding)
		if updated {
			if err := r.Status().Update(ctx, serviceBinding); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true, RequeueAfter: r.Config.SyncPeriod}, nil
	}

	if serviceBinding.Status.BindingID == "" {
		smClient, err := r.getSMClient(ctx, log, serviceBinding.Namespace)
		if err != nil {
			setFailureConditions(smTypes.CREATE, fmt.Sprintf("failed to create service-manager client: %s", err.Error()), serviceBinding)
			if err := r.Status().Update(ctx, serviceBinding); err != nil {
				log.Error(err, "unable to update ServiceBinding status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}
		binding, err := r.getBindingForRecovery(smClient, serviceBinding, log)
		if err != nil {
			log.Error(err, "failed to list bindings from SM")
		}
		if binding != nil {
			// Recovery - restore binding from SM
			log.Info(fmt.Sprintf("found existing smBinding in SM with id %s, updating status", binding.ID))
			if err := r.SetOwner(ctx, serviceInstance, serviceBinding, log); err != nil {
				return ctrl.Result{}, err
			}

			if binding.LastOperation.Type != smTypes.CREATE || binding.LastOperation.State == smTypes.SUCCEEDED {
				// store secret unless binding is still being created or failed during creation
				if err := r.storeBindingSecret(ctx, serviceBinding, binding, log); err != nil {
					setFailureConditions(binding.LastOperation.Type, err.Error(), serviceBinding)
					return ctrl.Result{}, err
				}
			}

			r.resyncBindingStatus(serviceBinding, binding, serviceInstance.Status.InstanceID)
			if err := r.Status().Update(ctx, serviceBinding); err != nil {
				log.Error(err, "unable to update ServiceBinding status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}

		return r.createBinding(ctx, smClient, serviceInstance, serviceBinding, log)
	}

	log.Error(fmt.Errorf("update binding is not allowed, this line should not be reached"), "")
	return ctrl.Result{}, nil
}

func (r *ServiceBindingReconciler) createBinding(ctx context.Context, smClient smclient.Client, serviceInstance *v1alpha1.ServiceInstance, serviceBinding *v1alpha1.ServiceBinding, log logr.Logger) (ctrl.Result, error) {
	log.Info("Creating smBinding in SM")
	labels := make(map[string][]string, 3)

	// add labels that can be used to construct OSB context in SM
	labels[namespaceLabel] = []string{serviceInstance.Namespace}
	labels[k8sNameLabel] = []string{serviceBinding.Name}
	labels[clusterIDLabel] = []string{r.Config.ClusterID}

	bindingParameters, err := getParameters(serviceBinding)
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
		if err := r.updateStatus(ctx, serviceBinding, log); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if err != nil {
		log.Error(err, "failed to create smBinding", "serviceInstanceID", serviceInstance.Status.InstanceID)
		setFailureConditions(smTypes.CREATE, err.Error(), serviceBinding)
		if err := r.updateStatus(ctx, serviceBinding, log); err != nil {
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
		if err := r.updateStatus(ctx, serviceBinding, log); err != nil {
			log.Error(err, "unable to update ServiceBinding status")
			return ctrl.Result{}, err
		}
		if serviceBinding.Status.BindingID == "" {
			return ctrl.Result{}, fmt.Errorf("failed to extract smBinding ID from operation URL %s", operationURL)
		}

		return ctrl.Result{Requeue: true, RequeueAfter: r.Config.PollInterval}, nil
	}

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
	if err := r.updateStatus(ctx, serviceBinding, log); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ServiceBindingReconciler) delete(ctx context.Context, serviceBinding *v1alpha1.ServiceBinding, log logr.Logger) (ctrl.Result, error) {
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
		smClient, err := r.getSMClient(ctx, log, serviceBinding.Namespace)
		if err != nil {
			setFailureConditions(smTypes.DELETE, fmt.Sprintf("failed to create service-manager client: %s", err.Error()), serviceBinding)
			if err := r.updateStatus(ctx, serviceBinding, log); err != nil {
				log.Error(err, "unable to update ServiceBinding status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}

		log.Info(fmt.Sprintf("Deleting binding with id %v from SM", serviceBinding.Status.BindingID))
		operationURL, err := smClient.Unbind(serviceBinding.Status.BindingID, nil)
		if err != nil {
			smError, ok := err.(*smclient.ServiceManagerError)
			if ok {
				if smError.StatusCode == http.StatusNotFound || smError.StatusCode == http.StatusGone {
					log.Info(fmt.Sprintf("Binding id %s not found in SM", serviceBinding.Status.BindingID))
					//if not found it means success
					serviceBinding.Status.BindingID = ""
					setSuccessConditions(smTypes.DELETE, serviceBinding)
					if err := r.updateStatus(ctx, serviceBinding, log); err != nil {
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
				} else if smError.StatusCode == http.StatusTooManyRequests {
					setInProgressCondition(smTypes.DELETE, fmt.Sprintf("Reached SM api call treshold, will try again in %d seconds", r.Config.LongPollInterval/1000), serviceBinding)
					if err := r.updateStatus(ctx, serviceBinding, log); err != nil {
						log.Info("failed to set in progress condition in response to 429 error got from SM, ignoring...")
					}
					return ctrl.Result{Requeue: true, RequeueAfter: r.Config.LongPollInterval}, nil
				}
			}

			log.Error(err, "failed to delete binding")
			// if fail to delete the binding in SM, return with error
			// so that it can be retried
			if setFailureConditions(smTypes.DELETE, err.Error(), serviceBinding) {
				if err := r.updateStatus(ctx, serviceBinding, log); err != nil {
					return ctrl.Result{}, err
				}
			}
			return ctrl.Result{}, err
		}

		if operationURL != "" {
			log.Info("Deleting binding async")
			serviceBinding.Status.OperationURL = operationURL
			serviceBinding.Status.OperationType = smTypes.DELETE
			setInProgressCondition(smTypes.DELETE, "", serviceBinding)
			if err := r.updateStatus(ctx, serviceBinding, log); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true, RequeueAfter: r.Config.PollInterval}, nil
		}

		log.Info("Binding was deleted successfully")
		serviceBinding.Status.BindingID = ""
		setSuccessConditions(smTypes.DELETE, serviceBinding)
		if err := r.updateStatus(ctx, serviceBinding, log); err != nil {
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
	return ctrl.Result{}, nil
}

func (r *ServiceBindingReconciler) poll(ctx context.Context, serviceBinding *v1alpha1.ServiceBinding, log logr.Logger) (ctrl.Result, error) {
	log.Info(fmt.Sprintf("resource is in progress, found operation url %s", serviceBinding.Status.OperationURL))
	smClient, err := r.getSMClient(ctx, log, serviceBinding.Namespace)
	if err != nil {
		setFailureConditions(serviceBinding.Status.OperationType, fmt.Sprintf("failed to create service-manager client: %s", err.Error()), serviceBinding)
		if err := r.updateStatus(ctx, serviceBinding, log); err != nil {
			log.Error(err, "unable to update ServiceBinding status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	status, err := smClient.Status(serviceBinding.Status.OperationURL, nil)
	if err != nil {
		log.Error(err, "failed to fetch operation", "operationURL", serviceBinding.Status.OperationURL)
		if smErr, ok := err.(*smclient.ServiceManagerError); ok && smErr.StatusCode == http.StatusNotFound {
			log.Info(fmt.Sprintf("Operation %s does not exist in SM, resyncing..", serviceBinding.Status.OperationURL))
			smBinding, err := smClient.GetBindingByID(serviceBinding.Status.BindingID, nil)
			if err != nil {
				log.Error(err, fmt.Sprintf("unable to get binding with id %s from SM", serviceBinding.Status.BindingID))
				return ctrl.Result{}, err
			}
			r.resyncBindingStatus(serviceBinding, smBinding, serviceBinding.Status.InstanceID)
			if err := r.updateStatus(ctx, serviceBinding, log); err != nil {
				log.Error(err, "unable to update ServiceBinding status")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	switch status.State {
	case string(smTypes.IN_PROGRESS):
		fallthrough
	case string(smTypes.PENDING):
		return ctrl.Result{Requeue: true, RequeueAfter: r.Config.PollInterval}, nil
	case string(smTypes.FAILED):
		setFailureConditions(smTypes.OperationCategory(status.Type), status.Description, serviceBinding)
		if serviceBinding.Status.OperationType == smTypes.DELETE {
			serviceBinding.Status.OperationURL = ""
			serviceBinding.Status.OperationType = ""
			if err := r.updateStatus(ctx, serviceBinding, log); err != nil {
				log.Error(err, "unable to update ServiceBinding status")
				return ctrl.Result{}, err
			}
			errMsg := "Async unbind operation failed"
			if status.Errors != nil {
				errMsg = fmt.Sprintf("Async unbind operation failed, errors: %s", string(status.Errors))
			}
			return ctrl.Result{}, fmt.Errorf(errMsg)
		}
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
			if !serviceBinding.DeletionTimestamp.IsZero() {
				//if binding is being deleted, remove our finalizer
				if err = r.removeFinalizer(ctx, serviceBinding, log); err != nil {
					return ctrl.Result{}, err
				}
			}
		}
	}

	serviceBinding.Status.OperationURL = ""
	serviceBinding.Status.OperationType = ""

	err = r.updateStatus(ctx, serviceBinding, log)
	return ctrl.Result{}, err
}

func (r *ServiceBindingReconciler) SetOwner(ctx context.Context, serviceInstance *v1alpha1.ServiceInstance, serviceBinding *v1alpha1.ServiceBinding, log logr.Logger) error {
	if bindingAlreadyOwnedByInstance(serviceInstance, serviceBinding) {
		log.Info("Binding already owned by instance", "bindingName", serviceBinding.Name, "instanceName", serviceInstance.Name)
		return nil
	}

	log.Info("Binding instance as owner of binding", "bindingName", serviceBinding.Name, "instanceName", serviceInstance.Name)
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

func bindingAlreadyOwnedByInstance(instance *v1alpha1.ServiceInstance, binding *v1alpha1.ServiceBinding) bool {
	if existing := metav1.GetControllerOf(binding); existing != nil {
		aGV, err := schema.ParseGroupVersion(existing.APIVersion)
		if err != nil {
			return false
		}

		bGV, err := schema.ParseGroupVersion(instance.APIVersion)
		if err != nil {
			return false
		}

		return aGV.Group == bGV.Group && existing.Kind == instance.Kind && existing.Name == instance.Name
	}

	return false
}

func serviceNotUsable(instance *v1alpha1.ServiceInstance) bool {
	return isDelete(instance.ObjectMeta) || instance.Status.Conditions[0].Reason == getConditionReason(smTypes.CREATE, smTypes.FAILED)
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

func (r *ServiceBindingReconciler) removeFinalizer(ctx context.Context, binding *v1alpha1.ServiceBinding, log logr.Logger) error {
	log.Info("removing finalizer")
	binding.Finalizers = removeString(binding.Finalizers, bindingFinalizerName)
	//binding.DeletionTimestamp = &metav1.Time{Time: time.Now()}
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

func (r *ServiceBindingReconciler) resyncBindingStatus(k8sBinding *v1alpha1.ServiceBinding, smBinding *smclientTypes.ServiceBinding, serviceInstanceID string) {
	k8sBinding.Status.ObservedGeneration = k8sBinding.Generation
	k8sBinding.Status.BindingID = smBinding.ID
	k8sBinding.Status.InstanceID = serviceInstanceID
	k8sBinding.Status.OperationURL = ""
	k8sBinding.Status.OperationType = ""
	switch smBinding.LastOperation.State {
	case smTypes.PENDING:
		fallthrough
	case smTypes.IN_PROGRESS:
		k8sBinding.Status.OperationURL = smclient.BuildOperationURL(smBinding.LastOperation.ID, smBinding.ID, web.ServiceBindingsURL)
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

	var credentialsMap map[string][]byte
	if len(smBinding.Credentials) == 0 {
		log.Info("Binding credentials are empty")
		credentialsMap = make(map[string][]byte)
	} else {
		var err error
		credentialsMap, err = normalizeCredentials(smBinding.Credentials)
		if err != nil {
			logger.Error(err, "Failed to store binding secret")
			return fmt.Errorf("failed to store binding secret: %s", err.Error())
		}
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
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

	log.Info("Creating binding secret")
	if err := r.Create(ctx, secret); err != nil {
		logger.Error(err, "Failed to store binding secret")
		return err
	}

	k8sBinding.Status.SecretName = k8sBinding.Name
	return nil
}

func (r *ServiceBindingReconciler) deleteBindingSecret(ctx context.Context, binding *v1alpha1.ServiceBinding, log logr.Logger) error {
	log.Info("Deleting binding secret")
	bindingSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: binding.Namespace,
		Name:      binding.Status.SecretName,
	}, bindingSecret); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch binding secret")
			return err
		}

		// secret not found, nothing more to do
		return nil
	}

	if err := r.Delete(ctx, bindingSecret); err != nil {
		log.Error(err, "Failed to delete binding secret")
		return err
	}

	return nil
}

func (r *ServiceBindingReconciler) getBindingForRecovery(smClient smclient.Client, serviceBinding *v1alpha1.ServiceBinding, log logr.Logger) (*smclientTypes.ServiceBinding, error) {
	parameters := smclient.Parameters{
		FieldQuery: []string{
			fmt.Sprintf("name eq '%s'", serviceBinding.Spec.ExternalName)},
		LabelQuery: []string{
			fmt.Sprintf("%s eq '%s'", clusterIDLabel, r.Config.ClusterID),
			fmt.Sprintf("%s eq '%s'", namespaceLabel, serviceBinding.Namespace),
			fmt.Sprintf("%s eq '%s'", k8sNameLabel, serviceBinding.Name)},
		GeneralParams: []string{"attach_last_operations=true"},
	}

	bindings, err := smClient.ListBindings(&parameters)
	if err != nil {
		log.Error(err, "failed to list bindings in SM")
		return nil, err
	}
	if bindings != nil && len(bindings.ServiceBindings) == 1 {
		return &bindings.ServiceBindings[0], nil
	}

	return nil, nil
}

func (r *ServiceBindingReconciler) updateStatus(ctx context.Context, serviceBinding *v1alpha1.ServiceBinding, log logr.Logger) error {
	log.Info("updating service binding status")
	if err := r.Status().Update(ctx, serviceBinding); err != nil {
		status := serviceBinding.Status
		log.Info(fmt.Sprintf("failed to update status - %s, fetching latest binding and trying again", err.Error()))
		if err := r.Get(ctx, types.NamespacedName{Name: serviceBinding.Name, Namespace: serviceBinding.Namespace}, serviceBinding); err != nil {
			log.Error(err, "failed to fetch latest binding")
			return err
		}

		serviceBinding.Status = status
		if err := r.Status().Update(ctx, serviceBinding); err != nil {
			log.Error(err, "unable to update service binding status")
			return err
		}
	}
	log.Info("updated service binding status in k8s")
	return nil
}
