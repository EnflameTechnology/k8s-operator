// Copyright (c) 2024 Enflame. All Rights Reserved.

package workload

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml"

	"gcu-operator/components/common/args"
	"gcu-operator/components/common/consts"
	"gcu-operator/components/common/utils"
	"gcu-operator/pkg/common"
	"gcu-operator/pkg/logs"
)

func SetupRuntime(args *args.ToolkitArgs) error {
	if args.ContainerEngine == consts.CONTAINERD {
		logs.Info("Starting setup %s for %s", args.ConfigFile, consts.CONTAINERD)
		if err := setupRuntimeForContainerd(args); err != nil {
			return err
		}
	} else if args.ContainerEngine == consts.DOCKER {
		logs.Info("Starting setup %s for %s", args.ConfigFile, consts.DOCKER)
		if err := setupRuntimeForDocker(args); err != nil {
			return err
		}
	} else {
		err := fmt.Errorf("container engine: %s is not support", args.ContainerEngine)
		logs.Error(err)
		return err
	}
	return nil
}

func setupRuntimeForContainerd(args *args.ToolkitArgs) error {
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
	cmdline := fmt.Sprintf("enflame-ctk runtime configure --runtime=%s --enflame-set-as-default --config=%s",
		args.ContainerEngine, args.ConfigFile)
	if err := utils.ExecCommand("sh", "-c", cmdline); err != nil {
		return err
	}
	logs.Info("setup containerd config: %s finish", args.ConfigFile)
	return nil
}

func setupRuntimeForDocker(args *args.ToolkitArgs) error {
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
	cmdline := fmt.Sprintf("enflame-ctk runtime configure --runtime=%s --enflame-set-as-default --config=%s",
		args.ContainerEngine, args.ConfigFile)
	if err := utils.ExecCommand("sh", "-c", cmdline); err != nil {
		return err
	}
	logs.Info("setup docker config: %s finish", args.ConfigFile)
	return nil
}

func SetupCDI(args *args.ToolkitArgs) error {
	configFile := args.EnflameRuntimeConfig
	if !common.FileIsExist(args.EnflameRuntimeConfig) {
		err := fmt.Errorf("enflame runtime config file: %s not found", configFile)
		logs.Error(err)
		return err
	}
	backupFile := fmt.Sprintf("%s%s", configFile, consts.BackupFileSuffix)
	if common.FileIsExist(backupFile) {
		logs.Info("found enflame runtime config: %s and backup file: %s already exist, need not backup again",
			configFile, backupFile)
	} else {
		cmdline := fmt.Sprintf("cp %s %s", configFile, backupFile)
		if err := utils.ExecCommand("sh", "-c", cmdline); err != nil {
			return err
		}
		logs.Info("found enflame runtime config: %s, and back up to %s%s success", configFile, configFile,
			consts.BackupFileSuffix)
	}
	file, err := os.Open(configFile)
	if err != nil {
		logs.Error(err, "Failed to open enflame runtime config file: %s", configFile)
		return err
	}
	defer file.Close()

	var config map[string]interface{}
	decoder := toml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		logs.Error(err, "Failed to decode config: %s to map", configFile)
		return err
	}
	if err := modifyRuntimeMode(args.CDIEnabled, config, []string{}); err != nil {
		logs.Error(err)
		return err
	}
	if err := modifyCDIModeSpec(args, config, []string{}); err != nil {
		logs.Error(err)
		return err
	}

	file, err = os.Create(configFile)
	if err != nil {
		logs.Error(err, "Failed to create config file: %s", configFile)
		return err
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		logs.Error(err, "Failed to encode config: %s", configFile)
		return err
	}
	logs.Info("setup enflame runtime config: %s finish", configFile)
	return nil
}

func modifyRuntimeMode(cdiEnabled bool, config map[string]interface{}, paths []string) error {
	if len(paths) == 0 {
		paths = append(paths, "enflame-container-runtime")
	}
	tmp, ok := config[paths[0]].(map[string]interface{})
	if !ok {
		return fmt.Errorf("internal error: parse '%s' to map failed", paths[0])
	}
	if len(paths) == 1 {
		if cdiEnabled {
			if tmp["mode"] == "cdi" {
				logs.Info("enflame runtime config mode already is cdi")
			} else {
				logs.Info("modify enflame runtime config mode: %s to cdi", tmp["mode"])
				tmp["mode"] = "cdi"
			}
		} else if tmp["mode"] == "cdi" {
			logs.Info("enflame runtime config mode is cdi, but cdi enabled is false, reset it to auto")
			tmp["mode"] = "auto"
		}
		logs.Info("set enflame runtime config mode success")
		return nil
	}
	return modifyRuntimeMode(cdiEnabled, tmp, paths[1:])
}

func modifyCDIModeSpec(args *args.ToolkitArgs, config map[string]interface{}, paths []string) error {
	if len(paths) == 0 {
		paths = append(paths, "enflame-container-runtime", "modes", "cdi")
	}
	tmp, ok := config[paths[0]].(map[string]interface{})
	if !ok {
		return fmt.Errorf("internal error: parse '%s' to map failed", paths[0])
	}
	if len(paths) == 1 {
		if args.CDIEnabled {
			annotationPrefixes, ok := tmp["annotation-prefixes"].([]interface{})
			if !ok {
				return fmt.Errorf("cdi annotation prefixes: %v is not slice", tmp["annotation-prefixes"])
			}
			if !utils.HasValue(annotationPrefixes, args.CDIAnnotationPrefix) {
				logs.Info("append cdi annotation prefix: %s", args.CDIAnnotationPrefix)
				annotationPrefixes = append(annotationPrefixes, args.CDIAnnotationPrefix)
				tmp["annotation-prefixes"] = annotationPrefixes
			}
			if tmp["default-kind"] != args.CDIDefaultKind {
				logs.Info("set cdi mode default kind: %v to %s", tmp["default-kind"], args.CDIDefaultKind)
				tmp["default-kind"] = args.CDIDefaultKind
			}
			specDir, ok := tmp["spec-dirs"].([]interface{})
			if !ok {
				return fmt.Errorf("cdi spec dirs: %s is not slice", tmp["spec-dirs"])
			}
			if !utils.HasValue(specDir, args.CDIConfigPath) {
				logs.Info("append cdi spec dir: %s", args.CDIConfigPath)
				specDir = append(specDir, args.CDIConfigPath)
				tmp["spec-dirs"] = specDir
			}
		}
		logs.Info("modify cdi mode spec success")
		return nil
	}
	return modifyCDIModeSpec(args, tmp, paths[1:])
}
