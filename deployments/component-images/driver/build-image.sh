#!/bin/bash
#
# Copyright 2024 Enflame. All Rights Reserved.
#

if [ -f "./get_env.sh" ]; then
    source ./get_env.sh
else
    echo "./get_env.sh is not found, exit."; exit 0
fi

function usage() {
  cat <<EOF
Usage: ./build-image.sh [OPTIONS]... [COMMANDS]

Description:
  This script is used to build and save the image for a component.

Options:
  --name      Specify the name of the image to build.
  --tag       Specify the tag for the image.
  --repo      Specify the repository for the image.
  --os        Specify the operating system for the image.
  --cli       Specify the CLI tool for building the image.

Commands(Required, must be located at the end of the command line):
  build       Build the image.
  clean       Clean up related build files.

Examples:
  ./build-image.sh --cli docker --os ubuntu build
  ./build-image.sh --cli nerdctl --os ubuntu build
EOF
}

function checkOldImage() {
    local checkImage

    if [ "$cli" == "nerdctl" ]; then
        checkImage=$($cli --namespace $namespace images | grep $imageName | grep -F $imageTag)
    elif [ "$cli" == "ctr" ]; then
        checkImage=$($cli --namespace $namespace images list | grep $imageName | grep -F $imageTag)
    else
        checkImage=$($cli images | grep $imageName | grep -F $imageTag)
    fi

    if [ -n "$checkImage" ]; then
        echo "WARN: Image $image already exists, do you want to overwrite this old image? (y/yes, n/no)"
        read input
        case $input in
            [yY]*)
                return
                ;;
            [nN]*)
                echo "Exit the build process"
                echo "Note: Maybe you can refer to the usage to modify the build parameters and rebuild this image"
                exit 0
                ;;
            *)
                echo "WARN: Exit the build process due to unknown input parameters: $input"
                exit 1
                ;;
        esac
    fi
}

function buildWithCli() {
    local osType=$1
    local dockerfilePath="dockerfiles/Dockerfile.${osType}"
    local buildCmd

    case "$cli" in
        nerdctl)
            buildCmd="$cli --namespace $namespace build -t $image --file $dockerfilePath ."
            ;;
        ctr)
            buildCmd="docker build -t $image --file $dockerfilePath ."
            ;;
        *)
            buildCmd="$cli build -t $image --file $dockerfilePath ."
            ;;
    esac

    echo "$buildCmd"
    eval "$buildCmd"
}

function buildImage() {
    source /etc/os-release
    nameID="${ID}${VERSION_ID}"
    echo -e "\033[33mBuilding $image for system nameID: $nameID start...\033[0m"
    checkOldImage

    if [ -z "$driverPath" ] || [ ! -d "$driverPath" ]; then
        echo -e "\033[31mERROR: Invalid path: '$driverPath', please make sure you specify the correct absolute path of driver folder\033[0m"
        exit 1
    fi
    cp $driverPath/enflame*.run ./

    case "$nameID" in
        ubuntu*)
            buildWithCli "$nameID"
            ;;
        openEuler* | H3Linux*)
            buildWithCli "openeuler"
            ;;
        tencentos*)
            buildWithCli "tlinux"
            ;;
        *)
            echo -e "\033[31mERROR: container toolkit 2.0 is not support system: $nameID\033[0m"
            exit 1
            ;;
    esac

    saveImage
}

function saveImage() {
    imagesDir="../../images"
    if [ ! -d $imagesDir ]; then
        mkdir $imagesDir
    fi

    IMAGE_PACKAGE_NAME="${imageName}.tar"
    if [ "$cli" == "nerdctl" ]; then
        $cli --namespace $namespace save -o $imagesDir/$IMAGE_PACKAGE_NAME $image
    elif [ "$cli" == "ctr" ]; then
        docker save -o $imagesDir/$IMAGE_PACKAGE_NAME $image
        $cli --namespace $namespace image import $imagesDir/$IMAGE_PACKAGE_NAME
    else
        $cli save -o $imagesDir/$IMAGE_PACKAGE_NAME $image
    fi
}

function cleanAll() {
    echo -e "\033[33mclean driver build files...\033[0m"
    rm -fr enflame*.run
    echo -e "\033[33mclean success\033[0m"
}

function main() {
    [[ "$EUID" -ne 0 ]] && { echo "[INFO] Run as non-root"; }
    case $1 in
    "build")
        cleanAll
        buildImage;;
    "clean")
        cleanAll;;
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
                os=$2
                shift;;
            --cli)
                cli=$2
                shift;;
            --name)
                imageName=$2
                shift;;
            --tag)
                imageTag=$2
                shift;;
            --repo)
                repo=$2
                shift;;
            --path)
                driverPath=$2
                shift;;
            --namespace)
                namespace=$2
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

if [[ $imageTag == "" ]] || [[ $imageTag == "default" ]]; then
    imageTag=$IMAGE_TAG
fi

image="${repo}/${imageName}:${imageTag}"

main "$@"
