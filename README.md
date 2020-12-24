# sapcp-operator
[![Coverage Status](https://coveralls.io/repos/github/sm-operator/sapcp-operator/badge.svg?branch=master&service=github)](https://coveralls.io/github/sm-operator/sapcp-operator?branch=master)
[![Build Status](https://github.com/sm-operator/sapcp-operator/workflows/Go/badge.svg)](https://github.com/sm-operator/sapcp-operator/actions)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/sm-operator/sapcp-operator/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/sm-operator/sapcp-operator)](https://goreportcard.com/report/github.com/sm-operator/sapcp-operator)

## Prerequisite
- kubernetes cluster
- kubectl
- helm

## Setup
`kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.1.0/cert-manager.yaml`

`kubectl create secret generic sapcp-operator-secret --from-literal=clientid="< clientid >" --from-literal=clientsecret="< secret >" --from-literal=url="< sm_url >" --from-literal=subdomain="< subdomain >" --namespace=sapcp-operator-system` </br>
e.g.</br>
`kubectl create secret generic sapcp-operator-secret --from-literal=clientid="myclient" --from-literal=clientsecret="mysecret" --from-literal=url="https://service-manager.cfapps.sap.hana.ondemand.com" --from-literal=subdomain="MyDemoSubaccount0909" --namespace=sapcp-operator-system`

`helm install test https://github.com/sm-operator/sapcp-operator/releases/download/${release}/sapcp-operator-${release}.tgz` </br>
sapcp-operator [releases]: https://github.com/sm-operator/sapcp-operator/releases


## Local setup
`./hack/install-kubebuilder.sh` </br>
`./hack/kind-with-registry.sh` </br>

### run tests
`make test`

### run locally
`make docker-build` </br>
`docker tag controller:latest localhost:5000/controller:latest` </br>
`docker push localhost:5000/controller:latest` </br>
`make deploy IMG=localhost:5000/controller:latest` </br>

