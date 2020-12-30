# sapcp-operator
[![Coverage Status](https://coveralls.io/repos/github/sm-operator/sapcp-operator/badge.svg?branch=master&service=github)](https://coveralls.io/github/sm-operator/sapcp-operator?branch=master)
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
2. Create service manager secret:
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
3. Deploy the sapcp-operator in the cluster:
    ```
    helm install sapcp-operator https://github.com/sm-operator/sapcp-operator/releases/download/${release}/sapcp-operator-${release}.tgz
    ```

    The list of available releases is available here: [sapcp-operator releases](https://github.com/sm-operator/sapcp-operator/releases)

## Local setup
### Create kind with local docker registry
`./hack/kind-with-registry.sh`

### Run tests
`make test`

### Deploy locally
```
make docker-build
docker tag controller:latest localhost:5000/controller:latest
docker push localhost:5000/controller:latest
make deploy IMG=localhost:5000/controller:latest
```

