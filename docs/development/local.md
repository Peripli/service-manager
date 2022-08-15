# Local Setup for Service Manager Components

## Prerequisites

Each of the sections below outlines only the prerequisites required to run the particular component in the particular platform. However, if you want to setup the full-fledged local environment, its recommended to setup everything that is described in both the [local dev prerequisites](develop-prerequisites.md) and the [installation prerequisites](install-prerequisites.md).

## Service Manager

### Service Manager on `minikube`

#### Prerequisites

* [general deployment prerequisites](./install-prerequisites.md#general-deployment-prerequisites)
* [minikube](./develop-prerequisites.md#minikube)
* [K8S deployment prerequisites](./install-prerequisites.md#kubernetes-deployment-prerequisites)
* in case you are planning to develop/contribute code, you should check the [local development prerequisites](./develop-prerequisites#local-development-prerequisites)

#### Installation 

Follow [run on Kubernetes](./../install/sm.md#run-on-Kubernertes) installation steps.

### Service Manager on `cfdev/pcfdev`

#### Prerequisites

* [general deployment prerequisites](./install-prerequisites.md#general-deployment-prerequisites)
* [local CF installation](./develop-prerequisites.md#local-cf-setup)
* [CF deployment prerequisites](./install-prerequisites.md#cloud-foundry-deployment-prerequisites)
* in case you are planning to develop/contribute code, you should check the [local development prerequisites](./develop-prerequisites#local-development-prerequisites)
 
#### Configuration Adjustments

Edit the following `env` vars in the `$GOPATH/src/github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/deployment/cf/manifest.yml`:

* set the token endpoint of the authorization server that was setup as part of the prerequisites

    ```yml
    env:
      API_TOKEN_ISSUER_URL: https://uaa.dev.cfdev.sh
    ```

* set the storage configuration

    If you are using an external database that is running on `localhost:5432` then set the `STORAGE_URI` (use 10.0.2.2 as the application will run inside the CF Dev VM and `10.0.2.2` maps to the host's `localhost`):

    **Note:** You might need to setup a loopback alias. On MacOS, you may use `sudo ifconfig lo0 alias 10.0.2.2`.

    ```yml
    env:
      STORAGE_URI: postgres://postgres:postgres@10.0.2.2:5432/postgres?sslmode=disable
    ```

    If your CF installation has a `postgres` service, then create and bind it to the application and set `STORAGE_NAME` to the postgres service instance name:

    ```yml
    env:
      STORAGE_NAME: postgres
    ```

**Note:** Comment the `docker.image` in the `manifest.yml` in order to use local sources.

#### Installation

Follow [run on CF](./../install/sm.md#run-on-CF) installation steps.

### Service Manager on `localhost`

#### Prerequisites

* check the [local development prerequisites](./develop-prerequisites.md#local-development-prerequisites) 

#### Configuration Adjustments

Edit the following `properties` in the `$GOPATH/src/github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/application.yml`:

* set the token endpoint of the authorization server that was setup as part of the prerequisites

    ```yml
    api:
      token_issuer_url: https://uaa.dev.cfdev.sh
      skip_ssl_verification: true
    ```
* set the storage configuration

    ```yml
    storage:
      uri: postgres://postgres:postgres@10.0.2.2:5432/postgres?sslmode=disable
    ```

    **Note:** Comment `storage.name` if its present in application.yml


#### Installation

```console
go run main.go
```

**Note:** One may skip the configuration adjustments in the `application.yml` file and instead set the values as commandline flags.

```console
go run main.go --storage.uri=postgres://postgres:postgres@10.0.2.2:5432/postgres?sslmode=disable --api.token_issuer_url=https://uaa.dev.cfdev.sh --api.skip_ssl_verification=true
```

## Service Broker CF Proxy

### CF Proxy on `cfdev/pcfdev`

#### Prerequisites

 * [git](https://git-scm.com/)
 * [smctl](https://github.com/Peripli/service-manager-cli/blob/master/README.md)
 * [local CF installation](./develop-prerequisites.md#local-cf-setup)
 * [CF deployment prerequisites](./install-prerequisites.md#cloud-foundry-deployment-prerequisites)

#### Configuration Adjustments

Example adjustments for `manifest.yml`

```yml 
# Access the SM API
  SM_URL: http://service-manager.dev.cfdev.sh
  SM_USER: /u4PGZHsLVX9WXsibWHZ4lNWDvhreeRvkUpFnBYPq/k=
  SM_PASSWORD: Pe4jPzR18pO3xvhkUD1MfnU3jHTJvot4blc9RR5ustk=
  SM_SKIP_SSL_VALIDATION: true
# Access the CC API to register brokers/ manage access
  CF_CLIENT_USERNAME: admin
  CF_CLIENT_PASSWORD: admin
  CF_CLIENT_APIADDRESS: https://api.dev.cfdev.sh
  CF_CLIENT_SKIPSSLVALIDATION: true
```

The necessary configuration adjustments are mentioned in the installation section about [modifying the manifest](../install/cf-proxy.md#modify-manifest.yml). Additionally, as shown above `skipping SSL validation` needs to be set to `true`.

#### Installation

Follow the [run on CF](./../install/cf-proxy.md) installation steps.

**Note:** Remember to set the skipping of SSL validation to `true` as shown in the example above.

## Service Broker K8S Proxy

### K8S Proxy on `minikube`

#### Prerequisites

* [git](https://git-scm.com/)
* [smctl](https://github.com/Peripli/service-manager-cli/blob/master/README.md)
* [minikube](./develop-prerequisites.md#local-k8s-setup)
* [K8S deployment prerequisites](./install-prerequisites.md#kubernetes-deployment-prerequisites)

#### Installation

Follow the [run on K8S](./../install/k8s-proxy.md) installation steps.

## Service Manager CLI

### smctl as Binary

Follow the [installation steps](./../installation/cli.md) to install the binary.

### smctl Locally

* Clone the [smctl](https://github.com/Peripli/service-manager-cli) repository.

    ```console
    git clone https://github.com/Peripli/service-manager-cli.git $GOPATH/src/github.com/Peripli/service-manager-cli && cd $GOPATH/src/github.com/Peripli/service-manager-cli
    ```

* install dependencies

    ```console
    go get
    ```

* Run/debug a command by executing `go run main.go` followed by the command

    Example:
    ```console
    $ go run main.go version
    Service Manager Client 0.0.1
    ```