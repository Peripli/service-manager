# Deploy Service Manager on Kubernetes on Minikube:

## Prerequisites

The following must be fulfilled:

 * Minikube is installed and configured.
 * Helm is installed and configured.
 * Ingress controller is configured (optional)

## Build a docker image

You can set the docker client to the Minikube docker environment by executing
* windows: ```@FOR /f "tokens=*" %i IN ('minikube docker-env') DO @%i```
* linux/mac: ```eval $(minikube docker-env)```

and build the image

```sh
docker build -t "service-manager:latest" -f Dockerfile .
```

Alternatively you can build a docker image and push it to an external repository.

```sh
docker build -t "<image_name>:<tag>" -f Dockerfile .
docker push "<image_name>:<tag>"
```

In this case you have to specify the image, tag and pullPolicy in the Service Manager helm install command.

## Install Service Manager

Go to *deployment/k8s/charts/service-manager* folder.

Execute:

```sh
helm dependency update
```

to get the required dependencies.

To install the Service Manager and PostgreSQL database, execute:

```sh
helm install --name service-manager --namespace service-manager .
```

Alternatively you can use different PostgreSQL username or password:

```sh
helm install --name service-manager --namespace service-manager . --set postgresql.postgresUser=<pguser> --set postgresql.postgresPassword=<pgpass>
```

Or install only Service Manager with external PostgreSQL:

```sh
helm install --name service-manager --namespace service-manager . --set postgresql.install=false --set externalPostgresURI=<postgresql_connection_string>
```

Or use Service Manager docker image from external repo:
```sh
helm install --name service-manager --namespace service-manager . --set image.repository=<image_repo> --set image.tag=<image_tag> --set image.pullPolicy=Always
```

If ingress controller is not available you can disable ingress with `--set ingress.enabled=false`.
To expose the Service Manager outside the Minikube you can change the service type to NodePort or LoadBalancer (if available).
For example:

```sh
helm install --name service-manager --namespace service-manager . --set ingress.enabled=false --set service.type=NodePort
```
