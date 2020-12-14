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
	servicesv1alpha1 "github.com/sm-operator/sapcp-operator/api/v1alpha1"
	"github.com/sm-operator/sapcp-operator/internal/config"
	"github.com/sm-operator/sapcp-operator/internal/secrets"
	"github.com/sm-operator/sapcp-operator/internal/smclient"
	"github.com/sm-operator/sapcp-operator/internal/smclient/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	types2 "k8s.io/apimachinery/pkg/types"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const instanceFinalizerName string = "storage.finalizers.peripli.io.service-manager.serviceInstance"

// ServiceInstanceReconciler reconciles a ServiceInstance object
type ServiceInstanceReconciler struct {
	client.Client
	Log            logr.Logger
	Scheme         *runtime.Scheme
	SMClient       func() smclient.Client
	Config         config.Config
	SecretResolver *secrets.SecretResolver
}

// +kubebuilder:rbac:groups=services.cloud.sap.com,resources=serviceinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=services.cloud.sap.com,resources=serviceinstances/status,verbs=get;update;patch

func (r *ServiceInstanceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("serviceinstance", req.NamespacedName)

	serviceInstance := &servicesv1alpha1.ServiceInstance{}
	if err := r.Get(ctx, req.NamespacedName, serviceInstance); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch ServiceInstance")
		}
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if len(serviceInstance.Status.OperationURL) > 0 {
		// ongoing operation - poll status from SM
		log.Info(fmt.Sprintf("resource is in progress, found operation url %s", serviceInstance.Status.OperationURL))
		smClient, err := r.getSMClient(ctx, log, serviceInstance)
		if err != nil {
			setFailureConditions(serviceInstance.Status.OperationType, fmt.Sprintf("failed to create service-manager client: %s", err.Error()), serviceInstance)
			if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}

		status, err := smClient.Status(serviceInstance.Status.OperationURL, nil)
		if err != nil {
			log.Error(err, "failed to fetch operation", "operationURL", serviceInstance.Status.OperationURL)
			if smErr, ok := err.(*smclient.ServiceManagerError); ok && smErr.StatusCode == http.StatusNotFound {
				log.Info(fmt.Sprintf("Operation %s does not exist in SM, resyncing..", serviceInstance.Status.OperationURL))
				smInstance, err := smClient.GetInstanceByID(serviceInstance.Status.InstanceID, nil)
				if err != nil {
					log.Error(err, fmt.Sprintf("unable to get ServiceInstance with id %s from SM", serviceInstance.Status.InstanceID))
					return ctrl.Result{}, err
				}
				r.resyncInstanceStatus(serviceInstance, *smInstance)
				if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
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
			setFailureConditions(smTypes.OperationCategory(status.Type), status.Description, serviceInstance)
			if serviceInstance.Status.OperationType == smTypes.DELETE {
				serviceInstance.Status.OperationURL = ""
				serviceInstance.Status.OperationType = ""
				if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
					return ctrl.Result{}, err
				}
				errMsg := "Async deprovision operation failed"
				if status.Errors != nil {
					errMsg = fmt.Sprintf("Async deprovision operation failed, errors: %s", string(status.Errors))
				}
				return ctrl.Result{}, fmt.Errorf(errMsg)
			}
		case string(smTypes.SUCCEEDED):
			setSuccessConditions(smTypes.OperationCategory(status.Type), serviceInstance)
			if serviceInstance.Status.OperationType == smTypes.DELETE {
				// delete was successful - remove our finalizer from the list and update it.
				if err = r.removeFinalizer(ctx, serviceInstance, log); err != nil {
					return ctrl.Result{}, err
				}
			}
		}

		serviceInstance.Status.OperationURL = ""
		serviceInstance.Status.OperationType = ""

		if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
			return ctrl.Result{}, err
		}
	}

	if isDelete(serviceInstance.ObjectMeta) {
		if containsString(serviceInstance.ObjectMeta.Finalizers, instanceFinalizerName) {
			if len(serviceInstance.Status.InstanceID) == 0 {
				log.Info("instance does not exists in SM, removing finalizer")
				if err := r.removeFinalizer(ctx, serviceInstance, log); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, nil
			}

			// our finalizer is present, so we need to delete the instance in SM
			smClient, err := r.getSMClient(ctx, log, serviceInstance)
			if err != nil {
				setFailureConditions(smTypes.DELETE, fmt.Sprintf("failed to create service-manager client: %s", err.Error()), serviceInstance)
				if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, err
			}

			log.Info(fmt.Sprintf("Deleting instance with id %v from SM", serviceInstance.Status.InstanceID))
			operationURL, err := smClient.Deprovision(serviceInstance.Status.InstanceID, nil)
			if err != nil {
				smError, isSMError := err.(*smclient.ServiceManagerError)
				if isSMError {
					if smError.StatusCode == http.StatusNotFound {
						log.Info(fmt.Sprintf("instance id %s not found in SM", serviceInstance.Status.InstanceID))
						//if not found it means success
						serviceInstance.Status.InstanceID = ""
						setSuccessConditions(smTypes.DELETE, serviceInstance)
						if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
							return ctrl.Result{}, err
						}

						// remove our finalizer from the list and update it.
						if err := r.removeFinalizer(ctx, serviceInstance, log); err != nil {
							return ctrl.Result{}, err
						}

						return ctrl.Result{}, err
					} else if smError.StatusCode == http.StatusTooManyRequests {
						setInProgressCondition(smTypes.DELETE, fmt.Sprintf("Reached SM api call treshold, will try again in %d seconds", r.Config.LongPollInterval/1000), serviceInstance)
						r.updateStatus(ctx, serviceInstance, log)
						return ctrl.Result{Requeue: true, RequeueAfter: r.Config.LongPollInterval}, nil
					}
				}

				log.Error(err, "failed to delete instance")
				// if fail to delete the instance in SM, return with error
				// so that it can be retried
				if setFailureConditions(smTypes.DELETE, err.Error(), serviceInstance) {
					if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
						return ctrl.Result{}, err
					}
				}

				return ctrl.Result{}, err
			}

			if operationURL != "" {
				log.Info("Deleting instance async")
				serviceInstance.Status.OperationURL = operationURL
				serviceInstance.Status.OperationType = smTypes.DELETE
				setInProgressCondition(smTypes.DELETE, "", serviceInstance)

				if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
					return ctrl.Result{}, err
				}

				return ctrl.Result{Requeue: true, RequeueAfter: r.Config.PollInterval}, nil
			}
			log.Info("Instance was deleted successfully")
			serviceInstance.Status.InstanceID = ""
			setSuccessConditions(smTypes.DELETE, serviceInstance)
			if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			if err := r.removeFinalizer(ctx, serviceInstance, log); err != nil {
				return ctrl.Result{}, err
			}

			// Stop reconciliation as the item is being deleted
			return ctrl.Result{}, nil

		}
	} else {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(serviceInstance.ObjectMeta.Finalizers, instanceFinalizerName) {
			log.Info("instance has no finalizer, adding it...")
			if err := r.addFinalizer(ctx, serviceInstance, log); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if serviceInstance.Generation == serviceInstance.Status.ObservedGeneration {
		log.Info(fmt.Sprintf("Spec is not changed - ignoring... Generation is - %v", serviceInstance.Generation))
		return ctrl.Result{}, nil
	}

	log.Info(fmt.Sprintf("Spec is changed, current generation is %v and observed is %v", serviceInstance.Generation, serviceInstance.Status.ObservedGeneration))
	if serviceInstance.Status.InstanceID == "" {
		log.Info("Instance ID is empty, checking if instance exist in SM")

		if len(serviceInstance.Spec.ExternalName) == 0 {
			serviceInstance.Spec.ExternalName = serviceInstance.Name
		}

		smClient, err := r.getSMClient(ctx, log, serviceInstance)
		if err != nil {
			setFailureConditions(smTypes.CREATE, fmt.Sprintf("failed to create service-manager client: %s", err.Error()), serviceInstance)
			if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}
		parameters := smclient.Parameters{
			FieldQuery: []string{
				fmt.Sprintf("name eq '%s'", serviceInstance.Spec.ExternalName),
				fmt.Sprintf("service_plan_id eq '%s'", serviceInstance.Spec.ServicePlanID)},
			LabelQuery: []string{
				fmt.Sprintf("_clusterid eq '%s'", r.Config.ClusterID),
				fmt.Sprintf("_namespace eq '%s'", serviceInstance.Namespace),
				fmt.Sprintf("_k8sname eq '%s'", serviceInstance.Name)},
			GeneralParams: []string{"attach_last_operations=true"},
		}

		instances, err := smClient.ListInstances(&parameters)
		if err != nil {
			log.Error(err, "failed to list instances in SM")
			return ctrl.Result{Requeue: true, RequeueAfter: r.Config.SyncPeriod}, nil
		}
		if instances != nil && len(instances.ServiceInstances) == 1 {
			log.Info(fmt.Sprintf("found existing instance in SM with id %s, updating status", instances.ServiceInstances[0].ID))
			r.resyncInstanceStatus(serviceInstance, instances.ServiceInstances[0])
			if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}

		log.Info("Creating instance in SM")
		labels := make(map[string][]string, 3)

		// add labels that can be used to construct OSB context in SM
		labels["_namespace"] = []string{serviceInstance.Namespace}
		labels["_k8sname"] = []string{serviceInstance.Name}
		labels["_clusterid"] = []string{r.Config.ClusterID}

		serviceInstance.Status.ObservedGeneration = serviceInstance.Generation

		instanceParameters, err := getInstanceParameters(serviceInstance)
		if err != nil {
			log.Error(err, "failed to parse instance parameters")
			return ctrl.Result{}, err
		}

		smInstanceID, operationURL, err := smClient.Provision(&types.ServiceInstance{
			Name:          serviceInstance.Spec.ExternalName,
			ServicePlanID: serviceInstance.Spec.ServicePlanID,
			Labels:        labels,
			Parameters:    instanceParameters,
		}, serviceInstance.Spec.ServiceOfferingName, serviceInstance.Spec.ServicePlanName, nil)

		if err != nil {
			log.Error(err, "failed to create service instance", "servicePlanID", serviceInstance.Spec.ServicePlanID)
			setFailureConditions(smTypes.CREATE, err.Error(), serviceInstance)
			if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
				log.Error(err, "unable to update ServiceInstance status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		if operationURL != "" {
			log.Info("Provision request is async")
			serviceInstance.Status.OperationURL = operationURL
			serviceInstance.Status.OperationType = smTypes.CREATE
			setInProgressCondition(smTypes.CREATE, "", serviceInstance)
			serviceInstance.Status.InstanceID = smInstanceID
			if serviceInstance.Status.InstanceID == "" {
				return ctrl.Result{}, fmt.Errorf("failed to extract instance ID from operation URL %s", operationURL)
			}

			if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true, RequeueAfter: r.Config.PollInterval}, nil
		}
		log.Info("Instance provisioned successfully")
		setSuccessConditions(smTypes.CREATE, serviceInstance)
		serviceInstance.Status.InstanceID = smInstanceID
		if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	log.Info(fmt.Sprintf("Updating instance with ID %s", serviceInstance.Status.InstanceID))
	smClient, err := r.getSMClient(ctx, log, serviceInstance)
	if err != nil {
		setFailureConditions(smTypes.CREATE, fmt.Sprintf("failed to create service-manager client: %s", err.Error()), serviceInstance)
		if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	serviceInstance.Status.ObservedGeneration = serviceInstance.Generation

	instanceParameters, err := getInstanceParameters(serviceInstance)
	if err != nil {
		log.Error(err, "failed to parse instance parameters")
		return ctrl.Result{}, err
	}

	_, operationURL, err := smClient.UpdateInstance(serviceInstance.Status.InstanceID, &types.ServiceInstance{
		Name:          serviceInstance.Spec.ExternalName,
		ServicePlanID: serviceInstance.Spec.ServicePlanID,
		Parameters:    instanceParameters,
		// TODO labels
	}, nil)
	if err != nil {
		log.Error(err, fmt.Sprintf("failed to update service instance with ID %s", serviceInstance.Status.InstanceID))
		setFailureConditions(smTypes.UPDATE, err.Error(), serviceInstance)

		if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	if operationURL != "" {
		log.Info(fmt.Sprintf("Update request accepted, operation URL: %s", operationURL))
		serviceInstance.Status.OperationURL = operationURL
		serviceInstance.Status.OperationType = smTypes.UPDATE
		setInProgressCondition(smTypes.UPDATE, "", serviceInstance)

		if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true, RequeueAfter: r.Config.PollInterval}, nil
	}
	log.Info("Instance updated successfully")
	setSuccessConditions(smTypes.UPDATE, serviceInstance)
	if err := r.updateStatus(ctx, serviceInstance, log); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ServiceInstanceReconciler) resyncInstanceStatus(k8sInstance *servicesv1alpha1.ServiceInstance, smInstance types.ServiceInstance) {
	//set observed generation to 0 because we dont know which generation the current state in SM represents
	k8sInstance.Status.ObservedGeneration = 0
	k8sInstance.Status.InstanceID = smInstance.ID
	k8sInstance.Status.OperationURL = ""
	k8sInstance.Status.OperationType = ""
	switch smInstance.LastOperation.State {
	case smTypes.PENDING:
		fallthrough
	case smTypes.IN_PROGRESS:
		k8sInstance.Status.OperationURL = buildOperationURL(smInstance.LastOperation.ID, smInstance.ID, web.ServiceInstancesURL)
		k8sInstance.Status.OperationType = smInstance.LastOperation.Type
		setInProgressCondition(smInstance.LastOperation.Type, smInstance.LastOperation.Description, k8sInstance)
	case smTypes.SUCCEEDED:
		setSuccessConditions(smInstance.LastOperation.Type, k8sInstance)
	case smTypes.FAILED:
		setFailureConditions(smInstance.LastOperation.Type, smInstance.LastOperation.Description, k8sInstance)
	}
}

func (r *ServiceInstanceReconciler) removeFinalizer(ctx context.Context, serviceInstance *servicesv1alpha1.ServiceInstance, log logr.Logger) error {
	log.Info("removing finalizer")
	if err := r.Get(ctx, types2.NamespacedName{Name: serviceInstance.Name, Namespace: serviceInstance.Namespace}, serviceInstance); err != nil {
		log.Error(err, "failed to fetch latest service instance")
		return err
	}
	serviceInstance.ObjectMeta.Finalizers = removeString(serviceInstance.ObjectMeta.Finalizers, instanceFinalizerName)
	if err := r.Update(ctx, serviceInstance); err != nil {
		log.Error(err, "failed to remove finalizer")
		return err
	}
	return nil
}

func (r *ServiceInstanceReconciler) addFinalizer(ctx context.Context, serviceInstance *servicesv1alpha1.ServiceInstance, log logr.Logger) error {
	if err := r.Get(ctx, types2.NamespacedName{Name: serviceInstance.Name, Namespace: serviceInstance.Namespace}, serviceInstance); err != nil {
		log.Error(err, "failed to fetch latest service instance")
		return err
	}
	serviceInstance.ObjectMeta.Finalizers = append(serviceInstance.ObjectMeta.Finalizers, instanceFinalizerName)
	if err := r.Update(ctx, serviceInstance); err != nil {
		return err
	}
	return nil
}

func (r *ServiceInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&servicesv1alpha1.ServiceInstance{}).
		Complete(r)
}

func (r *ServiceInstanceReconciler) getSMClient(ctx context.Context, log logr.Logger, instance *servicesv1alpha1.ServiceInstance) (smclient.Client, error) {
	if r.SMClient != nil {
		return r.SMClient(), nil
	}

	secret, err := r.SecretResolver.GetSecretForResource(ctx, instance)
	if err != nil {
		return nil, err
	}

	return getSMClient(ctx, secret, log)
}

func getInstanceParameters(serviceInstance *servicesv1alpha1.ServiceInstance) (json.RawMessage, error) {
	var instanceParameters json.RawMessage
	if serviceInstance.Spec.Parameters != nil {
		parametersJSON, err := serviceInstance.Spec.Parameters.MarshalJSON()
		if err != nil {
			return nil, err
		}
		instanceParameters = parametersJSON
	}
	return instanceParameters, nil
}

func (r *ServiceInstanceReconciler) updateStatus(ctx context.Context, serviceInstance *servicesv1alpha1.ServiceInstance, log logr.Logger) error {
	log.Info("updating service instance status")
	if err := r.Status().Update(ctx, serviceInstance); err != nil {
		log.Info(fmt.Sprintf("failed to update status - %s, fetching latest instance and trying again", err.Error()))
		latest := &servicesv1alpha1.ServiceInstance{}
		if err := r.Get(ctx, types2.NamespacedName{Name: serviceInstance.Name, Namespace: serviceInstance.Namespace}, latest); err != nil {
			log.Error(err, "failed to fetch latest instance")
			return err
		}

		latest.Status = serviceInstance.Status
		if err := r.Status().Update(ctx, latest); err != nil {
			log.Error(err, "unable to update service instance status")
			return err
		}
		serviceInstance = latest
	}
	log.Info("updated ServiceInstance status in k8s")
	return nil
}
