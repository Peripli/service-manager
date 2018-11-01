#!/usr/bin/env bash
# Copyright 2018 The Service Manager Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

while [ $# -gt 0 ]; do
    case "$1" in
    --sbproxy.directory=*)
        SBPROXY_DIR="${1#*=}";;
    --app.username=*)
        APP_USERNAME="${1#*=}";;
    --app.password=*)
        APP_PASSWORD="${1#*=}";;
    --sm.url=*)
        SM_URL="${1#*=}";;
    --sm.user=*)
        SM_USER="${1#*=}";;
    --sm.password=*)
        SM_PASSWORD="${1#*=}";;
    --sm.osb.api.path=*)
        SM_OSB_API_PATH="${1#*=}";;
    *) echo "Invalid option $1"
        exit 1
    esac
    shift
done

cd ${SBPROXY_DIR}

helm install charts/service-broker-proxy-k8s --name service-broker-proxy --namespace service-broker-proxy \
    --set config.sm.url=${SM_URL} --set sm.user=${SM_USER} --set sm.password=${SM_PASSWORD} \
    --set config.sm.osb_api_path=${SM_OSB_API_PATH} --set app.username=${APP_USERNAME} --set app.password=${APP_PASSWORD}


