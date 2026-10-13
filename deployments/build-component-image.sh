#!/bin/bash
# Copyright (c) 2022 Enflame. All Rights Reserved.
#

function usage() {
    cat <<EOF
Usage: ./build-component-image.sh [OPTIONS]... [COMMANDS]

Description:
  This script is used to build and save the image, you can edit config.json to modify the image information you want to build.

Options:
  --os        Specify the operating system for the image.
  --cli       Specify the CLI tool for building the image.

Commands(Required, must be located at the end of the command line):
    build       Build the image
    clean       Clean relate build files

Examples:
    ./build-component-image.sh build
    ./build-component-image.sh --cli ctr build
    ./build-component-image.sh --cli ctr --os ubuntu build
EOF
}

function build() {
    arch=`uname --p`
    source /etc/os-release
    nameID=$ID$VERSION_ID
    echo -e "\033[33mBuilding component images for system nameID: $nameID, arch: $arch start...\033[0m"
    os=$(./tools/jy-read --file config.json --path common.os)
    cli=$(./tools/jy-read --file config.json --path common.cli)
    repo=$(./tools/jy-read --file config.json --path common.repo)
    namespace=$(./tools/jy-read --file config.json --path common.namespace)

    os=${os_name:-"$os"}
    cli=${cli_name:-"$cli"}
    echo -e "\033[31m$0 : cli=$cli, os=$os\033[0m"

    if [ $(./tools/jy-read --file config.json --path nfd.build) == "true" ]; then
        echo -e "\033[31m###################### NFD ###########################\033[0m"
        name=$(./tools/jy-read --file config.json --path nfd.name)
        tag=$(./tools/jy-read --file config.json --path nfd.tag)
        cd ./component-images/nfd
        ./build-image.sh --cli $cli --os $os --name $name --tag $tag --repo $repo --namespace $namespace build
        cd -
    fi
    if [ $(./tools/jy-read --file config.json --path driver.build) == "true" ]; then
        echo -e "\033[31m#################### DRIVER ##########################\033[0m"
        driver_path=$(./tools/jy-read --file config.json --path driver.path)
        if [ -z "$driver_path" ]; then
            echo -e "\033[31mERROR: Invalid path: '$driver_path', the absolute path of driver folder can not be empty\033[0m"
            exit 1
        fi
        name=$(./tools/jy-read --file config.json --path driver.name)
        tag=$(./tools/jy-read --file config.json --path driver.tag)
        cd ./component-images/driver
        ./build-image.sh  --cli $cli --os $os --name $name --tag $tag --repo $repo --namespace $namespace --path $driver_path build
        cd -
    fi
    if [ $(./tools/jy-read --file config.json --path runtime.build) == "true" ]; then
        echo -e "\033[31m################# CONTAINER TOOLKIT ##################\033[0m"
        name=$(./tools/jy-read --file config.json --path runtime.name)
        tag=$(./tools/jy-read --file config.json --path runtime.tag)
        cd ./component-images/toolkit
        ./build-image.sh  --cli $cli --os $os --name $name --tag $tag --repo $repo --namespace $namespace build
        cd -
    fi
    if [ $(./tools/jy-read --file config.json --path plugin.build) == "true" ]; then
        echo -e "\033[31m################# K8S PLUGIN ##################\033[0m"
        name=$(./tools/jy-read --file config.json --path plugin.name)
        tag=$(./tools/jy-read --file config.json --path plugin.tag)
        cd ./component-images/plugin
        ./build-image.sh  --cli $cli --os $os --name $name --tag $tag --repo $repo --namespace $namespace build
        cd -
    fi
    if [ $(./tools/jy-read --file config.json --path nodeExporter.build) == "true" ]; then
        echo -e "\033[31m################# NODE EXPORTER ##################\033[0m"
        name=$(./tools/jy-read --file config.json --path nodeExporter.name)
        tag=$(./tools/jy-read --file config.json --path nodeExporter.tag)
        cd ./component-images/node-exporter
        ./build-image.sh  --cli $cli --os $os --name $name --tag $tag --repo $repo --namespace $namespace build
        cd -
    fi
    if [ $(./tools/jy-read --file config.json --path gcuExporter.build) == "true" ]; then
        echo -e "\033[31m################# GCU EXPORTER ##################\033[0m"
        name=$(./tools/jy-read --file config.json --path gcuExporter.name)
        tag=$(./tools/jy-read --file config.json --path gcuExporter.tag)
        cd ./component-images/gcu-exporter
        ./build-image.sh  --cli $cli --os $os --name $name --tag $tag --repo $repo --namespace $namespace build
        cd -
    fi
    if [ $(./tools/jy-read --file config.json --path gfd.build) == "true" ]; then
        echo -e "\033[31m################## GFD ##########################\033[0m"
        name=$(./tools/jy-read --file config.json --path gfd.name)
        tag=$(./tools/jy-read --file config.json --path gfd.tag)
        cd ./component-images/gfd
        ./build-image.sh  --cli $cli --os $os --name $name --tag $tag --repo $repo --namespace $namespace build
        cd -
    fi
    if [ $(./tools/jy-read --file config.json --path gsp.build) == "true" ]; then
        echo -e "\033[31m################## GSP ##########################\033[0m"
        name=$(./tools/jy-read --file config.json --path gsp.name)
        tag=$(./tools/jy-read --file config.json --path gsp.tag)
        compatVersion=$(./tools/jy-read --file config.json --path gsp.compatVersion)
        if [[ $compatVersion == "" ]]; then
            echo -e "\033[31mgsp.compatVersion in config.json must be v1(k8s1.24+), v1beta1(k8s1.18, k8s 1.20, k8s1.22)\033[0m\n"
            exit 1
        fi
        cd ./component-images/gsp
        ./build-image.sh  --cli $cli --os $os --name $name --tag $tag --repo $repo --namespace $namespace --compatVersion $compatVersion build
        cd -
    fi
    if [ $(./tools/jy-read --file config.json --path gdp.build) == "true" ]; then
        echo -e "\033[31m################## GDP ##########################\033[0m"
        name=$(./tools/jy-read --file config.json --path gdp.name)
        tag=$(./tools/jy-read --file config.json --path gdp.tag)
        cd ./component-images/gdp
        ./build-image.sh  --cli $cli --os $os --name $name --tag $tag --repo $repo --namespace $namespace build
        cd -
    fi
    if [ $(./tools/jy-read --file config.json --path kdm.build) == "true" ]; then
        echo -e "\033[31m################## GDM ##########################\033[0m"
        name=$(./tools/jy-read --file config.json --path kdm.name)
        tag=$(./tools/jy-read --file config.json --path kdm.tag)
        cd ./component-images/kdm
        ./build-image.sh  --cli $cli --os $os --name $name --tag $tag --repo $repo --namespace $namespace build
        cd -
    fi
    echo -e "\033[33mBuild all images success!\033[0m\n"
}

function clean() {
    echo -e "\033[33mClean all build files start...\033[0m\n"
    cd ./component-images/nfd
    ./build-image.sh clean
    cd -
    cd ./component-images/driver
    ./build-image.sh clean
    cd -
    cd ./component-images/toolkit
    ./build-image.sh clean
    cd -
    cd ./component-images/plugin
    ./build-image.sh clean
    cd -
    cd ./component-images/node-exporter
    ./build-image.sh clean
    cd -
    cd ./component-images/gcu-exporter
    ./build-image.sh clean
    cd -
    cd ./component-images/gfd
    ./build-image.sh clean
    cd -
    cd ./component-images/gdp
    ./build-image.sh clean
    cd -
    cd ./component-images/gsp
    ./build-image.sh clean
    cd -
    cd ./component-images/kdm
    ./build-image.sh clean
    cd -
    echo -e "\033[33mClean all build files success!\033[0m\n"
}

function main() {
    [[ "$EUID" -ne 0 ]] && { echo "[INFO] Run as non-root"; }

    case $1 in
        --help | -h)
            usage
            exit 0;;
        "build")
            build;;
        "clean")
            clean;;
        *)
            echo "ERROR: unknown command: $1, Please make sure you entered the correct command(build or clean)?"
            usage
            exit 1;;
    esac
}

while true :
    do
        [[ "$#" -lt 1 ]] && { echo "Usage error: Please make sure that you have entered the correct command?"; usage; exit 1;}
        case $1 in
            "" | build | clean)
                break;;
            --os)
                os_name=$2
                shift;;
            --cli)
                cli_name=$2
                shift;;
            --help)
                usage
                exit 0;;
            *)
                echo "Usage error: unknown parameter: $1, Please refer the usage to execute the command"
                usage
                exit 1;;
        esac
    shift
done

main "$@"
