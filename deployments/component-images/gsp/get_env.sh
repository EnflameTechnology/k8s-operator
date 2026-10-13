#!/bin/bash
#
# Copyright 2024 Enflame. All Rights Reserved.
#

if [ -f "../../g_version.txt" ]; then
    source ../../g_version.txt
else
    echo "../../g_version.txt is not found, exit."; exit 0
fi

IMAGE_TAG=$GCUSHARE_VERSION

TOPSCLOUD_PATH="../../../"
BUILD_IMAGE_PATH="$TOPSCLOUD_PATH/gcushare_*/gcushare-scheduler-plugin_*"
SAVE_IMAGE_PATH="$TOPSCLOUD_PATH/gcushare_*/gcushare-scheduler-plugin_*/images"
