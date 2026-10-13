// Copyright (c) 2024 Enflame. All Rights Reserved.

package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"gcu-operator/components/common/args"
	"gcu-operator/components/common/consts"
	"gcu-operator/pkg/common"
	"gcu-operator/pkg/logs"
)

func Restart(args *args.Args, checkNeedRestart func(map[string]interface{}) (bool, error), params map[string]interface{}) error {
	if !args.Restart {
		logs.Warn("skip restart progress, container toolkit maybe can not work")
		return nil
	}
	if common.FileIsExist(consts.RestartFlag) {
		logs.Warn("restart flag exist, will not restart containerd again")
		return nil
	}
	needRestart, err := checkNeedRestart(params)
	if err != nil {
		return err
	}
	if !needRestart {
		return nil
	}
	switch args.RestartMode {
	case consts.SIGNAL:
		logs.Info("restart %s with %s is recommended", args.ContainerEngine, consts.SIGNAL)
		return signalRestart(args.ContainerEngine, args.SocketFile)
	case consts.SYSTEMD:
		logs.Warn("restart %s with %s is a dangerous operation, and all containers created with %s can be corrupted",
			args.ContainerEngine, consts.SYSTEMD, args.ContainerEngine)
		return systemdRestart(args.ContainerEngine, args.HostRootMount)
	default:
		err := fmt.Errorf("restart mode: %s is invalid", args.RestartMode)
		return err
	}
}

func signalRestart(engine, socket string) error {
	if !common.FileIsExist(socket) {
		err := fmt.Errorf("socket file: %s not found, can not signal to %s", socket, engine)
		logs.Error(err)
		return err
	}
	logs.Info("it is detected socket file: %s", socket)
	logs.Info("sending SIGHUP signal to %s for reloading config", engine)
	retriable := func() error {
		conn, err := net.Dial("unix", socket)
		if err != nil {
			logs.Error(err, "dial socket: %s failed", socket)
			return err
		}
		defer conn.Close()

		sconn, err := conn.(*net.UnixConn).SyscallConn()
		if err != nil {
			logs.Error(err, "unable to get syscall connection")
			return err
		}

		err1 := sconn.Control(func(fd uintptr) {
			err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_PASSCRED, 1)
		})
		if err1 != nil {
			logs.Error(err, "unable to issue call on socket fd")
			return err
		}
		if err != nil {
			logs.Error(err, "unable to SetsockoptInt on socket fd")
			return err
		}

		_, _, err = conn.(*net.UnixConn).WriteMsgUnix([]byte(consts.SocketMessageToGetPID), nil, nil)
		if err != nil {
			logs.Error(err, "unable to WriteMsgUnix on socket fd")
			return err
		}

		oob := make([]byte, 1024)
		_, oobn, _, _, err := conn.(*net.UnixConn).ReadMsgUnix(nil, oob)
		if err != nil {
			logs.Error(err, "unable to ReadMsgUnix on socket fd")
			return err
		}

		oob = oob[:oobn]
		scm, err := syscall.ParseSocketControlMessage(oob)
		if err != nil {
			logs.Error(err, "unable to ParseSocketControlMessage from message received on socket fd")
			return err
		}

		ucred, err := syscall.ParseUnixCredentials(&scm[0])
		if err != nil {
			logs.Error(err, "unable to ParseUnixCredentials from message received on socket fd")
			return err
		}

		err = syscall.Kill(int(ucred.Pid), syscall.SIGHUP)
		if err != nil {
			logs.Error(err, "sending SIGHUP signal to %s failed", engine)
			return err
		}
		return nil
	}

	if engine == consts.CONTAINERD {
		if _, err := os.Create(consts.RestartFlag); err != nil {
			logs.Error(err, "create restart flag file: %s failed", consts.RestartFlag)
			return err
		}
		logs.Info("create restart flag file: %s success", consts.RestartFlag)
	}
	var err error
	for i := 0; i < consts.MaxReloadAttempts; i++ {
		err = retriable()
		if err == nil {
			break
		}
		logs.Warn("start the retry times: %d, max retry times: %d", i+1, consts.MaxReloadAttempts)
		time.Sleep(consts.ReloadBackoff)
	}
	if err != nil {
		logs.Warn("max retries reached %d/%d, aborting", consts.MaxReloadAttempts, consts.MaxReloadAttempts)
		if err := os.Remove(consts.RestartFlag); err != nil {
			logs.Error(err)
			os.Exit(1)
		}
		return err
	}

	logs.Info("Successfully signaled %s", engine)
	return nil
}

func systemdRestart(engine, hostRootMount string) error {
	if engine == consts.CONTAINERD {
		if _, err := os.Create(consts.RestartFlag); err != nil {
			logs.Error(err, "create restart flag file: %s failed", consts.RestartFlag)
			return err
		}
		logs.Info("create restart flag file: %s success", consts.RestartFlag)
	}
	logs.Info("restarting %s using systemd", engine)
	command := fmt.Sprintf("chroot %s systemctl restart %s", hostRootMount, engine)
	if err := ExecCommand("sh", "-c", command); err != nil {
		logs.Error("restarting %s using systemd failed", engine)
		if err := os.Remove(consts.RestartFlag); err != nil {
			logs.Error(err)
			os.Exit(1)
		}
		return err
	}
	logs.Info("restarting %s using systemd success", engine)
	return nil
}

func HasValue(list []interface{}, value interface{}) bool {
	for _, eachValue := range list {
		if value == eachValue {
			return true
		}
	}
	return false
}

func ExecCommand(name string, args ...string) error {
	// catch standard output of error
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		logs.Error(err, "execute command: '%s %s' failed, detail: %s", name, strings.Join(args, " "), stderrBuf.String())
		return fmt.Errorf(stderrBuf.String())
	}
	// if stdoutBuf.String() != "" {
	// 	logs.Info("execute command: '%s %s' success, output: \n%s", name, strings.Join(args, " "), stdoutBuf.String())
	// } else {
	// 	logs.Info("execute command: '%s %s' success", name, strings.Join(args, " "))
	// }
	return nil
}

func SetNodeLabelWithClientSet(clientSet *kubernetes.Clientset, nodeName, label, value string) error {
	patchData := map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": map[string]string{
				label: value,
			},
		},
	}
	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		logs.Error(err, "json marshal failed, content: %v", patchData)
		return err
	}
	_, err = clientSet.CoreV1().Nodes().Patch(context.TODO(), nodeName, k8stypes.MergePatchType, patchBytes,
		metav1.PatchOptions{})
	if err != nil {
		logs.Error(err, "patch node: %s set label %s: %s failed", nodeName, label, value)
		return err
	}
	logs.Info("patch node: %s set label %s: %s success", nodeName, label, value)
	return nil
}
