# Run Service Manager

## Clone the Repository

Clone the [service-manager](https://github.com/Peripli/service-manager) repository.

    ```console
    $ git clone https://github.com/Peripli/service-manager.git $GOPATH/src/github.com/Peripli/service-manager && cd $GOPATH/src/github.com/Peripli/service-manager
    ```

**Note:** Do not use `go get`. Instead use git to clone the repository.

## Run on CF

### Prerequisites for CF deployment

* git
* go > 1.11
* dep
* OpenID compliant Authorization Server 
* CF CLI installed and configured.
* go_buildpack 1.8.19+
* PostgreSQL service is available or an external PostgreSQL is accessible from the CF environment.

**Note:** For details about the prerequisites you may refer to the [installation prerequisites page](./../development/install-prerequisites.md)

### Create PostgreSQL service instance in your CF environment

```console
cf create-service <postgres_service_name> <plan_name> <postgre_instance_name>
```

Alternatively, you can use external PostgreSQL as described in the [installation prerequisites page](./../development/install-prerequisites.md#postgres-database). In this case you need to have a PostgreSQL URI and substitute it in the the `STORAGE_URI` in `manifest.yml` as outlined below.

### Update manifest.yml file

Prepare the manifest for deployment using *[deployment/cf/manifest.yml](https://github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/blob/master/deployment/cf/manifest.yml)* as template:

* Update environment variable `STORAGE_NAME` by replacing the value *<postgre_instance_name>* with the instance name of your PostgreSQL service. Alternatively, you can use the `STORAGE_URI` environment variable to set external PostgreSQL URI, but in this case `STORAGE_NAME` environment variable and its value must be removed from the manifest.yml.
* Update environment variable `API_TOKEN_ISSUER_URL` by replacing the value *<api_token_issuer_url>* with the URL of your OAuth server. For example if you are running in CFDev you can use the CFDev UAA.

**Note:** To get the CF UAA URL you can execute the following command (you need to install [jq](https://stedolan.github.io/jq/)):

```console
cf curl /v2/info | jq .token_endpoint
```

### Push the application

From the root of the service manager project execute:

```console
cf push -f deployment/cf/manifest.yml
```

## Run on Kubernetes

### Prerequisites for Kubernetes deployment

* git
* go > 1.11
* dep
* OpenID compliant Authorization Server 
* kubectl is installed and configured to be used with the Kubernetes cluster
* helm is installed and configured on the cluster
* ingress controller is configured on the cluster *(optional)*
* External PostgreSQL accessible from the cluster *(optional)*

**Note:** For details about the prerequisites you may refer to the [installation prerequisites page](./../development/install-prerequisites.md)

### Install Service Manager

Go to *deployment/k8s/charts/service-manager* folder.

Execute:

```console
helm dependency build
```

to get the required dependencies.

To install the Service Manager and PostgreSQL database, execute:

```console
helm install --name service-manager --namespace service-manager . --set config.api.token_issuer_url=<api_token_issuer_url>
```

where *<api_token_issuer_url>* is the URL of your OAuth server. If this configuration is not set it will use the CFDev UAA URL - `https://uaa.dev.cfdev.sh`

You can also install the Service Manager with external PostgreSQL using a connection string (here `externalPostgresURI` sets the value for `STORAGE_URI` environment variable):

```console
helm install --name service-manager --namespace service-manager . --set postgresql.install=false --set externalPostgresURI=<postgresql_connection_string>
```

Or use Service Manager docker image from a different repository:

```console
helm install --name service-manager --namespace service-manager . --set image.repository=<image_repo> --set image.tag=<image_tag>
```

If ingress controller is not available you can disable ingress with `--set ingress.enabled=false`.
To expose the Service Manager outside the Kubernetes cluster you can change the service type to NodePort or LoadBalancer (if available).
For example:

```console
helm install --name service-manager --namespace service-manager . --set ingress.enabled=false --set service.type=NodePort
```
