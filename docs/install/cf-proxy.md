# Install the service-broker-proxy-cf

## Prerequisites

* git
* go 1.10
* dep
* CF CLI
* go_buildpack 1.8.19+
* [Service-Manager](./sm.md) is installed.

**Note:** For details about the prerequisites you may refer to the [installation prerequisites page](./../development/install-prerequisites.md)

## Clone the repository

Clone the [service-broker-proxy-cf](https://github.com/Peripli/service-broker-proxy-cf) git repository.

```console
$ git clone https://github.com/Peripli/service-broker-proxy-cf.git $GOPATH/src/github.com/Peripli/service-broker-proxy-cf && cd $GOPATH/src/github.com/Peripli/service-broker-proxy-cf
```

**Note:** Do not use `go get`. Instead use git to clone the repository.

## Install dependencies

```console
dep ensure --vendor-only
```

## Register CF in Service Manager

To start the service-broker-proxy-cf you need to register CF in Service Manager. You can use the [smctl](./cli.md) `register-platform` command.
As a result this will return the credentials used for communicating with the Service Manager.
For example:

```console
$ smctl login -u admin -p admin -a http://service-manager.dev.cfdev.sh --skip-ssl-validation

Logged in successfully.
```

```console
$ smctl register-platform mycf cf example

ID                                    Name  Type  Description  Created               Updated               Username                                      Password
------------------------------------  ----  ----  -----------  --------------------  --------------------  --------------------------------------------  --------------------------------------------
16909cbe-610c-4c46-b586-fac2747beb47  mycf  cf    example      2018-10-09T10:26:01Z  2018-10-09T10:26:01Z  0oyT2r0L3A8aXi+zXWgMUiiH3KKibDbGYiE6Vu0KJDw=  /9wdPqTRuBUS4vx4DI3E8dABC7A37j8rkbgWmkkT09Y=
```

## Modify manifest.yml

In the [service-broker-proxy-cf](https://github.com/Peripli/service-broker-proxy-cf) repository you need to replace in the `manifest.yml` the following things:

* Administrative credentials for CF with env variables `CF_CLIENT_USERNAME` and `CF_CLIENT_PASSWORD`.
* Optional: adapt env variable `CF_CLIENT_APIADDRESS`.
* Service-Manager URL using the `SM_URL` env variable.
* Credentials for Service Manager with env variables `SM_USER` and `SM_PASSWORD`. These are the credentials obtained by the `smctl register-platform` command

In addition you can change other configurations like log level and log format.
You can also use the `application.yml` file which has lower priority than the Environment variables.

## Push

Execute:

```console
cf push -f manifest.yml
```
