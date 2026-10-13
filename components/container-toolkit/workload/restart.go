// Copyright (c) 2024 Enflame. All Rights Reserved.

package workload

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"gcu-operator/components/common/args"
	"gcu-operator/components/common/consts"
	"gcu-operator/components/common/utils"
	"gcu-operator/pkg/logs"
)

func Restart(args *args.ToolkitArgs) error {
	params := map[string]interface{}{
		"engine":                args.ContainerEngine,
		"enflameRuntimeEnabled": args.EnflameRuntimeEnabled,
		"hostRoot":              args.HostRootMount,
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
	cmd := exec.Command("sh", "-c", fmt.Sprintf("chroot %s docker info|grep 'Default Runtime'|awk '{print $3}'", params["hostRoot"]))
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		logs.Error(err, "get docker default runtime failed: %s", stderrBuf.String())
		return false, err
	}
	defaultRuntime := strings.Trim(stdoutBuf.String(), "\n")
	if params["enflameRuntimeEnabled"] == (defaultRuntime == "enflame") {
		logs.Info("enflame runtime enabled is %v, docker default runtime is %s, need not start",
			params["enflameRuntimeEnabled"], defaultRuntime)
		return false, nil
	}
	logs.Info("enflame runtime enabled is %v, docker default runtime is %s, need start", params["enflameRuntimeEnabled"],
		defaultRuntime)
	return true, nil
}
