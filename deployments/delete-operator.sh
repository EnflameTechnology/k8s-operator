#!/bin/bash
# Copyright (c) 2022 Enflame. All Rights Reserved.
#

RELEASE_NAME="gcu-operator"
NAMESPACE="kube-system"

function usage() {
  cat <<EOF
Usage: ./delete-operator.sh <param name> <param value>
Description: This script is used to delete gcu upgrade manager component
Param:
    --name        The release name of gcu operator, default gcu-operator.
    --namespace   The release namespace of gcu operator, default kube-system.
Example:
    ./delete-operator.sh --namespace kube-system
EOF
}

function delete_gcu_reaource() {
    echo "delete gcu resource start..."
    for time in 1 2 3
    do
        kubectl delete gr --all
        if [ $? -eq 0 ]; then
            break
        fi
    done
    echo "delete gcu resource finish"
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

delete_gcu_reaource

set -o errexit

printf "Uninstall operator release %s in namespace:%s start...\n" $RELEASE_NAME $NAMESPACE

if ! command -v helm >/dev/null 2>&1; then
    echo "helm not found, please install helm first!"
    exit 0
fi

helm uninstall $RELEASE_NAME -n $NAMESPACE
kubectl delete crd gcuresources.topscloud.enflame.com
