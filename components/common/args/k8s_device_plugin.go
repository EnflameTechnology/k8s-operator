// Copyright (c) 2024 Enflame. All Rights Reserved.

package args

import (
	"fmt"
	"gcu-operator/pkg/kube"
	"gcu-operator/pkg/logs"
	"strings"

	"github.com/urfave/cli/v2"
)

type PluginArgs struct {
	Args
	KubeletConfigFile string
	KubeletVersion    string
}

func NewPluginArgs() (*PluginArgs, error) {
	clientSet, err := kube.GetClientSet()
	if err != nil {
		logs.Error(err, "get kube client set failed")
		return nil, err
	}
	return &PluginArgs{
		Args: Args{
			ClientSet: clientSet,
		},
	}, nil
}

func (args *PluginArgs) ParseArgs(ctx *cli.Context) error {
	if err := args.SetVersions(); err != nil {
		return err
	}
	if err := args.ParseAndValidateBaseArgs(ctx); err != nil {
		return err
	}
	args.KubeletConfigFile = ctx.String("kubelet-config")
	return nil
}

func (args *PluginArgs) SetVersions() error {
	versions, err := args.GetVersion()
	if err != nil {
		return err
	}
	if versions.ContainerRuntimeVersion == "" {
		err := fmt.Errorf("container runtime version must set, please check container-toolkit pod envs")
		logs.Error(err)
		return err
	}
	args.ContainerEngine = strings.Split(versions.ContainerRuntimeVersion, "://")[0]

	if versions.KubeletVersion == "" {
		err := fmt.Errorf("kubelet version must set, please check k8s-device-plugin pod envs")
		logs.Error(err)
		return err
	}
	args.KubeletVersion = versions.KubeletVersion
	return nil
}
