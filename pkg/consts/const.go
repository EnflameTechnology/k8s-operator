// Copyright (c) 2022 Enflame. All Rights Reserved.

package consts

const (
	// gcu resource info
	GROUP         = "topscloud.enflame.com"
	VERSION       = "v1"
	OPERATOR_NAME = "gcu-operator"
	GCURESOURCE   = "gcuresource"
	KIND          = "GcuResource"

	// for log
	LOGPATH  = "/var/log/enflame/gcu-operator/gcu-operator.log"
	LOGINFO  = "INFO"
	LOGERROR = "ERROR"
	LOGWARN  = "WARNING"

	// for gcu resouce action
	DEPLOY   = "deploy"
	DELETE   = "delete"
	UPGRADE  = "upgrade"
	ROLLBACK = "rollback"

	// for phase
	DEPLOYING   = "deploying"
	DELETING    = "deleting"
	UPGRADING   = "upgrading"
	ROLLBACKING = "rollbacking"
	RUNNING     = "Running"
	FAILED      = "Failed"

	// Finalizers
	GCU_RESOURCE_INFO = GROUP + "/" + VERSION

	// for action retry
	ACTION_RETRY = 0
	ACTION_FAIL  = 1
	ACTION_STOP  = 2 // cluster has no gcu node, need not create sub resource

	// for gcu label
	// only nodes containing follow labels can deploy GCU components
	EnflameLabelPrefix = "enflame.com/"
	NfdLabelPrefix     = "feature.node.kubernetes.io/"

	UnifiedDeviceLabelKey   = EnflameLabelPrefix + "gcu.present"
	UnifiedDeviceLabelValue = "true"

	K8sPluginResourceName  = EnflameLabelPrefix + "gcu"
	K8sPluginVResourceName = EnflameLabelPrefix + "vgcu"
	GcuSharePluginLabel    = EnflameLabelPrefix + "gcushare"

	GCUShareConfigFile = "gcushare-scheduler-plugin-config.yaml"

	GcuMemoryName = EnflameLabelPrefix + "gcu-memory"
	GcuCountName  = EnflameLabelPrefix + "gcu-count"

	// gcu resource list
	NFD               = "node-feature-discovery"
	Driver            = "enflame-driver"
	Toolkit           = "container-toolkit"
	Plugin            = "k8s-device-plugin"
	GcuExporter       = "gcu-exporter"
	NodeExporter      = "node-exporter"
	GFD               = "gcu-feature-discovery"
	GCUSharePlugin    = "gcushare-device-plugin"
	GCUShareScheduler = "gcushare-scheduler-plugin"
	K8sDriverManager  = "k8s-driver-manager"

	// for decorators
	RetryTimes = 5
	Interval   = 3

	// labels
	LABEL_INSTANCE    = "app.kubernetes.io/instance"
	LABEL_NAME        = "app.kubernetes.io/name"
	LABEL_COMPONENT   = "app.kubernetes.io/component"
	LABEL_DEPLOYED_BY = "app.kubernetes.io/deployed-by"

	DriverInstalledLabel = "gcu-operator/driver.installed"

	// error tag
	NoKindErr = "no kind is registered for the type"

	// other
	RefreshTimes     = 5
	ResourceCount    = "6"
	ImagePullPolicy  = "IfNotPresent"
	DefaultNamespace = "kube-system"

	// nfd containers
	NfdMaster = "nfd-master"
	NfdWorker = "nfd-worker"

	// envs
	NODE_VERSION = "NODE_VERSION"

	// versions
	ContainerRuntimeVersion = "containerRuntimeVersion"
	KubeletVersion          = "kubeletVersion"

	// component version
	VersionFile = "/tmp/g_version.txt"

	CONTAINER_TOOLKIT_VERSION         = "CONTAINER_TOOLKIT_VERSION"
	ENFLAME_CONTAINER_TOOLKIT_VERSION = "ENFLAME_CONTAINER_TOOLKIT_VERSION"
	K8S_DEVICE_PLUGIN_VERSION         = "K8S_DEVICE_PLUGIN_VERSION"
	ENFLAME_DEVICE_PLUGIN_VERSION     = "ENFLAME_DEVICE_PLUGIN_VERSION"
	GCU_EXPORTER_VERSION              = "GCU_EXPORTER_VERSION"
	NODE_EXPORTER_VERSION             = "NODE_EXPORTER_VERSION"
	GCU_MONITOR_VERSION               = "GCU_MONITOR_VERSION"
	GCUSHARE_VERSION                  = "GCUSHARE_VERSION"
	NODE_FEATURE_DISCOVERY_VERSION    = "NODE_FEATURE_DISCOVERY_VERSION"
	GCU_FEATURE_DISCOVERY_VERSION     = "GCU_FEATURE_DISCOVERY_VERSION"
	K8S_DRIVER_MANAGER_VERSION        = "K8S_DRIVER_MANAGER_VERSION"
	GCU_OPERATOR_DRIVER_VERSION       = "GCU_OPERATOR_DRIVER_VERSION"
)

// cluster node label
var EnflameVendorIdLabelsMap = map[string]string{
	// enflame gcu vendor id
	"feature.node.kubernetes.io/pci-1200_1e36.present": "true",
}
