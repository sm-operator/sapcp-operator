# sapcp-operator
[![Coverage Status](https://coveralls.io/repos/github/sm-operator/sapcp-operator/badge.svg?branch=master)](https://coveralls.io/github/sm-operator/sapcp-operator?branch=master)
[![Build Status](https://github.com/sm-operator/sapcp-operator/workflows/Go/badge.svg)](https://github.com/sm-operator/sapcp-operator/actions)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/sm-operator/sapcp-operator/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/sm-operator/sapcp-operator)](https://goreportcard.com/report/github.com/sm-operator/sapcp-operator)

## Prerequisites
- kubernetes cluster
- kubectl
- helm

## Setup
1. Install [cert-manager](https://cert-manager.io/docs/installation/kubernetes):
    ```
    kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.1.0/cert-manager.yaml
    ```

2. Deploy the sapcp-operator in the cluster:
    ```
    helm install sapcp-operator https://github.com/sm-operator/sapcp-operator/releases/download/${release}/sapcp-operator-${release}.tgz
    ```

    The list of available releases is available here: [sapcp-operator releases](https://github.com/sm-operator/sapcp-operator/releases)

3. Create service manager secret:
    ```
    kubectl create secret generic sapcp-operator-secret \
      --from-literal=clientid="< clientid >" \
      --from-literal=clientsecret="< secret >" \
      --from-literal=url="< sm_url >" \
      --from-literal=subdomain="< subdomain >" \
      --namespace=sapcp-operator-system
     ```
     e.g.
    ```
    kubectl create secret generic sapcp-operator-secret \
     --from-literal=clientid="myclient" \
     --from-literal=clientsecret="mysecret" \
     --from-literal=url="https://service-manager.cfapps.sap.hana.ondemand.com" \
     --from-literal=subdomain="MyDemoSubaccount0909" \
     --namespace=sapcp-operator-system
    ```

## Local setup
### Prerequisites
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/)

### Deploy locally
```
make docker-build
kind load docker-image controller:latest
make deploy IMG=controller:latest
```

### Run tests
`make test`

### SAPCP kubectl extension (experimental) 
Download https://github.com/sm-operator/sapcp-operator/releases/download/${release}/kubectl-sapcp
move its executable file to anywhere on your ``PATH``

#### Usage
 namespace parameter indicates the namespace where to find SM secret, defaulting to default namespace 
```
  kubectl sapcp marketplace -n <namespace>
  kubectl sapcp plans -n <namespace>
  kubectl sapcp services -n <namespace>
```

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

3.  Check that your service status is **Created**.

    ```bash
    kubectl get service
    NAME        INSTANCE    STATUS    AGE
    mybinding   myservice   Created   9s
    
    ```

4.  Check that a secret of the same name as your binding is created. The secret contains the service credentials that apps in your cluster can use to access the service.

    ```bash
    kubectl get secrets
    NAME        TYPE     DATA   AGE
    mybinding   Opaque   5      102s
    ```
