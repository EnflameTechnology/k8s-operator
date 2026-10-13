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
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"gcu-operator/components/common/utils"
	"gcu-operator/pkg/consts"
	"gcu-operator/pkg/kube"
	"gcu-operator/pkg/logs"
)

const (
	defaultWorkDir     string = "/work"
	KMDPkgSuffix       string = ".run"
	enflameDriver      string = "enflame-"
	enflamePeerMem     string = "enflame_peer_mem-"
	checkFile          string = "/tmp/driverCheckFlag.log"
	checkFileLine1Pass string = "PASS"
	checkFileLine2GCU  string = "GCU"
	checkFileLine2VGCU string = "VGCU"

	usrBin string = "/usr/bin"

	//more info at: http://ids.enflame.cn/
	PcieVendorID string = "1e36"
)

func main() {
	vgcuEnabled, err := enableVGCU()
	if err != nil {
		os.Exit(1)
	}

	node, clientSet, err := getNodeInfo()
	if err != nil {
		logs.Error(err, "get node info failed")
		os.Exit(1)
	}

	needInstall, err := checkNeedInstallDriver(node)
	if err != nil {
		logs.Error(err, "check need install driver failed")
		os.Exit(1)
	}
	if !needInstall {
		if err := writeDeviceType(checkFileLine1Pass, vgcuEnabled); err != nil {
			os.Exit(1)
		}
		logs.Info("waiting for the termination signal...")
		pendingSignal()
		return
	}

	if err := checkDevices(); err != nil {
		logs.Error(err, "can not found enflame gcu devices")
		os.Exit(1)
	}

	err = installEnflameKMD()
	if err != nil {
		os.Exit(1)
	}

	if err := foundSmiTool(); err != nil {
		os.Exit(1)
	}

	if err := utils.SetNodeLabelWithClientSet(clientSet, node.Name, consts.DriverInstalledLabel, "true"); err != nil {
		os.Exit(1)
	}

	if err := writeDeviceType(checkFileLine1Pass, vgcuEnabled); err != nil {
		os.Exit(1)
	}
	logs.Info("Install enflame driver success, waiting for the termination signal...")
	pendingSignal()
}

func pendingSignal() {
	termSignals := []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, termSignals...)
	logs.Info("Receive system signal: %v", <-sigChan)
}

func checkNeedInstallDriver(node *corev1.Node) (bool, error) {
	_, ok := node.Labels[consts.DriverInstalledLabel]
	if !ok {
		logs.Info("enflame driver is not installed on node %s, need to install", node.Name)
		return true, nil
	}
	logs.Warn("enflame driver is already installed on node %s", node.Name)
	if os.Getenv("ONE_SHOT") == "true" {
		logs.Warn("need not to install enflame driver again with one shot mode")
		return false, nil
	}
	logs.Info("need install enflame driver, this will overwrite the existing driver")
	return true, nil
}

func getNodeInfo() (*corev1.Node, *kubernetes.Clientset, error) {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		msg := "enflame driver need to set env NODE_NAME"
		logs.Error(msg)
		return nil, nil, errors.New(msg)
	}
	clientSet, err := kube.GetClientSet()
	if err != nil {
		logs.Error(err, "get kube client set failed")
		return nil, nil, err
	}
	node, err := clientSet.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		logs.Error(err, "get node %s failed", nodeName)
		return nil, nil, err
	}
	logs.Info("get node %s success", nodeName)
	return node, clientSet, nil
}

func foundSmiTool() error {
	var stdoutBuf, stderrBuf bytes.Buffer
	command := "whereis efsmi"
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		logs.Info("execute command: %s failed: %s", command, stderrBuf.String())
		return err
	}
	paths := strings.Split(strings.Trim(stdoutBuf.String(), "\n"), " ")
	if len(paths) < 2 {
		err := fmt.Errorf("efsmi not found")
		logs.Error(err)
		return err
	}
	for _, path := range paths[1:] {
		ok, err := resolveSymlink(path)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		ok, err = resolveExecutableFile(path)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	err := fmt.Errorf("executable file of efsmi is not found in %v", paths)
	logs.Error(err)
	return err
}

// IsExecutable checks if the given path is an executable file.
func resolveExecutableFile(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		logs.Error(err, "access file: %s failed", path)
		return false, err
	}

	if !fileInfo.Mode().IsRegular() {
		logs.Warn("file: %s is not a regular file", path)
		return false, nil
	}

	if fileInfo.Mode()&0111 == 0 {
		logs.Warn("file: %s is not a executable file", path)
		return false, nil
	}
	logs.Info("file: %s is a executable file", path)
	if !fileIsExist(usrBin + "/" + fileInfo.Name()) {
		cmdline := fmt.Sprintf("cp %s %s", path, usrBin)
		if err := execCommand("sh", "-c", cmdline); err != nil {
			logs.Error(err, "execute command: '%s' failed", cmdline)
			return false, err
		}
	}
	logs.Info("install efsmi to %s success", usrBin)
	return true, nil
}

func fileIsExist(file string) bool {
	_, err := os.Stat(file)
	return err == nil || os.IsExist(err)
}

func resolveSymlink(path string) (bool, error) {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		logs.Error(err, "access file: %s failed", path)
		return false, err
	}

	if fileInfo.Mode()&os.ModeSymlink == 0 {
		return false, nil
	}
	logs.Info("file: %s is a symlink file", path)
	link, err := os.Readlink(path)
	if err != nil {
		logs.Error(err, "read link of file: %s failed", path)
		return false, err
	}

	if filepath.IsAbs(link) {
		return resolveExecutableFile(link)
	}
	absPath, err := filepath.Abs(link)
	if err != nil {
		logs.Error(err, "get absolute path of link: %s failed", link)
		return false, err
	}
	return resolveExecutableFile(absPath)
}

// line1:  PASS/FAILED   #whether has enflame device
// line2:  GCU/VGCU    #device type
func writeDeviceType(line1 string, vgcuEnabled bool) error {
	line2 := checkFileLine2GCU
	if vgcuEnabled {
		line2 = checkFileLine2VGCU
	}
	logs.Info("writeDeviceType to driverCheckFlag.log, line1: %v, line2: %v", line1, line2)
	f, err := os.Create(checkFile)
	if err != nil {
		logs.Error(err, "create file: %s failed", checkFile)
		return err
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("%v\n%v\n", line1, line2))
	if err != nil {
		logs.Error(err, "write to file: %s failed", checkFile)
		return err
	}
	logs.Info("write driver check flag success, path: %s", checkFile)
	return nil
}

func checkDevices() error {
	//clean old file
	err := os.RemoveAll(checkFile)
	if err != nil {
		logs.Error(err, "delete flag file %s failed", checkFile)
		return err
	}

	//check pci device, then write record to the file.
	logs.Info("start check pci device")
	cmdline := fmt.Sprintf("lspci -vv | grep -iE 'Enflame|Device %s:'", PcieVendorID)
	if err := execCommand("sh", "-c", cmdline); err != nil {
		logs.Error(err, "check pci device failed, command: %s", cmdline)
		return err
	}
	logs.Info("check pci device success")
	return nil
}

func execCommand(name string, args ...string) error {
	// catch standard output of error
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		return errors.New(stderrBuf.String())
	}
	if stdoutBuf.String() != "" {
		logs.Info("execute command: '%s %s' success, output: \n%s", name, strings.Join(args, " "), stdoutBuf.String())
	} else {
		logs.Info("execute command: '%s %s' success", name, strings.Join(args, " "))
	}
	return nil
}

func installEnflameKMD() error {
	logs.Info("Begin to install KMD")
	if err := installEnflameDriver(); err != nil {
		return err
	}

	if err := installEnflamePeerMem(); err != nil {
		return err
	}
	return nil
}

func installEnflameDriver() error {
	driverFileName := getPackageName(defaultWorkDir, enflameDriver, KMDPkgSuffix)
	if driverFileName == "" {
		msg := fmt.Sprintf("enflame driver file not found in folder: %s", defaultWorkDir)
		logs.Error(msg)
		return errors.New(msg)
	}
	driverFilePath := fmt.Sprintf("%s/%s", defaultWorkDir, driverFileName)
	if err := cleanupKMD(driverFilePath); err != nil {
		return err
	}

	command := fmt.Sprintf("%v --no-dkms", driverFilePath)
	if os.Getenv("NO_S60G_CHECK") == "true" {
		command = command + " --no-s60g-check"
	}
	if err := execCommand("sh", "-c", command); err != nil {
		logs.Error(err, "execute command: '%s' failed", command)
		return err
	}
	logs.Info("install enflame driver success")

	time.Sleep(1 * time.Second)
	command = "modprobe enflame -C /etc/modprobe.d/gcu_test.conf"
	if err := execCommand("sh", "-c", command); err != nil {
		logs.Error(err, "execute command: '%s' failed", command)
		return err
	}
	logs.Info("modprobe enflame success")

	time.Sleep(1 * time.Second)
	command = "lsmod |grep ^enflame"
	if err := execCommand("sh", "-c", command); err != nil {
		logs.Error(err, "execute command: '%s' failed", command)
		return err
	}
	return nil
}

func installEnflamePeerMem() error {
	if os.Getenv("PEER_MEM_ENABLED") != "true" {
		logs.Warn("you can set env 'PEER_MEM_ENABLED=true' to install enflame_peer_mem")
		return nil
	}
	peerMemFileName := getPackageName(defaultWorkDir, enflamePeerMem, KMDPkgSuffix)
	if peerMemFileName == "" {
		msg := fmt.Sprintf("enflame peer mem file not found in folder: %s", defaultWorkDir)
		logs.Error(msg)
		return errors.New(msg)
	}
	peerMemFilePath := fmt.Sprintf("%s/%s", defaultWorkDir, peerMemFileName)
	if err := execCommand("sh", "-c", peerMemFilePath); err != nil {
		logs.Error(err, "execute command: '%s' failed", peerMemFilePath)
		return err
	}
	logs.Info("install enflame peer mem success")
	return nil
}

func rmmod(moduleName string) error {
	cmdline := fmt.Sprintf("lsmod | grep -q '^%s'", moduleName)
	if err := execCommand("sh", "-c", cmdline); err != nil {
		logs.Warn("module %s is not loaded, no need to remove", moduleName)
		return nil
	}
	cmdline = fmt.Sprintf("rmmod %s", moduleName)
	if err := execCommand("sh", "-c", cmdline); err != nil {
		logs.Error(err, "execute command: '%s' failed", cmdline)
		return err
	}
	logs.Info("rmmod '%s' success", moduleName)
	return nil
}

func cleanupKMD(runFile string) error {
	if err := rmmod("enflame_peer_mem"); err != nil {
		return err
	}
	if err := rmmod("enflame"); err != nil {
		return err
	}
	cmdline := fmt.Sprintf("%v --uninstall", runFile)
	if err := execCommand("sh", "-c", cmdline); err != nil {
		logs.Error(err, "execute command: '%s' failed", cmdline)
		return err
	}
	return nil
}

func enableVGCU() (vgcuEnabled bool, err error) {
	cmdline := "echo 'options enflame enable_usr_parameter=\"enable_vgcu=0\"' > /etc/modprobe.d/gcu_test.conf; cat /etc/modprobe.d/gcu_test.conf"
	if os.Getenv("VGCU_ENABLED") == "true" {
		vgcuEnabled = true
		logs.Info("the driver will be installed in VGCU mode")
		cmdline = "echo 'options enflame enable_usr_parameter=\"enable_vgcu=1\"' > /etc/modprobe.d/gcu_test.conf; cat /etc/modprobe.d/gcu_test.conf"
	}
	logs.Info("try to write options of enflame")
	if err := execCommand("sh", "-c", cmdline); err != nil {
		logs.Error(err, "execute command: '%s' failed", cmdline)
		return vgcuEnabled, err
	}
	return vgcuEnabled, nil
}

func getPackageName(workDir string, prefix string, suffix string) string {
	files, err := os.ReadDir(workDir)
	if err != nil {
		logs.Error(err, "read folder: %s failed", workDir)
		return ""
	}
	for _, file := range files {
		fileName := file.Name()
		if !file.IsDir() && strings.HasPrefix(fileName, prefix) && strings.HasSuffix(fileName, suffix) {
			return fileName
		}
	}
	logs.Warn("no file start with '%s' and end with '%s' is found in folder: %s", prefix, suffix, workDir)
	return ""
}
