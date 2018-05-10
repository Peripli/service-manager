# Local Setup for Service Manager Components

This page intends to explain how to setup a local / development environment for the Service Manager Components.

* [Preparing your machine](#preparing-your-machine)
  * [Setup Docker](#setup-docker)
  * [Setup VirtualBox](#setup-virtualbox)
  * [Setup PCF Dev](#setup-pcf-dev)
  * [Setup Minikube](#setup-minikube)
    * [Install Minikube](#install-minikube)
    * [Start and Prepare Minikube](#start-and-prepare-minikube)
* [Setup Service Manager Components](#setup-service-manager-components)
  * [Setup the Service Manager](#setup-the-service-manager)
    * [Run Service Manager on localhost](#run-service-manager-on-localhost)
    * [Run Service Manager inside PCF Dev](#run-service-manager-inside-pcf-dev)
    * [Run Service Manager inside Minikube](#run-service-manager-inside-minikube)
  * [Setup CF Proxy](#setup-cf-proxy)
    * [Run CF Proxy on localhost](#run-cf-proxy-on-localhost)
    * [Run CF Proxy inside PCF Dev](#run-cf-proxy-inside-pcf-dev)
  * [Setup K8S Proxy](#setup-k8s-proxy)
    * [Run K8S Proxy on localhost](#run-k8s-proxy-on-localhost)
    * [Run K8S Proxy inside Minikube](#run-k8s-proxy-inside-minikube)
  * [Setup a Dummy Service Broker](#setup-a-dummy-service-broker)
    * [Run Broker on localhost](#run-broker-on-localhost)
    * [Run Broker inside PCF Dev](#run-broker-inside-pcf-dev)
  * [Setup SM CLI](#setup-sm-cli)

## Preparing your machine

The following tools would need to be installed on your machine prior to being able to run the local setup for the Service Manager.

### Setup Docker

For MAC users both [Docker for Mac](https://docs.docker.com/docker-for-mac/install/) and [Docker ToolBox](https://docs.docker.com/toolbox/toolbox_install_mac/) should work.

For Windows users only [Docker ToolBox](https://docs.docker.com/toolbox/toolbox_install_windows/) works. The reason is that Docker Toolbox uses VirtualBox for virtualization and [Docker for Windows](https://docs.docker.com/docker-for-windows/install/#about-windows-containers) uses the native OS virtualization (Hyper-V). Since PCF Dev requires VirtualBox and using VirtualBox on Windows requires disabling Hyper-V, it is currently not an option to use Docker for Windows.

### Setup VirtualBox

> **Note:** For Windows make sure Hyper-V is turned off.

If you decide to use Docker ToolBox, VirtualBox is installed as part of it.

If you are a Mac user and you decide to use Docker for Mac, you would need to seperatly install VirtualBox. You may download it from [here](https://www.virtualbox.org/wiki/Downloads).

### Setup PCF Dev

The [following guide](https://pivotal.io/platform/pcf-tutorials/getting-started-with-pivotal-cloud-foundry-dev/introduction) describes how to install PCF Dev.

The Service Manager components use go1.10 and currenty  the buildpack installed in PCF Dev does not support go1.10. The following steps install the latest go buildpack version on PCF Dev:

* Download the latest Release from [here](https://github.com/cloudfoundry/go-buildpack/releases)
* Rename the zip so it cotnains no dots (.) and slashes (-)
* Navigate to the directory the zip was downloaded to and run:

    ```bash
    cf create-buildpack go_buildpack2 <buildpackzipname>.zip 1
    ```

### Setup Minikube

#### Install Minikube

In general the installation steps are in the [Installation section of the getting started guide](https://kubernetes.io/docs/getting-started-guides/minikube/#installation).

Alternatively, Windows users may follow these steps:

> **Note:** On Windows Minikube versions 26.x does not work on Windows. Currently version 25.2 works properly.

* Setup chocolately package manager

    ```powershell
    @"%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe" -NoProfile -InputFormat None -ExecutionPolicy Bypass -Command "iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))" && SET "PATH=%PATH%;%ALLUSERSPROFILE%\chocolatey\bin"
    ```

* install kubectl

    ```bat
    choco install kubernetes-cli
    ```

* install minikube

    ```bat
    choco install minikube --version 0.25.2
    ```

#### Start and Prepare Minikube

* install Helm

    In general Helm installation is explained [here](https://github.com/kubernetes/helm/blob/master/docs/install.md).

    Alternatively, Windows users may follow [this guide](https://medium.com/@JockDaRock/take-the-helm-with-kubernetes-on-windows-c2cd4373104b).

* start minikube with RBAC

    ```bash
    minikube start --extra-config=apiserver.Authorization.Mode=RBAC
    ```

* setup Tiller

    ```bash
    helm init
    ```

* install [service-catalog](https://github.com/kubernetes-incubator/service-catalog) in Minikube

    In general, the installation steps for service catalog are described [here](https://github.com/kubernetes-incubator/service-catalog/blob/master/docs/install.md).

    Alternatively, the following sequence of commands should work, too.

    ```bash
    kubectl create clusterrolebinding add-on-cluster-admin --clusterrole=cluster-admin --serviceaccount=kube-system:default

    helm repo add svc-cat https://svc-catalog-charts.storage.googleapis.com

    helm search service-catalog

    kubectl create clusterrolebinding tiller-cluster-admin  --clusterrole=cluster-admin --serviceaccount=kube-system:default

    kubectl -n kube-system patch deployment tiller-deploy -p '{"spec": {"template": {"spec": {"automountServiceAccountToken": true}}}}'

    helm install svc-cat/catalog  --name catalog --namespace catalog
    ```

>**Note:** In order to reuse the docker daemon and speed up local development check the [Reusing the Docker daemon section of the minikube getting started guide](https://kubernetes.io/docs/getting-started-guides/minikube/#reusing-the-docker-daemon)

___

## Setup Service Manager Components

This step shows how to setup the Service Manager as well as the Service Manager Proxies (CF and K8S). There are a couple of ways things can be done.
For development purposes, the easiest way is to have everything (Service Manager, CF Proxy, K8S Proxy, Dummy Broker) to be running locally on your machine and have them configured to talk to Minikube and PCF Dev. Alternatively, the components can be setup to run on PCF Dev or Minikube and you may run on `localhost` only the component that you are currently working on.

The next steps describe how each component can be setup either to run locally or to run on a locally prepared platform (local CF / K8S minikube)

### Setup the Service Manager

Clone the repository:

```bash
git clone https://github.com/Peripli/service-manager.git

```

#### Run Service Manager on localhost

>**Prerequisites:**
>
>Setup Postgresql
>
>In order for the Service Manager to run locally, itrequires a >Postgresql DB.
>
> * Install on your machine
>
>    Windows users may follow [this guide]>>(http://www.postgresqltutorial.com/install-postgresql/) in order to install postgresql.
>
> * Spin up a postgres docker container
>
>    ``` bash
>    docker run --name postgres-docker -d -p 5432:5432 -e POSTGRES_PASSWORD=postgres -e POSTGRES_USER=postgres postgres
>    ```
>
>The DB should be accessible on the ip that the docker machine is running on (docker-machine ip) and port 5432.

* Modify the contents of application.yml to point to the configured Postgresql

    ``` yml
    server:
      requestTimeout: 3000
      shutdownTimeout: 3000
      port: 8080
    log:
      level: debug
      format: text
    db:
      name: sm-postgres
      uri: postgres://postgres:postgres@{postgres-ip}:5432/postgres?sslmode=disable
    ```

    >**Note**: Modify `server.port` to specify a different port on which the Service Manager should run.

* Modify the contents of `service-manager/api/osb/logic.go`

    ```go
    func (b *BusinessLogic) ValidateBrokerAPIVersion(version string) error {
        return nil
    }
    ```

* Navigate to the `service-manager` folder and run the application:

    ``` go
    go run cmd/main.go

    ```

* Test that the setup works

    TODO

#### Run Service Manager inside PCF Dev

* Modify the contents of `service-manager/api/osb/logic.go`

    ```go
    func (b *BusinessLogic) ValidateBrokerAPIVersion(version string) error {
        return nil
    }
    ```

* Deploy as a Docker Container

    Service Manager is deployed on CF as a docker container. The steps are described [here](https://github.com/Peripli/service-manager/blob/0c40dff68d1547ecea6841b2b87354dcc44b3bd8/deployment/cf/README.md).

* Test that the setup works

    TODO

#### Run Service Manager inside Minikube

* Install with Helm

    The easiest way to install the Service Manager inside Minikube is to use the provided Helm charts. Run the following:

    ```bash
    TODO - add helm install for SM here
    ```

* Test that it works

    TODO

> **Note:** When regitering  a service broker inside the Service Manager beware of the fact that the Service Manager will need to be calling the broker's URLs. This is important in case in your local setup the Service Manager is running on PCF Dev or Minikube and the registered service broker is running on `localhost`. In this case the URLs of the broker registered in the SM should point to `10.0.2.2` instead of `localhost`. The reason is that the Service Manager is running inside a virtual machine(either Minikube or PCF Dev) and needs to call `localhost` (where the service broker runs). Inside both PCF Dev's and Minikube's VM the host's `localhost` is resolved under `10.0.2.2`.

### Setup CF Proxy

Clone the repository:

``` bash
git clone https://github.com/Peripli/service-manager-proxy-cf.git

```

#### Run CF Proxy on localhost

* Modify the contents of `vendor/github.com/Peripli/service-broker-proxy/pkg/osb/business_logic.go`

    ```go
    func (b *BusinessLogic) ValidateBrokerAPIVersion(version string) error {
        return nil
    }
    ```

* Make the following code changes

    Replace the following line from `main.go`: `cfEnv := cf.NewCFEnv(env.Default(""))` with `cfEnv := env.Default("")`

* Make sure the application.yml looks similar to:

    ``` yml
    app:
    port: 8080
    logLevel: debug
    logFormat: text
    timeoutSeconds: 500
    host: http://10.0.2.2:8080
    sm:
    user: admin
    password: admin
    host: http://localhost:8081
    osbApi: /v1/osb
    timeoutSeconds: 1000
    cf:
    api: https://api.local.pcfdev.io
    username: admin
    password: admin
    skipSSLVerify: true
    timeoutSeconds: 500
    reg:
        user: admin
        password: admin
    ```
* Run the following command:

    ``` go
    go run main.go

    ```

* Test that it works

    TODO

> **Note:** If the locally started CF Proxy uses different port from `8080`, replace it in both `app.port` and `app.host`.
>
> **Note:** `app.host` is used when the proxy is already registered as a broker inside the platform (in this case the PCF VM) and the platform wants to talk back the proxy (ex. to fetch catalog). In order for the Minikube PCF to be able to call out to the host's `localhost`, `app.host` should be set to `10.0.2.2:{port_of_cf_proxy}`
>
> **Note:** `sm.host` should point to the Service Manager that the proxy will be connecting to. For example, if the Service Manager is running locally, set `sm.host` to `http://localhost:{port-where-sm-is-running}`

#### Run CF Proxy inside PCF Dev

The following steps describe how to run the CF Proxy to run inside PCF Dev.

* Modify the contents of `vendor/github.com/Peripli/service-broker-proxy/pkg/osb/business_logic.go`

    ```go
    func (b *BusinessLogic) ValidateBrokerAPIVersion(version string) error {
        return nil
    }
    ```

* Make sure the `manifest.yml` looks simiar to:

```yml
---
applications:
  - name: sm-proxy
    memory: 256M
    env:
      GOVERSION: go1.10
      GOPACKAGENAME: github.com/Peripli/service-broker-proxy-cf
      SM_HOST: http://10.0.2.2:8081
      CF_USERNAME: cfuser
      CF_PASSWORD: cfpassword
```

> **Note:** `SM_HOST` should point to the Service Manager that the proxy will be connecting to. If the Service Manager is running on the host's `localhost`, use `10.0.2.2` instead.

### Setup K8S Proxy

Clone the repository:

``` bash
git clone https://github.com/Peripli/service-broker-proxy-k8s.git
git checkout dev
```

#### Run K8S Proxy on localhost

* Make sure the `application.yml` looks similar to:

``` yml
app:
  port: 8083
  logLevel: debug
  logFormat: text
  timeoutSeconds: 5
  host: http://10.0.2.2:8083
sm:
  user: admin
  password: admin
  host: http://localhost:8081
  osbApi: /v1/osb
  timeoutSeconds: 10
```

* Run the following command:

    ``` go
    go run main.go -kubeconfig=$HOME/.kube/config

    ```
* Test that it works

    TODO

> **Note:** If the locally started CF Proxy uses different port than `8083`, replace it in both `app.port` and `app.host`.
>
> **Note:** `app.host` is used when the proxy is already registered as a broker inside the platform (in this case the Minikube VM) and the platform wants to talk back the proxy (ex. to fetch catalog). In order for the Minikube VM to be able to call out to the host's `localhost`, `app.host` should be set to `10.0.2.2:{port_of_k8s_proxy}`
>
> **Note:** `sm.host` should point to the Service Manager that the proxy will be connecting to. For example, if the Service Manager is running locally, set `sm.host` to `http://localhost:{port-where-sm-is-running}`

#### Run K8S Proxy inside Minikube

Running inside Minikube is done via the provided Helm Charts.

* Navigate to the project directory and prepare an image

    ```bash
    docker build . -t local/k8sproxy:1
    ```

* Tell minikube to use local docker registry

    ```bash
    eval $(minikube docker-env)
    ```

* Install via Helm:

    ```bash
    helm install charts/service-broker-proxy --name service-broker-proxy --namespace service-broker-proxy --set config.serviceManager.host=<http://10.0.2.2:8081> --set image.repository=local/k8sproxy --set image.tag=1 --set image.pullPolicy=Never
    ```

    >**Note:** `config.serviceManager.host` should point to the Service Manager the proxy will be connecting to. In case the Service Manager is running on `localhost`, set `config.serviceManager.host` to `http://10.0.2.2:{port_of_sm}`

    Alternatively you may find additional details about setting up the k8s proxy inside Minikube [here](https://github.com/Peripli/service-broker-proxy-k8s/blob/dev/README.md)

### Setup a Dummy Service Broker

In order to have a fully setup Service Manager flow, it would be useful have a simple Service Broker running that will be registered in the Service Manager (this is often needed when you are working on either of the Proxies or on the OSB API of the Service Manager).

#### Run Broker on localhost

If the Dummy Service Broker is running on `localhost`, beware of where the Service Manager is running:

* If the Service Manager is running as application in PCF Dev or as deployment in Minikube and the Dummy Broker is running on `localhost`, then when the dummy brokers are registered inside the Service Manager, URL `10.0.2.2:{port}` instead of `localhost:{port}` should be used.

* If both the Service Manager and the Dummy Broker are running on `localhost`, then use `localhost:{port}` when registering the broker inside the Service Manager.

#### Run Broker inside PCF Dev

If the Dummy Service Broker is running inside PCF, beware of where the Service Manager is running:

* If the Serice Manager is running on `localhost` and the Dummy Broker is running on PCF Dev, then then broker must be registered with URL `{brokerAppName}.local.pcfdev.io`.

* If the Service Manager is running in PCF Dev and the Dummy Broker is running on PCF Dev, then the broker MUST be registered `{brokerAppName}.local.pcfdev.io`.

* If the Service Manager is running in Minikube and the Dummy Broker is running on PCF Dev, additional network configurations will be required.

### Setup SM CLI

Clone the repository:

``` bash
git clone https://github.com/Peripli/service-manager-cli.git
```
