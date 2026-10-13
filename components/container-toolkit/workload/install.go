// Copyright (c) 2024 Enflame. All Rights Reserved.

package workload

import (
	"gcu-operator/components/common/utils"
	"gcu-operator/pkg/logs"
)

func InstallToolkit() error {
	if err := utils.ExecCommand("sh", "-c", "/work/enflame-container-toolkit_*.run"); err != nil {
		logs.Error(err)
		return err
	}
	logs.Info("install enflame container toolkit success")
	cmdline := "nsenter --mount=/proc/1/ns/mnt -- ldconfig"
	if err := utils.ExecCommand("sh", "-c", cmdline); err != nil {
		return err
	}
	logs.Info("refresh host shared library cache successfully")
	return nil
}
