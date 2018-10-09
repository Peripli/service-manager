# Build

// TODO: Add link to prerequisite
**Note**: See Prerequisite

## Binaries

* Service Manager

Navigate to the project root directory.

The following command will build Service Manager binary and put it in the *bin* folder. Make sure you set `PLATFORM` and `ARCH` environment variable to your platform and architecture:
```sh
make build
```

* Cloud Foundry Proxy (Agent)

// TODO: link to repo

Navigate to the project root directory and execute the following command:
```sh
go build -o cf-proxy github.com/Peripli/service-broker-proxy-cf
```

* K8S Proxy (Agent)

For this component only Docker build is relevant, as it cannot be run outside of Kubernetes

## Docker

You can set the docker client to the Minikube docker environment by executing
* windows: ```@FOR /f "tokens=*" %i IN ('minikube docker-env') DO @%i```
* linux/mac: ```eval $(minikube docker-env)```

* Service Manager

Navigate to the project root directory.

Build the image:
```sh
docker build -t "service-manager:latest" -f Dockerfile .
```

Alternatively you can build a docker image and push it to an external repository.

```sh
docker build -t "<image_name>:<tag>" -f Dockerfile .
docker push "<image_name>:<tag>"
```

* Cloud Foundry Proxy (Agent)

// TODO: Create a Dockerfile for it?

* K8S Proxy (Agent)

// TODO: Link to repo

Navigate to the project root directory.

Build the image:
```sh
docker build -t "sb-proxy-k8s:latest" -f Dockerfile .
```

If you have set your docker client to the Minikube docker you can directly use the above image to run a pod with the k8s proxy.