#!/bin/bash
#
# Copyright 2022 Enflame. All Rights Reserved.
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

function buildImage() {
    arch=`uname --p`
    source /etc/os-release
    nameID=$ID$VERSION_ID
    echo -e "\033[33mBuilding $image for system nameID: $nameID, arch: $arch start...\033[0m"
    checkOldImage
    # build logic

    cp binary/cdi-manager $BUILD_IMAGE_PATH/bin
    cd $BUILD_IMAGE_PATH
    cp ./dockerfiles/Dockerfile.$os ./dockerfiles/Dockerfile.$os.origin
    sed -i '/COPY bin\/config-manager \/usr\/bin\//a COPY bin/cdi-manager /usr/bin/' ./dockerfiles/Dockerfile.$os
    ./build-image.sh --cli $cli --os $os --name $imageName --tag $imageTag --repo $repo --namespace $namespace
    mv ./dockerfiles/Dockerfile.$os.origin ./dockerfiles/Dockerfile.$os
    cd -

    saveImage
}

function saveImage() {
    imagesDir="../../images"
    if [ ! -d $imagesDir ]; then
        mkdir $imagesDir
    fi
    IMAGE_PACKAGE_NAME="${imageName}.tar"
    mv $SAVE_IMAGE_PATH/$IMAGE_PACKAGE_NAME $imagesDir
}

function cleanAll() {
    echo -e "\033[33mclean k8s-device-plugin building files...\033[0m"
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
    imageTag=$ENFLAME_DEVICE_PLUGIN_VERSION
fi

image="${repo}/${imageName}:${imageTag}"

main "$@"
