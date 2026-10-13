#!/bin/bash
#
# Copyright 2024 Enflame. All Rights Reserved.
#

if [ -f "../../g_version.txt" ]; then
    source ../../g_version.txt
else
    echo "../../g_version.txt is not found, exit."; exit 0
fi

IMAGE_TAG=$GCU_EXPORTER_VERSION

TOPSCLOUD_PATH="../../../"
BUILD_IMAGE_PATH="$TOPSCLOUD_PATH/gcu-exporter_*"
SAVE_IMAGE_PATH="$TOPSCLOUD_PATH/gcu-exporter_*/images"