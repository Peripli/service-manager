# Walkthrough

To follow the walkthrough you need to have Service Manager running either on CloudFoundry or on Kubernetes. Also you need the Service Broker Proxies for Kubernetes and/or CloudFoundry as well as the `smctl` command line tool. Please follow the [installation instructions](https://github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/blob/master/docs/install/README.md) if you don't already have the needed components.

This walkthrough uses the following setup:
- Service Manager running on [minikube](https://github.com/kubernetes/minikube)
- Service Broker running on [minikube](https://github.com/kubernetes/minikube)
- Service Broker Proxy for Kubernetes running on [minikube](https://github.com/kubernetes/minikube)
- Service Broker Proxy for CloudFoundry running on [cfdev](https://github.com/cloudfoundry-incubator/cfdev)

However, everything should work the same regardless of where you set up the components, provided that there is visibility from the `service broker proxies` to the `service manager` and from the `service manager` to the `service brokers`.

## Step 1 - Installing the Service Broker

We're going to deploy the sample service broker. For the purposes of this guide, we'll be using the Service Catalog's demo service broker as it's lightweight and has all the necessary functionalities.
To install the service broker, please follow the [installation guide](https://github.com/kubernetes-incubator/service-catalog/blob/master/charts/ups-broker/README.md) 


## Step 2 - Registering the Service Broker

We haven't registered any service brokers inside the service manager, so upon querying it should return an empty list:

```console
$ smctl list-brokers
No brokers registered.
```


### CloudFoundry
Because the service manager's job is to propagate the registrations to the respective platforms and as there are no registrations currently, the query to the platform's broker registry returns that there are no brokers provided by the service manager.

```console
$ cf service-brokers
Getting service brokers as admin...

name         url
p-mysql      https://p-mysql.dev.cfdev.sh:443
p.rabbitmq   http://10.144.0.146:8080
redis-odb    http://10.144.0.147:12345
```
Note: Service brokers provided by the service manager have a well known pattern which is `sm-<broker-name>`

### Kubernetes
```console
$ svcat get brokers
  NAME   NAMESPACE   URL   STATUS
+------+-----------+-----+--------+
```

We'll register the service broker by executing the `register-broker` command:

```console
$ smctl register-broker ups-broker http://192.168.64.2:30445 "UPS Broker" -b admin:admin

ID                                    Name        URL                        Description  Created               Updated
------------------------------------  ----------  -------------------------  -----------  --------------------  --------------------
a0d0a401-3048-4db7-a6d5-c5098be84d6d  ups-broker  http://192.168.64.2:30445  UPS Broker   2018-10-08T11:10:20Z  2018-10-08T11:10:20Z
```

We get in return the whole information for the service-broker as registered in the service manager.

## Step 3 - Ensure broker registration is propagated

After the service broker is registered in the service manager, the platforms will periodically synchronize the state in the platform and that in the service manager. Thus when we list the brokers in the respective platform, we should see that the broker we registered inside the service manager has been propagated.

### CloudFoundry

```console
$ cf service-brokers
Getting service brokers as admin...

name                                            url
p-mysql                                         https://p-mysql.dev.cfdev.sh:443
p.rabbitmq                                      http://10.144.0.146:8080
redis-odb                                       http://10.144.0.147:12345
sm-ups-broker  https://cfproxy.dev.cfdev.sh/v1/osb/a0d0a401-3048-4db7-a6d5-c5098be84d6d
```

### Kubernetes
```console
$ svcat get brokers
                      NAME                        NAMESPACE                                                 URL                                                 STATUS
+-----------------------------------------------+-----------+-------------------------------------------------------------------------------------------------+--------+
  sm-ups-broker              http://service-broker-proxy.service-broker-proxy:80/v1/osb/a0d0a401-3048-4db7-a6d5-c5098be84d6d   Ready
```

## Step 4 - Ensure the services are visible 

After the service broker is registered in each platform, its services and plans are made available to everyone. 

The service broker provides a set of services (`user-provided-service`, `user-provided-service-single-plan` and `user-provided-service-with-schemas`) each with its respective plans:

### CloudFoundry
```console
$ cf marketplace
Getting services from marketplace in org test / space test as admin...
OK

service                              plans                                    description
p-mysql                              10mb, 20mb                               MySQL databases on demand
p.rabbitmq                           solo, cluster                            RabbitMQ Dedicated Instance
p.redis                              cache-small, cache-medium, cache-large   Redis service to provide on-demand dedicated instances configured as a cache.
user-provided-service                default, premium                         A user provided service
user-provided-service-single-plan    default                                  A user provided service
user-provided-service-with-schemas   default                                  A user provided service
```

### Kubernetes
```console
$ svcat get classes
                 NAME                  NAMESPACE         DESCRIPTION
+------------------------------------+-----------+-------------------------+
  user-provided-service                            A user provided service
  user-provided-service-single-plan                A user provided service
  user-provided-service-with-schemas               A user provided service
```

```console
$ svcat get plans
   NAME     NAMESPACE                 CLASS                           DESCRIPTION
+---------+-----------+------------------------------------+--------------------------------+
  default               user-provided-service                Sample plan description
  premium               user-provided-service                Premium plan
  default               user-provided-service-single-plan    Sample plan description
  default               user-provided-service-with-schemas   Plan with parameter and
                                                             response schemas
```

## Step 5 - Create Service Instance

Now that we see the `user-provided-service` in both platforms we can go ahead and create a new service instance.

### CloudFoundry
```console
$ cf create-service user-provided-service default ups
Creating service instance ups in org test / space test as admin...
OK
```

### Kubernetes
```console
$ svcat provision ups --class user-provided-service --plan default
  Name:        ups
  Namespace:   default
  Status:
  Class:       user-provided-service
  Plan:        default

Parameters:
  No parameters defined
```

```console
$ svcat describe instance ups
  Name:        ups
  Namespace:   default
  Status:      Ready - The instance was provisioned successfully @ 2018-10-08 12:07:31 +0000 UTC
  Class:       user-provided-service
  Plan:        default

Parameters:
  No parameters defined

Bindings:
No bindings defined
```

## Step 6 - Create a new Service Binding

Now that we have created `ups` instance of the `user-provided-service` we can go ahead and create a service binding for it:

### CloudFoundry

For CloudFoundry it's easiest to create a service key:

```console
$ cf create-service-key ups ups-key
Creating service key ups-key for service instance ups as admin...
OK
```

```console
$ cf service-key ups ups-key
Getting key ups-key for service instance ups as admin...

{
 "special-key-1": "special-value-1",
 "special-key-2": "special-value-2"
}
```

### Kubernetes

```console
$ svcat bind ups
  Name:        ups
  Namespace:   default
  Status:
  Secret:      ups
  Instance:    ups

Parameters:
  No parameters defined
```

```console
$ svcat describe binding ups --show-secrets
  Name:        ups
  Namespace:   default
  Status:      Ready - Injected bind result @ 2018-10-08 12:10:33 +0000 UTC
  Secret:      ups
  Instance:    ups

Parameters:
  No parameters defined

Secret Data:
  special-key-1   special-value-1
  special-key-2   special-value-2
```

## Step 7 - Delete the service binding

Now let's clean up the resources we've created:

### CloudFoundry

```console
$ cf delete-service-key ups ups-key

Really delete the service key ups-key?> y
Deleting key ups-key for service instance ups as admin...
OK
```

### Kubernetes

```console
$ svcat unbind ups
deleted ups
```

Listing the bindings should show that the `ups` binding is gone:
```console
$ svcat get bindings
  NAME   NAMESPACE   INSTANCE   STATUS
+------+-----------+----------+--------+
```

## Step 8 - Delete the service instance

Next, we can deprovision the service instance:

### CloudFoundry

```console
$ cf delete-service ups

Really delete the service ups?> y
Deleting service ups in org test / space test as admin...
OK
```

### Kubernetes

```console
$ svcat deprovision ups
deleted ups
```

Listing the instances should show that the `ups` instance is gone:

```console
$ svcat get instances
  NAME   NAMESPACE   CLASS   PLAN   STATUS
+------+-----------+-------+------+--------+
```

## Step 9 - Delete the service broker

We can try and delete the service broker in the respective platform, but due to the architecture of the service manager it will reappear after a configured period of time:

### CloudFoundry

```console
$ cf delete-service-broker sm-ups-broker

Really delete the service-broker sm-a0d0a401-3048-4db7-a6d5-c5098be84d6d?> y
Deleting service broker sm-a0d0a401-3048-4db7-a6d5-c5098be84d6d as admin...
OK
```

Listing the brokers after a while we will see that the service broker is recreated:

```console
$ cf service-brokers
Getting service brokers as admin...

name                                            url
p-mysql                                         https://p-mysql.dev.cfdev.sh:443
p.rabbitmq                                      http://10.144.0.146:8080
redis-odb                                       http://10.144.0.147:12345
sm-ups-broker   https://cfproxy.dev.cfdev.sh/v1/osb/a0d0a401-3048-4db7-a6d5-c5098be84d6d
```

### Kubernetes

```console
$ kubectl delete clusterservicebrokers sm-ups-broker
clusterservicebroker.servicecatalog.k8s.io "sm-a0d0a401-3048-4db7-a6d5-c5098be84d6d" deleted
```

And if we start watching for the ClusterServiceBroker resource we'll see that the service broker is registered again:

```console
$ kubectl get clusterservicebrokers -w
NAME                                            AGE
sm-a0d0a401-3048-4db7-a6d5-c5098be84d6d   0s
sm-a0d0a401-3048-4db7-a6d5-c5098be84d6d   1s
```

To delete the service broker, you need to remove its registration from the service manager itself:

```console
$ smctl delete-broker ups-broker
Broker with name: ups-broker successfully deleted
```

And after the configured period of time, the platforms will resync the state from the service manager, removing the service broker:

### CloudFoundry

```console
$ cf service-brokers
Getting service brokers as admin...

name         url
p-mysql      https://p-mysql.dev.cfdev.sh:443
p.rabbitmq   http://10.144.0.146:8080
redis-odb    http://10.144.0.147:12345
```

### Kubernetes
```console
$ svcat get brokers
  NAME   NAMESPACE   URL   STATUS
+------+-----------+-----+--------+
```