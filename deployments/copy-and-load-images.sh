#!/bin/bash
#
# Copyright (c) 2024 Enflame. All Rights Reserved.
#

if [ -f "./build-image.conf" ]; then
    source ./build-image.conf
else
    echo "build-image.conf is not found, exit."; exit 0
fi

if ! command -v kubectl > /dev/null 2>&1; then
    echo "kubectl is not found, exit."; exit 0
fi

function loadImage() {
    local_ip=$(hostname -I)
    nodes=$(kubectl get nodes -o custom-columns=NAME:.metadata.name --no-headers)
    echo "cluster node name list:"
    echo $nodes
    for nodeName in $nodes
        do
            ip=$(kubectl get node $nodeName -o jsonpath='{.status.addresses[?(@.type=="InternalIP")].address}')
            isCurrentNode=$(echo $local_ip | grep -wq $ip &&  echo "true" || echo "false")
            if [[ $isCurrentNode == "false" ]]; then
                container_runtime_version=$(kubectl get node $nodeName -o jsonpath='{.status.nodeInfo.containerRuntimeVersion}')
                container_runtime=${container_runtime_version%%://*}
                printf "Copy image to node:%s\n" $ip
                scp ./images/$1 "${ip}:/home"
                if [[ $container_runtime == "containerd" ]]; then
                    echo -e "\033[33mLoad image in host ${ip} with ctr....\033[0m\n"
                    ssh ${ip} "docker load -i /home/${1}"
                    ssh ${ip} "ctr -n $namespace image import /home/${1}"
                else
                    ssh ${ip} "$CLI_NAME load -i /home/${1}"
                fi
                echo -e "\033[33mLoad image $1 on node $ip success\033[0m\n"
            fi
    done
}

function loadImageAllNodes() {
    echo -e "\033[33mLoad images to all cluster nodes start...\033[0m\n"
    imagesList=(`ls ./images`)
    for imagePackage in "${imagesList[@]}"
        do
            echo "Loading image package: $imagePackage"
            loadImage $imagePackage
    done
    echo -e "\033[33mLoad image to cluster nodes success\033[0m\n"
}

function main() {
    [[ "$EUID" -ne 0 ]] && { echo "[INFO] Run as non-root"; }

    loadImageAllNodes
}

main "$@"
