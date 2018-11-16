# Prerequisites

The prerequisites page contains details about all the tools that are needed for instaling / developing the Service Manager Components.

## Local Development Prerequisites

Prerequisites required for installing the Service Manager (and/or the CF and K8S Proxies) on `localhost`, performing builds, running tests, etc...

### Git

Setup is described [here](https://git-scm.com/)

### Go

Currently Service Manager requires Go version 1.10. Installation steps can be found [here](https://golang.org/doc/install).

### Dep

Most of the Service Manager github repositories do not include a `vendor` folder. You would need to have `dep` installed and run `dep ensure --vendor-only` to download the project dependencies. Installation details can be found [here](https://github.com/golang/dep#installation).

### GNU Make

Makefiles are provided in the repositories. In order to use them, one would need to setup `make` as described [here](https://www.gnu.org/software/make/manual/make.html).

**Note:** Running `make help` would print out the Makefile documentation.

### Docker

* [Docker for Windows](https://docs.docker.com/docker-for-mac/install/)
* [Docker for Mac](https://docs.docker.com/docker-for-windows/install/)

### Postgres on Docker

```console
$ docker run --name postgres -p 5432:5432 -e POSTGRES_PASSWORD=postgres -e POSTGRES_USER=postgres -d postgres
```

**Note:** If you are using `Docker for Window` or `Docker for Mac`, the database URI should be `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`. Otherwise, instead of `localhost` you would need to use the `$(docker-machine ip)`.

The obtained value can be used to set the Service Manger `STORAGE_URI`.

## Local K8S Setup

### minikube

A locally running K8S Cluster. Check the [minikube installation guide](https://kubernetes.io/docs/getting-started-guides/minikube/#installation.)

Alternatively, **Windows** users may follow these steps:

* Setup chocolately package manager

```powershell
    @"%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe" -NoProfile -InputFormat None -ExecutionPolicy Bypass -Command "iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))" && SET "PATH=%PATH%;%ALLUSERSPROFILE%\chocolatey\bin"
```

* Install minikube

    ```bat
    choco install minikube
    ```

>**Note:** In order to reuse the docker daemon and speed up local development check the [Reusing the Docker daemon section of the minikube getting started guide](https://kubernetes.io/docs/getting-started-guides/minikube/#reusing-the-docker-daemon)

### Fulfill the K8S Deployment Prerequisites

The [kubernetes deployment prerequisites]() section outlines some additional steps you would need to perform to fully setup the development environment.

## Local CF Setup

### PCF Dev

A locally running CF installation that includes an Authorization Server (UAA). Installation steps for PCFDev can be found [here](https://pivotal.io/platform/pcf-tutorials/getting-started-with-pivotal-cloud-foundry-dev/introduction).

### CF Dev

Alternatively, instead of `PCF Dev` one can use [CF Dev](https://github.com/cloudfoundry-incubator/cfdev)

**Note**: Installing `PCF Dev` or `CF Dev` also includes an Authrization Server(UAA). To get the CF UAA URL you can execute the following command (you need to install [jq](https://stedolan.github.io/jq/)):

```console
cf curl /v2/info | jq .token_endpoint
```

The obtained value can be used to set the Service Manager `API_TOKEN_ISSUER_URL`.

### Fulfill the Deployment Prerequisites

Depending on which pieces of the SM you will be developing on, you would need to fulfill the relevant deployment prerequisites in order to eventually be able to run/start/install the Service Manager components. For further details, check the [installation prerequisites page](install-prerequisites.md).