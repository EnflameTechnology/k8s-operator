// Copyright (c) 2024 Enflame. All Rights Reserved.

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/urfave/cli/v2"

	"gcu-operator/components/cdi-manager/api"
	"gcu-operator/components/cdi-manager/containerd"
	"gcu-operator/components/cdi-manager/docker"
	"gcu-operator/components/cdi-manager/kubelet"
	"gcu-operator/components/common/args"
	"gcu-operator/components/common/consts"
	"gcu-operator/components/common/utils"
	"gcu-operator/pkg/logs"
)

func main() {
	args, err := args.NewPluginArgs()
	if err != nil {
		logs.Error(err, "create plugin args failed")
		os.Exit(1)
	}
	app := cli.NewApp()
	app.Name = "Enflame K8s Device PLugin CDI Manager"
	app.Usage = "Enflame K8s Device PLugin CDI Manager For Kubernetes"
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
			Value:   "systemd",
			Usage:   "The mode of restart the container engine, only support 'systemd'. This parameter will be removed in the future",
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
		&cli.StringFlag{
			Name:    "kubelet-config",
			Value:   consts.KubeletConfigFile,
			Usage:   "the absolute path where the kubelet config file name",
			EnvVars: []string{"KUBELET_CONFIG"},
		},
		&cli.StringFlag{
			Name:    "cdi-config-path",
			Value:   consts.DefaultCDIConfigPath,
			Usage:   "the absolute path to the folder where the cdi config files is located",
			EnvVars: []string{"CDI_CONFIG_PATH"},
		},
		&cli.BoolFlag{
			Name:    "cdi-enabled",
			Usage:   "Whether to open cdi mode for the container engine",
			Value:   false,
			EnvVars: []string{"CDI_ENABLED"},
		},
	}

	logs.Info("k8s device plugin cdi manager work start...")
	if err = app.Run(os.Args); err != nil {
		logs.Error(err, "run enflame container toolkit failed")
		os.Exit(1)
	}
}

func start(args *args.PluginArgs) error {
	if !args.CDIEnabled {
		logs.Info("skip the config cdi process with cdiEnabled is false")
		logs.Warn("if you need to disable the cdi config on the premise of cdi already enabled, configure it manually")
		return nil
	}
	if err := kubelet.New().SetCDIEnabled(args); err != nil {
		return err
	}
	cdiHandlers := map[string]api.CDIHandler{
		consts.DOCKER:     docker.New(),
		consts.CONTAINERD: containerd.New(),
	}
	if err := cdiHandlers[args.ContainerEngine].SetCDIEnabled(args); err != nil {
		return err
	}
	if err := Restart(args); err != nil {
		return err
	}
	return nil
}

func Restart(args *args.PluginArgs) error {
	if args.RestartMode != "systemd" {
		err := fmt.Errorf("only supports restart the container engine through systemd, but found: %s", args.RestartMode)
		logs.Error(err)
		return err
	}
	params := map[string]interface{}{
		"engine":        args.ContainerEngine,
		"cdiConfigPath": args.CDIConfigPath,
		"cdiEnabled":    args.CDIEnabled,
		"hostRoot":      args.HostRootMount,
	}
	if err := utils.Restart(&args.Args, checkNeedRestart, params); err != nil {
		return err
	}
	return nil
}

func checkNeedRestart(params map[string]interface{}) (bool, error) {
	if params["engine"] == consts.CONTAINERD {
		logs.Info("restart containerd ensure that the config file takes effect")
		return true, nil
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := exec.Command("sh", "-c", fmt.Sprintf("chroot %s docker info", params["hostRoot"]))
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		logs.Error(err, "get docker info failed: %s", stderrBuf.String())
		return false, err
	}
	scanner := bufio.NewScanner(&stdoutBuf)
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, params["cdiConfigPath"].(string)) {
			found = true
			break
		}
	}
	if found != params["cdiEnabled"].(bool) {
		logs.Info("docker cdi feature is taking effect: %v, but cdi enabled is: %v, need restart", found, params["cdiEnabled"])
		return true, nil
	}
	logs.Info("docker cdi feature is taking effect: %v, and cdi enabled is: %v, need not restart", found, params["cdiEnabled"])
	return false, nil
}
