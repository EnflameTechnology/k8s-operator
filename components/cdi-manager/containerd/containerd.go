// Copyright (c) 2024 Enflame. All Rights Reserved.

package containerd

import (
	"fmt"
	"os"

	"gcu-operator/components/cdi-manager/api"
	"gcu-operator/components/common/args"
	"gcu-operator/components/common/consts"
	"gcu-operator/components/common/utils"
	"gcu-operator/pkg/common"
	"gcu-operator/pkg/logs"
)

type Containerd struct{}

func New() api.CDIHandler {
	return &Containerd{}
}

func (rer *Containerd) SetCDIEnabled(args *args.PluginArgs) error {
	if !common.FileIsExist(args.ConfigFile) {
		logs.Info("containerd config file: %s not found, will create default config: %s", args.ConfigFile, consts.CtrConfig)
		tmpFile := fmt.Sprintf("%s.tmp", consts.CtrConfig)
		cmdline := fmt.Sprintf("chroot %s containerd config default > %s", args.HostRootMount, tmpFile)
		if err := utils.ExecCommand("sh", "-c", cmdline); err != nil {
			return err
		}
		if err := os.Rename(tmpFile, consts.CtrConfig); err != nil {
			logs.Error(err, "rename file: %s to %s failed", tmpFile, consts.CtrConfig)
			return err
		}
		logs.Info("create containerd default config: %s success", consts.CtrConfig)
		args.ConfigFile = consts.CtrConfig
	} else {
		backupFile := fmt.Sprintf("%s%s", args.ConfigFile, consts.BackupFileSuffix)
		if common.FileIsExist(backupFile) {
			logs.Info("found containerd config: %s and backup file: %s already exist, need not backup again",
				args.ConfigFile, backupFile)
		} else {
			cmdline := fmt.Sprintf("cp %s %s", args.ConfigFile, backupFile)
			if err := utils.ExecCommand("sh", "-c", cmdline); err != nil {
				return err
			}
			logs.Info("found containerd config: %s, and backup to %s success", args.ConfigFile, backupFile)
		}
	}
	cmdline := fmt.Sprintf("chroot %s enflame-ctk runtime configure --runtime=%s --cdi.enabled --config=%s",
		args.HostRootMount, args.ContainerEngine, args.ConfigFile)
	if err := utils.ExecCommand("sh", "-c", cmdline); err != nil {
		return err
	}
	logs.Info("setup containerd config: %s for cdi enabled finish", args.ConfigFile)
	return nil
}
