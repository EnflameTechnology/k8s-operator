// Copyright (c) 2024 Enflame. All Rights Reserved.

package kubelet

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-version"

	"gcu-operator/business-logic/gcu-resource/base"
	"gcu-operator/components/cdi-manager/api"
	"gcu-operator/components/common/args"
	"gcu-operator/components/common/utils"
	"gcu-operator/pkg/common"
	"gcu-operator/pkg/logs"
)

type Kubelet struct{}

func New() api.CDIHandler {
	return &Kubelet{}
}

func (rer *Kubelet) SetCDIEnabled(args *args.PluginArgs) error {
	kubeletVersion, err := version.NewVersion(args.KubeletVersion)
	if err != nil {
		logs.Error(err, "kubelet version: %s is invalid", args.KubeletVersion)
		return err
	}
	constraint, err := version.NewConstraint("< v1.28")
	if err != nil {
		logs.Error(err)
		return err
	}
	if constraint.Check(kubeletVersion) {
		err := fmt.Errorf("kubelet version: %s lower than v1.28 does not support cdi feature", args.KubeletVersion)
		logs.Error(err)
		return err
	}
	constraint, err = version.NewConstraint(">= v1.29")
	if err != nil {
		logs.Error(err)
		return err
	}
	if constraint.Check(kubeletVersion) {
		logs.Info("kubelet version: %s is no less than v1.29, the cdi feature is enabled by default, do nothing", args.KubeletVersion)
		return nil
	}
	return setupKubelet(args)
}

func setupKubelet(args *args.PluginArgs) error {
	logs.Info("setup cdi feature for kubelet version: %s", args.KubeletVersion)
	configFile := args.HostRootMount + args.KubeletConfigFile
	if !common.FileIsExist(configFile) {
		err := fmt.Errorf("kubelet config file: %s not exist on the host", configFile)
		logs.Error(err)
		return err
	}
	content, err := base.ReadYamlToJson(configFile)
	if err != nil {
		logs.Error(err, "read config file: %s failed", configFile)
		return err
	}
	config := map[string]interface{}{}
	if err := json.Unmarshal([]byte(content), &config); err != nil {
		logs.Error(err, "json unmarshal config to struct failed, content: %s", content)
		return err
	}

	featureGates, ok := config["featureGates"]
	if ok && featureGates.(map[string]bool)["DevicePluginCDIDevices"] {
		logs.Info("cdi feature already enabled, do nothing")
		return nil
	}
	if !ok {
		config["featureGates"] = map[string]bool{"DevicePluginCDIDevices": true}
	} else {
		featureGates.(map[string]bool)["DevicePluginCDIDevices"] = true
	}
	logs.Info("set kubelet config cdi feature enabled success")
	if err := utils.ExecCommand("sh", "-c", "kill -SIGHUP $(pidof kubelet)"); err != nil {
		logs.Error("restarting kubelet using signal SIGHUP failed")
		return err
	}
	logs.Info("restart kubelet using signal SIGHUP success")
	return nil
}
