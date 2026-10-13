// Copyright (c) 2024 Enflame. All Rights Reserved.

package args

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"gcu-operator/components/common/consts"
	opconsts "gcu-operator/pkg/consts"
	"gcu-operator/pkg/logs"
)

type Args struct {
	ConfigFile      string
	SocketFile      string
	CDIEnabled      bool
	CDIConfigPath   string
	Restart         bool
	RestartMode     string
	HostRootMount   string
	ContainerEngine string
	ClientSet       *kubernetes.Clientset
}

type NodeVerison struct {
	KubeletVersion          string
	ContainerRuntimeVersion string
}

var configFileMap = map[string]string{
	consts.DOCKER:     consts.DockerConfig,
	consts.CONTAINERD: consts.CtrConfig,
}

var socketMap = map[string]string{
	consts.DOCKER:     consts.DockerSockFile,
	consts.CONTAINERD: consts.CtrSockFile,
}

func (args *Args) ParseAndValidateBaseArgs(ctx *cli.Context) error {
	configFile := ctx.String("config")
	if configFile == "" {
		if configFile = configFileMap[args.ContainerEngine]; configFile == "" {
			err := fmt.Errorf("internal error: container engine: %s is not support", args.ContainerEngine)
			logs.Error(err)
			return err
		}
	}
	args.ConfigFile = configFile

	socketFile := ctx.String("socket")
	if socketFile == "" {
		if socketFile = socketMap[args.ContainerEngine]; socketFile == "" {
			err := fmt.Errorf("internal error: container engine: %s is not support", args.ContainerEngine)
			logs.Error(err)
			return err
		}
	}
	args.SocketFile = socketFile
	args.Restart = ctx.Bool("restart-container-engine")
	args.CDIEnabled = ctx.Bool("cdi-enabled")
	args.CDIConfigPath = ctx.String("cdi-config-path")
	args.RestartMode = ctx.String("restart-mode")
	args.HostRootMount = ctx.String("host-root")
	return nil
}

func (args *Args) GetVersion() (*NodeVerison, error) {
	nodeVersionMap := os.Getenv(opconsts.NODE_VERSION)
	if nodeVersionMap == "" {
		err := fmt.Errorf("env: %s can not be empty", opconsts.NODE_VERSION)
		logs.Error(err)
		return nil, err
	}
	nodeVersion := map[string]*NodeVerison{}
	if err := json.Unmarshal([]byte(nodeVersionMap), &nodeVersion); err != nil {
		logs.Error(err, "change string: %s to struct failed", nodeVersionMap)
		return nil, err
	}
	nodeName := os.Getenv(consts.NODE_NAME)
	if nodeName == "" {
		err := fmt.Errorf("env: %s can not be empty", consts.NODE_NAME)
		logs.Error(err)
		return nil, err
	}
	if nodeVer, ok := nodeVersion[nodeName]; ok {
		return nodeVer, nil
	}
	logs.Warn("node version not found for node: %s, try to get it from cluster node", nodeName)
	return args.getNodeVersionFromCluster(nodeName)
}

func (args *Args) getNodeVersionFromCluster(nodeName string) (*NodeVerison, error) {
	node, err := args.ClientSet.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		logs.Error(err, "get node: %s from cluster failed", nodeName)
		return nil, err
	}
	nodeVerison := &NodeVerison{
		KubeletVersion:          node.Status.NodeInfo.KubeletVersion,
		ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
	}
	logs.Info("get node version from cluster node: %s success, content: %v", nodeName, *nodeVerison)
	return nodeVerison, nil
}
