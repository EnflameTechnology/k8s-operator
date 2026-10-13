/**
# Copyright 2021-2022 Enflame. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
*/

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"

	"gcu-operator/components/common/args"
	"gcu-operator/components/common/consts"
	"gcu-operator/components/common/utils"
	"gcu-operator/components/container-toolkit/workload"
	"gcu-operator/pkg/logs"
)

func main() {
	args, err := args.NewToolkitArgs()
	if err != nil {
		logs.Error(err, "create toolkit args failed")
		os.Exit(1)
	}
	app := cli.NewApp()
	app.Name = "Enflame Container-toolkit Containerization Tool"
	app.Usage = "Enflame Container-toolkit For Kubernetes"
	app.Action = func(ctx *cli.Context) error {
		return start(args)
	}
	app.Before = func(ctx *cli.Context) error {
		return args.ParseArgs(ctx)
	}

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:    "restart-container-engine",
			Value:   true,
			Usage:   "Whether to restart the container engine, docker or containerd. The recommended value is 'true'",
			EnvVars: []string{"RESTART_CONTAINER_ENGINE"},
		},
		&cli.StringFlag{
			Name:    "restart-mode",
			Value:   consts.DefaultRestartMode,
			Usage:   "The mode of restart the container engine, choose from ['signal', 'systemd']",
			EnvVars: []string{"RESTART_MODE"},
		},
		&cli.StringFlag{
			Name:    "config",
			Usage:   "The path to the config file used by docker or containerd",
			Value:   "",
			EnvVars: []string{"CONTAINER_ENGINE_CONFIG"},
		},
		&cli.StringFlag{
			Name:    "socket",
			Usage:   "The path to the socket file of docker or containerd",
			Value:   "",
			EnvVars: []string{"SOCKET_FILE"},
		},
		&cli.StringFlag{
			Name:    "host-root",
			Usage:   "Specify the path to the host root to be used when restarting docker or containerd using systemd",
			Value:   consts.DefaultHostRootMount,
			EnvVars: []string{"HOST_ROOT_MOUNT"},
		},
		&cli.BoolFlag{
			Name:    "cdi-enabled",
			Usage:   "Whether to open cdi mode for the container engine",
			Value:   false,
			EnvVars: []string{"CDI_ENABLED"},
		},
		&cli.StringFlag{
			Name:    "cdi-annotation-prefix",
			Value:   consts.DefaultCDIAnnotationPrefix,
			Usage:   "the prefix to use for CDI container annotation keys",
			EnvVars: []string{"CDI_ANNOTATION_PREFIX"},
		},
		&cli.StringFlag{
			Name:    "cdi-config-path",
			Value:   consts.DefaultCDIConfigPath,
			Usage:   "the absolute path to the folder where the cdi config files is located",
			EnvVars: []string{"CDI_CONFIG_PATH"},
		},
		&cli.StringFlag{
			Name:    "cdi-default-kind",
			Value:   consts.DefaultCDIKind,
			Usage:   "default CDI kind used by container-toolkit",
			EnvVars: []string{"CDI_DEFAULT_KIND"},
		},
		&cli.StringFlag{
			Name:    "enflame-runtime-config",
			Usage:   "Specify the path to the enflame runtime config file",
			Value:   consts.EnflameRuntimeConfig,
			EnvVars: []string{"ENFLAME_RUNTIME_CONFIG"},
		},
		&cli.BoolFlag{
			Name:    "enflame-runtime-enabled",
			Usage:   "Whether to use enflame as the default runtime for the container engine",
			Value:   true,
			EnvVars: []string{"ENFLAME_RUNTIME_ENABLED"},
		},
		&cli.StringFlag{
			Name:    "enflame-runtime-file",
			Usage:   "The path where the enflame-container-runtime binaries are located, it is required when 'ENFLAME_RUNTIME_ENABLED' is true",
			Value:   consts.EnflameRuntimeFile,
			EnvVars: []string{"ENFLAME_RUNTIME_FILE"},
		},
	}

	logs.Info("Container toolokit work start...")
	if err = app.Run(os.Args); err != nil {
		logs.Error(err, "run enflame container toolkit failed")
		os.Exit(1)
	}
}

func start(args *args.ToolkitArgs) error {
	if err := workload.InstallToolkit(); err != nil {
		return err
	}

	if err := workload.SetupCDI(args); err != nil {
		return err
	}

	if err := workload.SetupRuntime(args); err != nil {
		return err
	}

	if err := workload.Restart(args); err != nil {
		return err
	}

	if err := os.WriteFile(consts.HealthCheckFile, []byte("running"), 0644); err != nil {
		logs.Error(err, "create health check file: %s failed", consts.HealthCheckFile)
		return err
	}
	logs.Info("create health check file: %s success", consts.HealthCheckFile)
	nodeName := os.Getenv(consts.NODE_NAME)
	if err := utils.SetNodeLabelWithClientSet(args.ClientSet, nodeName, consts.ContainerToolkitReady, "true"); err != nil {
		return err
	}
	termSignals := []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, termSignals...)
	logs.Info("Receive system signal: %v", <-sigChan)
	return nil
}
