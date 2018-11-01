#!/usr/bin/env bash

set -e

# deploy service manager
SM_ARGS=""
SBPROXY_CF_ARGS=""
while [ $# -gt 0 ]; do
    case "$1" in
    --cf.api.endpoint=*)
        SM_ARGS=("$SM_ARGS --api.endpoint=${1#*=}")
        SBPROXY_CF_ARGS=("$SBPROXY_CF_ARGS --api.endpoint=${1#*=}")
        ;;
    --cf.org.name=*)
        SM_ARGS=("$SM_ARGS --org.name=${1#*=}")
         SBPROXY_CF_ARGS=("$SBPROXY_CF_ARGS --org.name=${1#*=}")
        ;;
    --cf.space.name=*)
        SM_ARGS=("$SM_ARGS --space.name=${1#*=}")
        SBPROXY_CF_ARGS=("$SBPROXY_CF_ARGS --space.name=${1#*=}")
        ;;
    --cf.username=*)
        SM_ARGS=("$SM_ARGS $1")
        SBPROXY_CF_ARGS=("$SBPROXY_CF_ARGS $1")
        ;;
    --cf.password=*)
        SM_ARGS=("$SM_ARGS $1")
        SBPROXY_CF_ARGS=("$SBPROXY_CF_ARGS $1")
        ;;
    --sm.manifest.location=*)
        SM_ARGS=("$SM_ARGS --manifest.location=${1#*=}")
        ;;
    --sm.api.token_issuer.url=*)
        SM_ARGS=("$SM_ARGS --api.token_issuer.url=${1#*=}")
        ;;
    --postgresql.service.name=*)
        SM_ARGS=("$SM_ARGS $1")
        ;;
    --postgresql.service.plan=*)
        SM_ARGS=("$SM_ARGS $1")
        ;;
    --postgresql.service.instance=*)
        SM_ARGS=("$SM_ARGS $1")
        ;;
    --cf.proxy.app.username=*)
        SBPROXY_CF_ARGS=("$SBPROXY_CF_ARGS --app.username=${1#*=}")
        ;;
    --cf.proxy.app.password=*)
        SBPROXY_CF_ARGS=("$SBPROXY_CF_ARGS --app.password=${1#*=}")
        ;;
    --cf.proxy.manifest.location=*)
        SBPROXY_CF_ARGS=("$SBPROXY_CF_ARGS --manifest.location=${1#*=}")
        ;;
    --cf.proxy.client.username=*)
        SBPROXY_CF_ARGS=("$SBPROXY_CF_ARGS --cf.client.username=${1#*=}")
        ;;
    --cf.proxy.client.password=*)
        SBPROXY_CF_ARGS=("$SBPROXY_CF_ARGS --cf.client.password=${1#*=}")
        ;;
    *) echo "Invalid option $1"
        exit 1
    esac
    shift
done

echo ${SM_ARGS}

./deploy_service_manager_cf ${SM_ARGS}

smctl login -u ${USERNAME} -p ${PASSWORD} -a ${SM_URL}

if [ -z ${TEST_CF_PLATFORM} ]; then
    register_cf_response=$(smctl register-platfrom test-cf cloudfoundry "cloud foundry installation for e2e test" -o json)
    CF_USERNAME=$(echo ${register_cf_response} | jq -r .credentials.basic.username)
    CF_PASSWORD=$(echo ${register_cf_response} | jq -r .credentials.basic.password)

    ./deploy_cf_proxy --
fi
#
#if [ ! -z ${TEST_K8S_PLATFORM} ]; then
#    register_k8s_response=$(smctl register-platfrom test-k8s kubernetes "kubernetes cluster for e2e test" -o json)
#    K8S_USERNAME=$(echo ${register_cf_response} | jq -r .credentials.basic.username)
#    K8S_PASSWORD=$(echo ${register_cf_response} | jq -r .credentials.basic.password)
#fi

# deploy demo test service broker

smctl register-broker test-broker https://test-broker.cfapps.example.com -b random-username:random-password

# wait resync timeout

cf service-brokers
cf marketplace # see that the service is in the marketplace

kubectl get clusterservicebrokers
kubectl get clusterserviceclasses
kubectl get clusterserviceplans

cf create-service testinstance <service> <plan>
cf create-service-key testinstance testkey

# somehow assert that the calls are passing throught the cloud controller, cfproxy, service manager and broker

kubectl create namespace test-ns
svcat provision -n test-ns testinstance --plan {{test_broker_plan}} --class {{test_broker_service_class}}
svcat bind -n test-ns testinstance
svcat describe testinstance -n test-ns testinstance
kubectl get secret binding -o yaml -n test-ns

# somehow assert that the calls are passing through the service catalog, k8sproxy, service manager and broker

#cleanup

cf delete-service-key testinstance testkey
cf delete-service testinstance

svcat unbind -n test-ns testinstance
svcat deprovision -n test-ns testinstance

smctl delete-broker test-broker
smctl delete-platform cf k8s