# Prerequisites

Prerequisites that will help you get ready to deploy/run the Service Manager components. 

## General Deployment Prerequisites

Generally required prerequisites in order to get started with the Service Manager. These would be required no matter where the Service Manager would be installed.

### Git

Setup is described [here](https://git-scm.com/)

### Go

Currently Service Manager requires Go version 1.10. Installation steps can be found [here](https://golang.org/doc/install).

### Dep

Most of the Service Manager github repositories do not include a `vendor` folder. You would need to have `dep` installed and run `dep ensure --vendor-only` to download the project dependencies. Installation details can be found [here](https://github.com/golang/dep#installation).

### smctl

Install `smctl` as described [here](https://github.com/Peripli/service-manager-cli/blob/master/README.md).

**Note:** `smctl` is required to register the proxies as platforms in SM and obtain credentials that they can use to access the SM APIs.

### Postgres Database

The Service Manager uses a Postgres database to store its data. The database could be running as a CF service, K8S deployment or externally. 

In the end, what is important is for the Service Manager to have `STORAGE_URI` env var or `storage.uri` pflag provided and the value should be an accessible postgres URI. 

If SM will be running in CF and the CF installation offers a postgres backing service, one could bind a postges service instance to SM and set the `STORAGE_NAME` env var or the `storage.name` pflag to the name of the postgres service instance. This would internally take care of setting the `storage URI`. More details can be found in the respective [CF instllation section](./../install/sm.md#run-on-cf).

If SM will be running on K8S, there are helm chart parameters which indicate whether to use an external postgres database or to setup one as part of the helm installation. More details can be found in the respective [K8S instllation section](./../install/sm.md#run-on-kubernetes).

**Note:** A postgres database is required to run the Service Manager. The proxies do not use a database.

### Authorization Server

You need to have an OAuth server to be used by the Service Manager. This OAuth server must support [OpenID Connect Discovery](https://openid.net/specs/openid-connect-discovery-1_0.html). In a CF installation this could be the CF UAA.

In the end, what is important is for the Service Manager to have `API_TOKEN_ISSUER_URL` env var or `api.token_issuer_url` pflag provided pointing to an accessible OAuth2 OpenID-compliant Authorization Server.

If SM will be running on CF, one could use UAA's token endpoint.

If SM will be running on K8S, one could provide a URL to an Authorization Server as part of the helm installation.

**Note:** An authorization server is required to run the Service Manager. The proxies do not use an authorization server.

### OSB-compliant platforms (CF/K8S with service catalog)

In order to run the Service Broker Proxies, actual platforms are required. 

The CF Proxy calls Cloud Controller APIs for service broker management and service access enablement and therefore requires a Cloud Foundry installation. 

The K8S proxy calls service catalog APIs for service broker management and requires a K8S cluster with service catalog installed.

**Note:** The Service Manager itself does not talk to the platforms , so technically if you want to run just the Service Manager, you may do so without any OSB-compliant platforms provided that you have a place to actually deploy SM.

## Kubernetes Deployment Prerequisites

Additional prerequisites to the [general deployment prerequisites](#general-deployment-prerequisites) required for installing the Service Manager (and/or the K8S Proxy) on a K8S Cluster.

### kubectl

Setup `kubectl` as described [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/). Also install the [svcat plugin](https://github.com/kubernetes-incubator/service-catalog/blob/master/docs/install.md#plugin).

### helm

Setup `helm` on your K8S Cluster. Details about installing `helm` can be found [here](https://github.com/kubernetes/helm/blob/master/docs/install.md).

Alternatively, **Windows** users may follow [these steps](https://medium.com/@JockDaRock/take-the-helm-with-kubernetes-on-windows-c2cd4373104b):


Afterwards, start Minikube and initialize Tiller.

```console
$ minikube start --extra-config=apiserver.Authorization.Mode=RBAC

$ helm init
```

 ### service catalog

 Setup `service catalog` on your K8S Cluster. Details about installing the `service catalog` can be found [here](https://github.com/kubernetes-incubator/service-catalog/blob/master/docs/install.md).

Alternatively, **Windows** users may follow these steps:

 ```console
 $ kubectl create clusterrolebinding add-on-cluster-admin --clusterrole=cluster-admin --serviceaccount=kube-system:default

 $ helm repo add svc-cat https://svc-catalog-charts.storage.googleapis.com

 $ kubectl create clusterrolebinding tiller-cluster-admin  --clusterrole=cluster-admin --serviceaccount=kube-system:default

 $ kubectl -n kube-system patch deployment tiller-deploy -p '{"spec": {"template": {"spec": {"automountServiceAccountToken": true}}}}'

 $ helm install svc-cat/catalog  --name catalog --namespace catalog
 ```

## Cloud Foundry Deployment Prerequisites

Additional prerequisites to the [general deployment prerequisites](#general-deployment-prerequisites) required for installing the Service Manager (and/or the CF Proxy) on CF.

#### CF CLI

Details about installing `CF CLI` can be found [here](https://github.com/cloudfoundry/cli#downloads).

**Note:**  kubectl, svcat, helm and smctl binaries should be in your $PATH.

#### Cloud Foundry Installation

You would also need a Cloud Foundry installation and either user/pass or oauth client credentials to manage service brokers as well as service access at the Cloud Controller API.

**Note:** The CF installation should have a `go_buildpack` installed with support for Go 1.10+. This would imply that the go buildpack version should be newer than 1.8.19. The buildpack should use the name `go_buildpack`. Steps to upgrade the `go_buildpack` in case this is required:

* Download the latest Release from [here](https://github.com/cloudfoundry/go-buildpack/releases)
* Rename the zip so it contains no dots (.) and slashes (-)
* Navigate to the directory the zip was downloaded to and run:

    ```console
    cf update-buildpack go_buildpack <buildpackzipname>.zip 1
    ```
