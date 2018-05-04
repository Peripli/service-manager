## How to deploy PostgreSQL on K8S Minikube:

1. Create a secret for the Postgres password
```sh
    kubectl create -f postgre-secret.yml
```

2. To create the Postgre pod run
```sh
    kubectl create -f postgre-pod.yml
```

3. To expose a service for the postgres pod run:
```sh
    kubectl create -f postgre-service.yml
```

4. Create a secret for the SM postgres db
```sh
    kubectl create -f sm-secret.yml
```

5. To create the Service Manager deployment run
```sh
    kubectl create -f sm-deployment.yml
```

6. To expose a service for the Service Manager pod run:
```sh
    kubectl create -f sm-service.yml
```
