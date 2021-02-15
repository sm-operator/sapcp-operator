[![Coverage Status](https://coveralls.io/repos/github/sm-operator/sapcp-operator/badge.svg?branch=master&killcache=1)](https://coveralls.io/github/sm-operator/sapcp-operator?branch=master)
[![Build Status](https://github.com/sm-operator/sapcp-operator/workflows/Go/badge.svg)](https://github.com/sm-operator/sapcp-operator/actions)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/sm-operator/sapcp-operator/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/sm-operator/sapcp-operator)](https://goreportcard.com/report/github.com/sm-operator/sapcp-operator)

# SAP Business Technology Platform (SAP BTP) Service Operator


With the SAP BTP Operator, you can manage SAP BTP services from your Kubernetes cluster in a Kubernetes-native way. The SAP BTP Service Operator is based on the [Kubernetes custom resource definition (CRD) API](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) so that your applications can use the CRUD operations in regards to SAP BTP services from within the cluster by calling the Kubernetes APIs.

## Table of Content
* [Prerequisites](#prerequisites)
* [Setting Up Operator](#setting-up-operator)
* [Local Setup](#local-setup)
* [SAP CP kubectl Extension](#sap-cp-kubectl-extension-experimental)
* [Using the SAP BTP Service Operator](#using-the-sap-btp-service-operator)
    * [Creating a Service Instance](#step-1-creating-a-service-instance)
    * [Binding a Service Instance](#step-2-binding-a-service-instance)
* [Reference Documentation](#reference-documentation)
    * [Service Instance Properties](#service-instance-properties)
    * [Service Binding Properties](#service-binding-properties)    

## Prerequisites
- SAP BTP [Global Account](https://help.sap.com/viewer/65de2977205c403bbc107264b8eccf4b/Cloud/en-US/d61c2819034b48e68145c45c36acba6e.html) and [Subaccount](https://help.sap.com/viewer/65de2977205c403bbc107264b8eccf4b/Cloud/en-US/55d0b6d8b96846b8ae93b85194df0944.html) 
- [Kubernetes cluster](https://kubernetes.io/) running version 1.17 or higher 
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) v1.17 or higher
- [helm](https://helm.sh/) v3.0 or higher

[Back to top](#sapcp-operator)

## Setting Up Operator
1. Install [cert-manager](https://cert-manager.io/docs/installation/kubernetes)

1. Obtain access credentials for the SAP BTP Service Operator:
   1. Using SAP BTP Cockpit or CLI, create an instance of the SAP Cloud Service Management service (`service-manager`) service, with the plan `service-operator-access`.
      
      More information about creating service instances is available here: 
      [Cockpit](https://help.sap.com/viewer/09cc82baadc542a688176dce601398de/Cloud/en-US/bf71f6a7b7754dbd9dfc2569791ccc96.html), 
      [CLI](https://help.sap.com/viewer/09cc82baadc542a688176dce601398de/Cloud/en-US/b327b66b711746b085ec5d2ea16e608e.html).  
   
   1. Create a binding to the created service instance.
      
      More information about creating service bindings is available here: 
            [Cockpit](https://help.sap.com/viewer/09cc82baadc542a688176dce601398de/Cloud/en-US/bf71f6a7b7754dbd9dfc2569791ccc96.html), 
            [CLI](https://help.sap.com/viewer/09cc82baadc542a688176dce601398de/Cloud/en-US/f53ff2634e0a46d6bfc72ec075418dcd.html). 
   
   1. Retrieve the generated access credentials from the created binding.
   
       Example:
       ```json
        {
            "clientid": "xxxxxxx",
            "clientsecret": "xxxxxxx",
            "url": "https://mysubaccount.authentication.eu10.hana.ondemand.com",
            "xsappname": "b15166|service-manager!b1234",
            "sm_url": "https://service-manager.cfapps.eu10.hana.ondemand.com"
        }
       ```  
   
1. Deploy the sapcp-service-operator in the Kubernetes cluster using the obtained access credentials:
    ```bash
    helm upgrade --install sapcp-operator https://github.com/sm-operator/sapcp-operator/releases/download/<release>/sapcp-operator-<release>.tgz \
        --create-namespace \
        --namespace=sapcp-operator \
        --set manager.secret.clientid=<clientid> \
        --set manager.secret.clientsecret=<clientsecret> \
        --set manager.secret.url=<sm_url> \
        --set manager.secret.tokenurl=<url>
    ```

    The list of available releases is available here: [sapcp-operator releases](https://github.com/sm-operator/sapcp-operator/releases)

[Back to top](#sapcp-operator)

## Using the SAP BTP Service Operator

#### Step 1: Creating a Service Instance

1.  To create an instance of an SAP BTP service, first create a `ServiceInstance` custom resource file:

```yaml
    apiVersion: services.cloud.sap.com/v1alpha1
    kind: ServiceInstance
    metadata:
        name: my-service-instance
    spec:
        serviceOfferingName: <offering>
        servicePlanName: <plan>
   ```

   *   `<offering>` is the name of the SAP BTP service that you are creating.
       You can find [the list of the available services](https://help.sap.com/viewer/09cc82baadc542a688176dce601398de/Cloud/en-US/55b31ea23c474f6ba2f64ee4848ab1b3.html) in the SAP CP Cockpit.
   *   `<plan>` is the plan of the selected service offering that you are creating.

2.  Apply the custom resource file in your cluster to create the instance.

    ```bash
    kubectl apply -f path/to/my-service-instance.yaml
    ```

3.  Check that your service status is **Created** in your cluster.
    
    //TODO update example output with all fields
    
    ```bash
    kubectl get serviceinstances
    NAME                  STATUS   AGE
    my-service-instance   Created  19s
    ```
[Back to top](#sapcp-operator)

#### Step 2: Binding a Service Instance

1.  To get access credentials to your service instance and make them available in the cluster so that your applications can use it, create a `ServiceBinding` custom resource, and set the `serviceInstanceName` field to the name of the `ServiceInstance` resource you created.

    ```yaml
    apiVersion: services.cloud.sap.com/v1alpha1
    kind: ServiceBinding
    metadata:
        name: my-binding
    spec:
        serviceInstanceName: my-service-instance
    ```

1.  Apply the custom resource file in your cluster to create the binding.

    ```bash
    kubectl apply -f path/to/my-binding.yaml
    ```

1.  Check that your binding status is **Created**.

    ```bash
    kubectl get servicebindings
    NAME         INSTANCE              STATUS    AGE
    my-binding   my-service-instance   Created   16s
    
    ```

1.  Check that a secret with the same name as your binding is created. The secret contains the service credentials that apps in your cluster can use to access the chosen service.

    ```bash
    kubectl get secrets
    NAME         TYPE     DATA   AGE
    my-binding   Opaque   5      32s
    ```
    
    See [Using Secrets](https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets) for the different options to use the credentials for your application running in the Kubernetes cluster, 

[Back to top](#sapcp-operator)

## Reference Documentation

### Service Instance Properties
#### Spec
| Property         | Type     | Comments                                                                                                   |
|:-----------------|:---------|:-----------------------------------------------------------------------------------------------------------|
| serviceOfferingName`*`   | `string`   | The name of the SAP Business Technology Platform (SAP BTP) service offering |
| servicePlanName`*` | `string`   |  The plan to use for the service instance |
| servicePlanID   |  `string`   |  The plan ID in case service offering and plan names are ambiguous |
| externalName       | `string`   |  The name of the service instance in SAP BTP, defaults to the binding `metadata.name` if not specified |
| parameters       |  `[]object`  |  Provisioning parameters for the instance, check the documentation of the specific service you are using for details |

#### Status
| Property         | Type     | Comments                                                                                                   |
|:-----------------|:---------|:-----------------------------------------------------------------------------------------------------------|
| instanceID   | `string`   | The service instance ID in SAP Cloud Service Management service |
| operationURL | `string`   |  URL of the ongoing operation for the service instance |
| operationType   |  `string`   |  The operation type (CREATE/UPDATE/DELETE) of the ongoing operation |
| conditions       | `[]condition`   |  An array of conditions describing the status of the service instance. <br>The possible conditions types are:<br>- `Ready`: set to `true` if the instance is ready and usable<br>- `Failed`: set to `true` when an operation on the service instance fails, in this case the error details are available in the condition message  



### Service Binding Properties
#### Spec
| Parameter             | Type       | Comments                                                                                                   |
|:-----------------|:---------|:-----------------------------------------------------------------------------------------------------------|
| serviceInstanceName`*`   | `string`   | The Kubernetes name of the service instance to bind, should be in the namespace of the binding |
| externalName       | `string`   |  The name of the service binding in SAP Cloud Service Management service, defaults to the binding `metadata.name` if not specified |
| secretName       | `string`   |  The name of the secret where credentials are stored, defaults to the binding `metadata.name` if not specified |
| parameters       |  `[]object`  |  Parameters for the binding |

#### Status
| Property         | Type     | Comments                                                                                                   |
|:-----------------|:---------|:-----------------------------------------------------------------------------------------------------------|
| instanceID   | `string`   | The ID of the bound instance in the SAP Cloud Service Management service |
| bindingID   | `string`   | The ID of the service binding in SAP Cloud Service Management service |
| operationURL | `string`   |  URL of the ongoing operation for the service binding |
| operationType   |  `string`   |  The operation type (CREATE/UPDATE/DELETE) of the ongoing operation |
| conditions       | `[]condition`   |  An array of conditions describing the status of the service instance. <br>The possible conditions types are:<br>- `Ready`: set to `true` if the binding is ready and usable<br>- `Failed`: set to `true` when an operation on the service binding fails, in this case the error details will be available in the condition message  

[Back to top](#sapcp-operator)

## Support
Feel free to open new issues for feature requests, bugs, or general feedback on this project's GitHub Issues page. 
The SAP CP Service Operator project maintainers will respond to the best of their abilities. 

## Contributions
We currently do not accept community contributions. 

## SAP CP kubectl Extension (Experimental) 
The SAP CP kubectl plugin extends kubectl with commands for getting the available services in your SAP BTP account, 
using the access credentials stored in the cluster.

### Prerequisites
- [jq](https://stedolan.github.io/jq/)

### Limitations
- The SAP CP kubectl plugin is currently based on `bash`. If using Windows, you should use the SAP BTP plugin commands from a Linux shell (e.g. [Cygwin](https://www.cygwin.com/)).  

### Installation
- Download https://github.com/sm-operator/sapcp-operator/releases/download/${release}/kubectl-sapcp
- Move the executable file to any location in your `PATH`

#### Usage
```
  kubectl sapcp marketplace -n <namespace>
  kubectl sapcp plans -n <namespace>
  kubectl sapcp services -n <namespace>
```

Use the `namespace` parameter to specify the location of the secret containing the SAP BTP access credentials, usually this location is the namespace in which you installed the operator. 
If not specified the `default` namespace is used. 


[Back to top](#sapcp-operator)
