// Copyright (c) 2024 Enflame. All Rights Reserved.

package consts

import "time"

const (
	// for healthCheck
	HealthCheckFile = "/tmp/toolkit.running"

	// for containerd
	CtrConfig        = "/etc/containerd/config.toml"
	CtrConfigEnflame = "/work/config.enflame.toml"
	CtrSockFile      = "/run/containerd/containerd.sock"

	// for docker
	DockerConfig   = "/etc/docker/daemon.json"
	DockerSockFile = "/var/run/docker.sock"

	// common
	BackupFileSuffix = ".ori.op.bak"

	// Built-in envs
	NODE_NAME = "NODE_NAME"

	// support container engine
	CONTAINERD = "containerd"
	DOCKER     = "docker"

	// support runtime
	ENFLAME_RUNTIME = "enflame"
	RUNC            = "runc"

	// sopport restart mode
	SIGNAL  = "signal"
	SYSTEMD = "systemd"

	// for restart container engine
	MaxReloadAttempts                   = 5
	SocketMessageToGetPID               = "GET /info HTTP/1.0\r\n\r\n"
	ReloadBackoff         time.Duration = 5 * time.Second

	// other
	KubeletConfigFile          = "/var/lib/kubelet/config.yaml"
	DefaultCDIConfigPath       = "/etc/cdi"
	DefaultCDIAnnotationPrefix = "cdi.k8s.io/"
	DefaultCDIKind             = "enflame.com/gcu"
	DefaultHostRootMount       = "/host-root"
	DefaultRestartMode         = "signal"
	EnflameRuntimeFile         = "/usr/bin/enflame-container-runtime"
	EnflameRuntimeConfig       = "/etc/enflame-container-runtime/config.toml"
	RestartFlag                = "/restart/restart.flag"
	ContainerToolkitReady      = "gcu-operator/containerToolkit.installed"
)
