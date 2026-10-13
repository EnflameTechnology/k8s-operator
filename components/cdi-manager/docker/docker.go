// Copyright (c) 2024 Enflame. All Rights Reserved.

package docker

import (
	"fmt"

	"gcu-operator/components/cdi-manager/api"
	"gcu-operator/components/common/args"
	"gcu-operator/components/common/consts"
	"gcu-operator/components/common/utils"
	"gcu-operator/pkg/common"
	"gcu-operator/pkg/logs"
)

type Docker struct{}

func New() api.CDIHandler {
	return &Docker{}
}

func (rer *Docker) SetCDIEnabled(args *args.PluginArgs) error {
	if common.FileIsExist(args.ConfigFile) {
		backupFile := fmt.Sprintf("%s%s", args.ConfigFile, consts.BackupFileSuffix)
		if common.FileIsExist(backupFile) {
			logs.Info("found docker config: %s and backup file: %s already exist, need not backup again",
				args.ConfigFile, backupFile)
		} else {
			cmdline := fmt.Sprintf("cp %s %s", args.ConfigFile, backupFile)
			if err := utils.ExecCommand("sh", "-c", cmdline); err != nil {
				return err
			}
			logs.Info("found docker config: %s, and backup to %s success", args.ConfigFile, backupFile)
		}
	}
	cmdline := fmt.Sprintf("chroot %s enflame-ctk runtime configure --runtime=%s --cdi.enabled --config=%s",
		args.HostRootMount, args.ContainerEngine, args.ConfigFile)
	if err := utils.ExecCommand("sh", "-c", cmdline); err != nil {
		return err
	}
	logs.Info("setup docker config: %s for cdi enabled finish", args.ConfigFile)
	return nil
}
