#!/bin/bash
# Copyright (c) 2022 Enflame. All Rights Reserved.
#
RELEASE_NAME="gcu-operator"
NAMESPACE="kube-system"

function usage() {
  cat <<EOF
Usage: ./deploy-operator.sh <param name> <param value>
Description: This script is used to deploy gcushare device plugin component
Param:
    --name        The release name of gcu operator, default gcu-operator.
    --namespace   The release namespace of gcu operator, default kube-system.
Example:
    ./deploy-operator.sh --namespace kube-system
EOF
}

if ! command -v helm >/dev/null 2>&1; then
    echo "helm not found, please install helm first!"
    exit 0
fi

lsmod | grep -w enflame > /dev/null
driverInstalled=$?

gcuResourceDeploy=$(./tools/jy-read --file gcu-operator-chart/values.yaml --path gcuResource.deploy)
driverDeploy=$(./tools/jy-read --file gcu-operator-chart/values.yaml --path gcuResource.driver.deploy)

function install_operator() {
    echo "Deploy gcu-operator release start..."
    CHART_NAME=${RELEASE_NAME}-chart
    helm install $RELEASE_NAME $CHART_NAME -n $NAMESPACE
}

function install_gcu_resource() {
    if [ $gcuResourceDeploy == "true" ]; then
        echo "It is detected that gcuResource has installed together with gcu-operator, you can enter 'kubectl get gr' for detail"
        return
    fi
    # deploy example gcu-resource
    echo -e "\033[33mDo you want to create gcu-resource now?(y/yes, n/no)?\033[0m"
    read input
    case $input in
    (y | yes)
            kubectl create -f ./example/gcu-resource.yaml
            if [ $? -ne 0 ]; then
                echo "create gcu resource example failed"
            else
                echo "create gcu resource example success"
            fi;;
    (n | no)
            echo "exit"
            echo "note: you can execute command 'kubectl create -f ./example/gcu-resource.yaml' manually to create gcu resource";;
    (*)
            echo "exit"
    esac
}

function checkDriver() {
    if [ $driverInstalled == 1 ]; then
        echo -e "\033[33mWARN: It is detected that KMD is not installed on the system, the following information is useful for you:\033[0m"
        echo "#####################"
        echo "1. It is recommended that you install KMD on all GCU nodes in the k8s cluster before deploying gcu-operator"
        echo "2. You can also use gcu-operator to automatically install KMD for all GCU nodes in the cluster, then you need to do two things:
            1) The gcu-resource.yaml file provided by gcu-operator do not install drivers by default. 
                * If you deploy gcuResource together with gcu-operator(values.gcuResource.deploy=true), make sure 'gcuResource.driver.deploy=true' in values.yaml;
                * If you deploy gcuResource after installing gcu-operator(values.gcuResource.deploy=false), make sure 'spec.driver.deploy=false' in example/gcu-resource.yaml;
            2) Specify the absolute path to your driver package via fields: driver.path and set driver.build=true in config.json,
                the driver path can usually be found in the topsrider package. This path usually contains the following:
                # ll <your-path>/x86_64-linux-rel/lib/TopsPlatform_*-*_deb_amd64/driver/
                drwxr-xr-x  2 root     root         4096 Mar 13 11:30 ./
                drwx------ 10 root     root         4096 Mar 13 11:31 ../
                -rwxr--r--  1 ci_build ci_build   197547 Mar 11 16:16 enflame_peer_mem-x86_64-gcc-*.run
                -rwxr--r--  1 ci_build ci_build   259268 Mar 11 16:16 enflame_virt-x86_64-gcc-*.run
                -rwxr--r--  1 ci_build ci_build 72207575 Mar 11 16:16 enflame-x86_64-gcc-*.run

            Next, you can build the driver image according to the user guide."
        echo "#####################"
        if [ $gcuResourceDeploy == "true" ] && [ $driverDeploy == "false" ]; then
            echo -e "\033[31mError: It is detected that gcuResource will be installed together with gcu-operator, but the driver is not installed in the system. 
            You should install the driver on the node in advance, or use gcu-resource to install the driver.
            You can also get more information from the user guide.\033[0m"
            exit 1
        fi
    fi
}

while true :
    do
        [[ "$#" -lt 1 ]] && { break;}
        case $1 in
            "")
                break;;
            --name)
                RELEASE_NAME=$2
                shift;;
            --namespace)
                NAMESPACE=$2
                shift;;
            --help | -h)
                usage
                exit 0;;
            *)
                echo "Usage error! Please refer the usage to execute the command"
                usage
                exit 1;;
        esac
    shift
done

checkDriver
install_operator
install_gcu_resource