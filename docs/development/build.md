# Build from source

**Note**: See [Prerequisite](sm-tools.md)

You can set the docker client to the Minikube docker environment by executing:
* windows: ```@FOR /f "tokens=*" %i IN ('minikube docker-env') DO @%i```
* linux/mac: ```eval $(minikube docker-env)```

## Service Manager

Navigate to the project root directory.

The following command will build Service Manager binary and put it in the *bin* folder. Make sure you set `PLATFORM` and `ARCH` environment variable to your platform and architecture:
```console
$ make build
```

Alternatively you can build a docker image:

```console
$ docker build -t "service-manager:latest" -f Dockerfile .
```


## [Cloud Foundry Proxy (Agent)](https://github.com/Peripli/service-broker-proxy-cf)

Navigate to the project root directory.

First fetch all dependencies:
```console
$ dep ensure -v --vendor-only
```

To build an executable run the following command:
```console
$ go build -o cf-proxy github.com/Peripli/service-broker-proxy-cf
```

## [K8S Proxy (Agent)](https://github.com/Peripli/service-broker-proxy-k8s)

Navigate to the project root directory.

Build the image:
```console
$ docker build -t "sb-proxy-k8s:latest" -f Dockerfile .
```

If you have set your docker client to the Minikube docker you can directly use the above image to run a pod with the k8s proxy.
