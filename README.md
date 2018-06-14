# service-manager

[![Build Status](https://travis-ci.org/Peripli/service-manager.svg?branch=master)](https://travis-ci.org/Peripli/service-manager)
[![Go Report Card](https://goreportcard.com/badge/github.com/Peripli/service-manager)](https://goreportcard.com/report/github.com/Peripli/service-manager)
[![Coverage Status](https://coveralls.io/repos/github/Peripli/service-manager/badge.svg?branch=master)](https://coveralls.io/github/Peripli/service-manager?branch=master)

## What is Service Manager

Service Manager is a central registry for service brokers and platforms registration. It tracks service instances creation and allows sharing of service instances between different Platform Instances.

## Setup Service Manager Components

### Deploy Service Manager

Currently the Service Manager can be deployed on CF or on Kubernetes.

For more information see:

* [Deploy to CF](deployment/cf/README.md)
* [Deploy to Kubernetes](deployment/k8s/README.md)

After the deployment you need to have the Service Manager url.

For CF/PCF Dev you can get the url with the `cf app <service-manager-app-name>` command. For example, if your *service-manager-app-name* is *service-manager* the command will be:

```sh
cf app service-manager
```

### Deploy Service Broker Proxies

In order to consume services from the Service Manager you need to have a proxy deployed in your platform instance.
For proxies to work they need to be able to access the [deployed Service Manager](#deploy-service-manager).

* [Deploy the Service Broker Proxy on CF](https://github.com/Peripli/service-broker-proxy-cf)
* [Deploy the Service Broker Proxy on Kubernetes](https://github.com/Peripli/service-broker-proxy-k8s)

### Install smctl

In order to work with the Service Manager you can install the [command line tools](https://github.com/Peripli/service-manager-cli)

### Test the setup

If you deployed the Service Manager, smctl and at least one proxy you can register an OSB compliant broker using the *smctl register-broker* command.

```sh
smctl register-broker <broker_name> <broker_url> <description>
```

The services provided by the broker should appear in the platform instances in which the proxies reside.
For example, in CF you can check with:

```sh
cf service-brokers
```

**Note:** It could take a couple of minutes before the broker appears in the platform instance.
