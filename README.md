[![Coverage Status](https://coveralls.io/repos/github/sm-operator/sapcp-operator/badge.svg?branch=master)](https://coveralls.io/github/sm-operator/sapcp-operator?branch=master)
[![Build Status](https://github.com/sm-operator/sapcp-operator/workflows/Go/badge.svg)](https://github.com/sm-operator/sapcp-operator/actions)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/sm-operator/sapcp-operator/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/sm-operator/sapcp-operator)](https://goreportcard.com/report/github.com/sm-operator/sapcp-operator)

# SAPCP Operator


With the SAPCP Operator, you can provision and bind SAPCP services to your Kubernetes cluster in a Kubernetes-native way. The SAPCP Operator is based on the [Kubernetes custom resource definition (CRD) API](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) so that your applications can create, update, and delete SAPCP services from within the cluster by calling Kubnernetes APIs.

## Table of content
* [Prerequisites](#prerequisites)
* [Setup Operator](#setup)
* [Local Setup](#local-setup)
* [SAPCP kubectl extension](#sapcp-kubectl-extension-experimental)
* [Using the SAPCP Operator](#using-the-sapcp-operator)
    * [Creating a service instance](#step-1-creating-a-service-instance)
    * [Binding the service instance](#step-2-binding-the-service-instance)
* [Reference documentation](#reference-documentation)
    * [Service instance properties](#service-instance-properties)
    * [Binding properties](#binding-properties)    

## Prerequisites
- [kubernetes cluster](https://kubernetes.io/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [helm](https://helm.sh/)
</br></br>
[Back to top](#sapcp-operator)

## Setup
1. Install [cert-manager](https://cert-manager.io/docs/installation/kubernetes):
    ```
    kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.1.0/cert-manager.yaml
    ```

2. Deploy the sapcp-operator in the cluster:
    ```
    helm upgrade --install sapcp-operator https://github.com/sm-operator/sapcp-operator/releases/download/${release}/sapcp-operator-${release}.tgz \
        --create-namespace \
        --namespace=sapcp-operator \
        --set manager.secret.clientid=$clientid \
        --set manager.secret.clientsecret=$clientsecret \
        --set manager.secret.url=$url \
        --set manager.secret.tokenurl=$tokenurl
    ```

    The list of available releases is available here: [sapcp-operator releases](https://github.com/sm-operator/sapcp-operator/releases)
</br></br>
[Back to top](#sapcp-operator)

## Local setup
### Prerequisites
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/)

### Deploy locally
Edit [manager secret](hack/override_values.yaml) section with SM credentials. (DO NOT SUBMIT)
```
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.1.0/cert-manager.yaml
make docker-build
kind load docker-image controller:latest
make deploy
```
### Run tests
`make test`
</br></br>
[Back to top](#sapcp-operator)

## SAPCP kubectl extension (experimental) 
### Prerequisites
- [jq](https://stedolan.github.io/jq/)

Download https://github.com/sm-operator/sapcp-operator/releases/download/${release}/kubectl-sapcp
move its executable file to anywhere on your ``PATH``

#### Usage
 namespace parameter indicates the namespace where to find SM secret, defaulting to default namespace 
```
  kubectl sapcp marketplace -n <namespace>
  kubectl sapcp plans -n <namespace>
  kubectl sapcp services -n <namespace>
```
[Back to top](#sapcp-operator)
## Using the SAPCP Operator

#### Step 1: Creating a service instance

1.  To create an instance of SAPCP, first create a `ServiceInstance` custom resource file.
   *   `<serviceOfferingName>` is the SAPCP service that you want to create. To list SAPCP marketplace, run `kubectl sapcp marketplace`.
   *   `<servicePlanName>` is the plan for the SAPCP service that you want to create.

```yaml
    apiVersion: services.cloud.sap.com/v1alpha1
    kind: ServiceInstance
    metadata:
        name: myservice
    spec:
        servicePlanName: <PLAN>
        serviceOfferingName: <SERVICE_CLASS>
   ```

2.  Create the service instance in your cluster.

    ```bash
    kubectl apply -f filepath/myservice.yaml
    ```

3.  Check that your service status is **Created** in your cluster.

    ```bash
    kubectl get serviceinstances
    NAME        STATUS   AGE
    myservice   Created  12s
    ```
[Back to top](#sapcp-operator)

#### Step 2: Binding the service instance

1.  To bind your service to the cluster so that your apps can use the service, create a `ServiceBinding` custom resource, where the `serviceInstanceName` field is the name of the `ServiceInstance` custom resource that you previously created.

    ```yaml
    apiVersion: services.cloud.sap.com/v1alpha1
    kind: ServiceBinding
    metadata:
        name: mybinding
    spec:
        serviceInstanceName: myservice
    ```

2.  Create the binding in your cluster.

    ```bash
    kubectl apply -f filepath/mybinding.yaml
    ```

3.  Check that your binding status is **Created**.

    ```bash
    kubectl get servicebindings
    NAME        INSTANCE    STATUS    AGE
    mybinding   myservice   Created   9s
    
    ```

4.  Check that a secret of the same name as your binding is created. The secret contains the service credentials that apps in your cluster can use to access the service.

    ```bash
    kubectl get secrets
    NAME        TYPE     DATA   AGE
    mybinding   Opaque   5      102s
    ```
[Back to top](#sapcp-operator)

## Reference documentation

### Service Instance Properties
| Parameter             | Type       | Comments                                                                                                   |
|:-----------------|:---------|:-----------------------------------------------------------------------------------------------------------|
| serviceOfferingName`*`   | `string`   | SAPCP service offering name |
| servicePlanName`*` | `string`   |  The plan to use for the service instance, such as `free` or `standard`. |
| servicePlanID   |  `string`   |  The plan ID in case service offering and plan name are ambiguous |
| externalName       | `string`   |  The name for the service instance in SAPCP |
| parameters       |  `[]object`  |  Provisioning parameters for the instance |

### Binding Properties
| Parameter             | Type       | Comments                                                                                                   |
|:-----------------|:---------|:-----------------------------------------------------------------------------------------------------------|
| serviceInstanceName`*`   | `string`   | The k8s name of the service instance to bind, should be in the namespace of the binding |
| servicePlanID   |  `string`   |  The plan ID in case service offering and plan name are ambiguous |
| externalName       | `string`   |  The name for the service binding in SAPCP |
| secretName       | `string`   |  The name of the secret where credentials will be stored |
| parameters       |  `[]object`  |  Parameters for the binding |

[Back to top](#sapcp-operator)
