// Copyright (c) 2024 Enflame. All Rights Reserved.

package args

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	"gcu-operator/pkg/kube"
	"gcu-operator/pkg/logs"
)

type ToolkitArgs struct {
	Args
	EnflameRuntimeEnabled bool
	EnflameRuntimeFile    string
	EnflameRuntimeConfig  string
	CDIAnnotationPrefix   string
	CDIDefaultKind        string
}

func NewToolkitArgs() (*ToolkitArgs, error) {
	clientSet, err := kube.GetClientSet()
	if err != nil {
		logs.Error(err, "get kube client set failed")
		return nil, err
	}
	return &ToolkitArgs{
		Args: Args{
			ClientSet: clientSet,
		},
	}, nil
}

func (args *ToolkitArgs) ParseArgs(ctx *cli.Context) error {
	if err := args.SetVersions(); err != nil {
		return err
	}
	if err := args.ParseAndValidateBaseArgs(ctx); err != nil {
		return err
	}
	args.EnflameRuntimeEnabled = ctx.Bool("enflame-runtime-enabled")
	args.EnflameRuntimeFile = ctx.String("enflame-runtime-file")
	args.EnflameRuntimeConfig = ctx.String("enflame-runtime-config")
	args.CDIAnnotationPrefix = ctx.String("cdi-annotation-prefix")
	args.CDIDefaultKind = ctx.String("cdi-default-kind")
	return nil
}

func (args *ToolkitArgs) SetVersions() error {
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
	return nil
}
