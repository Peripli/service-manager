# Install the service-broker-proxy-k8s

## Prerequisites

* git
* kubectl
* helm
* service-catalog
* [Service-Manager](./sm.md) is installed.

**Note:** For details about the prerequisites you may refer to the [installation prerequisites page](./../development/install-prerequisites.md)

## Clone the repository

Clone the [service-broker-proxy-k8s](https://github.com/Peripli/service-broker-proxy-k8s) git repository.

```console
$ git clone https://github.com/Peripli/service-broker-proxy-k8s.git $GOPATH/src/github.com/Peripli/service-broker-proxy-k8s && cd $GOPATH/src/github.com/Peripli/service-broker-proxy-k8s
```

**Note:** Do not use `go get`. Instead use git to clone the repository.

## Register the Kubernetes cluster in Service Manager

To start the service-broker-proxy-k8s you need to register the kubernetes cluster in Service Manager. You can use the [smctl](./cli.md) `register-platform` command.
As a result this will return the credentials used for communicating with the Service Manager.
For example:

```console
$ smctl login -u admin -p admin -a http://service-manager.dev.cfdev.sh --skip-ssl-validation

Logged in successfully.
```

```console
$ smctl register-platform mycluster k8s example

ID                                    Name       Type  Description  Created               Updated               Username                                      Password
------------------------------------  ---------  ----  -----------  --------------------  --------------------  --------------------------------------------  --------------------------------------------
a6917890-457d-4c80-9660-9756825a8adb  mycluster  k8s   example      2018-10-09T10:28:07Z  2018-10-09T10:28:07Z  VdFGVssx1K6G0VWcId8lEmzj0/8meNNm5sRliGZ1qgc=  TkVWtgrOUZE4wTomC95dqKY33hXO46j/vWmvO49o9XI=
```

## Docker Images

Docker Images are available on [quay.io/service-manager/sb-proxy](https://quay.io/repository/service-manager/sb-proxy-k8s).

## Installation

The service-broker-proxy-k8s is installed via a helm chart located in the [service-broker-proxy GitHub repository](https://github.com/Peripli/service-broker-proxy-k8s).

Navigate to the root of the cloned repository and execute:

```console
helm install charts/service-broker-proxy-k8s --name service-broker-proxy --namespace service-broker-proxy --set config.sm.url=<SM_URL> --set sm.user=<SM_USER> --set sm.password=<SM_PASSWORD>
```

**Note:** Make sure you substitute `<SM_URL>` with the Service Manager URL, `<SM_USER>` and `<SM_PASSWORD>` with the credentials issued from Service Manager when this platform was registered there.

To use your own images you can set `image.repository`, `image.tag` and `image.pullPolicy` to the helm install command.
