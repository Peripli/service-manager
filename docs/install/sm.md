# Run Service Manager

## Prerequisites

* You need to have an OAuth server to be used by the Service Manager. This oauth server must support [OpenID Connect Discovery](https://openid.net/specs/openid-connect-discovery-1_0.html). In CF this could be the CF UAA.
* `git` is installed.

## Clone the repository

Clone the [service-manager](https://github.com/Peripli/service-manager) git repository.

```sh
git clone https://github.com/Peripli/service-manager.git
```

**Note:** Do not use `go get`. Instead use git to clone the repository.

## Run on CF

### Prerequisites for CF deployment

The following must be fulfilled:

* You have CF cli installed and configured. You are logged in to CF.
* PostgreSQL service is available or an external PostgreSQL is installed and accessible from the CF instance.

### Create PostgreSQL service instance in your CF environment

```sh
cf create-service <postgres_service_name> <plan_name> <postgre_instance_name>
```

Alternatively, you can use external PostgreSQL. In this case you need to have a PostgreSQL uri.

### Update manifest.yml file

Replace in *deployment/cf/manifest.yml*:

* *<postgre_instance_name>* with the instance name of your PostgreSQL service. Alternatively, you can use the `STORAGE_URI` environment variable to set external PostgreSQL uri. In this case `STORAGE_NAME` environment variable must not be present and you need to remove the service from the manifest.yml.
* *<api_token_issuer_url>* with the url of your OAuth server. For example if you are running in CFDev and you want to use the CFDev UAA you can replace it with `https://uaa.dev.cfdev.sh`.

### Push the application

From the root of the service manager project execute:

```sh
cf push -f deployment/cf/manifest.yml
```

## Run on Kubernetes

### Prerequisites for Kubernetes deployment

The following must be fulfilled:

* *kubectl* is installed and configured to be used with the Kubernetes cluster.
* Helm is installed and configured.
* Ingress controller is configured on the cluster *(optional)*

### Install Service Manager

Go to *deployment/k8s/charts/service-manager* folder.

Execute:

```sh
helm dependency build
```

to get the required dependencies.

To install the Service Manager and PostgreSQL database, execute:

```sh
helm install --name service-manager --namespace service-manager . --set config.api.token_issuer_url=<api_token_issuer_url>
```

where *<api_token_issuer_url>* is the url of your OAuth server. If this configuration is not set it will use the CFDev UAA url - `https://uaa.dev.cfdev.sh`

To change the PostgreSQL username or password you can use the `postgresql.postgresUser` and `postgresql.postgresPassword` configurations as in the example below:

```sh
helm install --name service-manager --namespace service-manager . --set postgresql.postgresUser=<pguser> --set postgresql.postgresPassword=<pgpass>
```

**Note:** These credentials will remain in your bash history. Alternatively you can change these values directly in *deployment/k8s/charts/service-manager/values.yaml* file.

You can install the Service Manager with external PostgreSQL using a connection string:

```sh
helm install --name service-manager --namespace service-manager . --set postgresql.install=false --set externalPostgresURI=<postgresql_connection_string>
```

Or use Service Manager docker image from a different repository:

```sh
helm install --name service-manager --namespace service-manager . --set image.repository=<image_repo> --set image.tag=<image_tag>
```

If ingress controller is not available you can disable ingress with `--set ingress.enabled=false`.
To expose the Service Manager outside the Kubernetes cluster you can change the service type to NodePort or LoadBalancer (if available).
For example:

```sh
helm install --name service-manager --namespace service-manager . --set ingress.enabled=false --set service.type=NodePort
```
