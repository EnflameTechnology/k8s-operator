// Copyright (c) 2022 Enflame. All Rights Reserved.

package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "gcu-operator/apis/topscloud/v1"
	"gcu-operator/business-logic/base/interfaces"
	"gcu-operator/business-logic/gcu-resource/base"
	gcuPro "gcu-operator/business-logic/gcu-resource/base/progress"
	gcuStatus "gcu-operator/business-logic/gcu-resource/base/status"
	argsConsts "gcu-operator/components/common/consts"
	"gcu-operator/pkg/common"
	"gcu-operator/pkg/consts"
	gcuErr "gcu-operator/pkg/error"
	"gcu-operator/pkg/logs"
	"gcu-operator/pkg/object"
)

type DeployGcuK8sPlugin struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeployGcuK8sPlugin{}

func NewDeployGcuK8sPlugin(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeployGcuK8sPlugin{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeployGcuK8sPlugin) GetProcedureName() string {
	return "DeployGcuK8sPlugin"
}

func (rer *DeployGcuK8sPlugin) GetProcedureStatus() []string {
	return []string{gcuStatus.DEPLOY_TOOLKIT_SUCCESS, gcuStatus.DEPLOY_DEVICE_PLUGIN_FAILED}
}

func (rer *DeployGcuK8sPlugin) GetProcedureOKStatus() string {
	return gcuStatus.DEPLOY_DEVICE_PLUGIN_SUCCESS
}

func (rer *DeployGcuK8sPlugin) GetProcedureFailStatus() string {
	return gcuStatus.DEPLOY_DEVICE_PLUGIN_FAILED
}

func (rer *DeployGcuK8sPlugin) GetProcedureOKProgress() int {
	return gcuPro.DEPLOY_DEVICE_PLUGIN_SUCCESS
}

func (rer *DeployGcuK8sPlugin) GetProcedureFailProgress() int {
	return gcuPro.DEPLOY_DEVICE_PLUGIN_FAILED
}

func (rer *DeployGcuK8sPlugin) GetProcedureAction() string {
	return consts.DEPLOY
}

func (rer *DeployGcuK8sPlugin) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeployGcuK8sPlugin) GetClient() client.Client {
	return rer.Client
}

func (rer *DeployGcuK8sPlugin) Execute() (err error) {

	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	var objectSets, deploy = []client.Object{}, true
	defer rer.updateStatus(status, &err, &objectSets, &deploy)

	//logic here
	if rer.GcuResource.Spec.DevicePlugin == nil || (rer.GcuResource.Spec.DevicePlugin.Deploy != nil &&
		!*rer.GcuResource.Spec.DevicePlugin.Deploy) {
		logs.Warn("resource %s need not deploy, will skip this procedure", consts.Plugin)
		deploy = false
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:         consts.Plugin,
		Namespace:    base.GetNamespace(consts.Plugin, rer.GcuResource.Spec.DevicePlugin.Namespace),
		Status:       gcuStatus.DEPLOYING,
		OwnResources: []v1.ResourceInfo{},
	}
	if err = rer.getObjectSets(&objectSets); err != nil {
		return err
	}
	if err = rer.deployResources(objectSets, status); err != nil {
		return err
	}
	return nil
}

func (rer *DeployGcuK8sPlugin) deployResources(objectSets []client.Object, status *base.ProcessStatus) error {
	for _, resObj := range objectSets {
		gvk := resObj.GetObjectKind().GroupVersionKind()
		kind, name, namespace := gvk.Kind, resObj.GetName(), resObj.GetNamespace()
		if kind == object.DaemonSet {
			if err := rer.checkK8sDevicePluginCanDeploy(resObj.(*appsv1.DaemonSet)); err != nil {
				return err
			}
		}
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster start, content: %s", kind, name, namespace,
			consts.Plugin, common.ChangeInterfaceToString(resObj))
		if err := status.CreateResourceToCluster(resObj); err != nil {
			return err
		}
		resObj.GetObjectKind().SetGroupVersionKind(gvk)
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster success, gr name:%s, uuid:%s",
			kind, name, namespace, consts.Plugin, rer.Name, rer.UUID)
	}
	logs.Info("deploy all resource of %s to cluster success, count: %d", consts.Plugin, len(objectSets))
	return nil
}

func (rer *DeployGcuK8sPlugin) checkK8sDevicePluginCanDeploy(resObj *appsv1.DaemonSet) error {
	nodeSelector := resObj.Spec.Template.Spec.NodeSelector
	nodeList := &corev1.NodeList{}
	listOption := []client.ListOption{client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(nodeSelector)}}
	if err := rer.Client.List(context.TODO(), nodeList, listOption...); err != nil {
		logs.Error(err, "get cluster node list by nodeSelector: %v failed", nodeSelector)
		return gcuErr.NewError(consts.ACTION_RETRY, err.Error())
	}
	// It is difficult to obtain the config file of the specified node, only try to clear the default resource name
	for _, node := range nodeList.Items {
		value, ok := node.Status.Allocatable[consts.GcuMemoryName]
		if ok && value.String() != "0" {
			msg := "check gcushare device plugin already deployed, so can not deploy k8s device plugin"
			logs.Error(msg)
			return gcuErr.NewError(consts.ACTION_FAIL, msg)
		}
	}
	return nil
}

func (rer *DeployGcuK8sPlugin) getObjectSets(objectSets *[]client.Object) error {
	filesDir := ""
	version := rer.GcuResource.Spec.DevicePlugin.Version
	args := rer.GcuResource.Spec.DevicePlugin.Args
	if strings.HasPrefix(version, "1.") {
		logs.Info("%s 1.x will be deployed", consts.Plugin)
		filesDir = base.GcuK8sPluginV1Path
		if common.HasValue("--pcie-switch-affinity", args) {
			logs.Info("deploy %s with pcie switch affinity", consts.Plugin)
			filesDir = base.GcuK8sPluginV1PathForPcieSwitch
		}
	} else if strings.HasPrefix(version, "2.") {
		logs.Info("%s 2.x will be deployed", consts.Plugin)
		filesDir = base.GcuK8sPluginV2Path
	} else {
		msg := fmt.Sprintf("%s version: %s is invalid", consts.Plugin, version)
		logs.Error(msg)
		return gcuErr.NewError(consts.ACTION_FAIL, msg)
	}
	files, err := os.ReadDir(filesDir)
	if err != nil {
		logs.Error(err, "read directory: %s failed", filesDir)
		return gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	for _, file := range files {
		filePath := fmt.Sprintf("%s/%s", filesDir, file.Name())
		resObj, err := rer.readAndParseBlueprint(filePath)
		if err != nil {
			return err
		}
		*objectSets = append(*objectSets, resObj)
	}
	logs.Info("read and parse all cluster resource of %s finish", consts.Plugin)
	return nil
}

func (rer *DeployGcuK8sPlugin) readAndParseBlueprint(filePath string) (client.Object, error) {
	content, err := base.ReadYamlToJson(filePath)
	if err != nil {
		logs.Error(err, "read file: %s content failed", filePath)
		return nil, err
	}
	resMeta, err := base.GetResourceTypeMeta(content)
	if err != nil {
		return nil, gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	resObj := object.Objects()[resMeta.Kind]
	if err := json.Unmarshal([]byte(content), resObj); err != nil {
		logs.Error(err, "json unmarshal resource: %s to client.Object failed", content)
		return nil, gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	switch realObj := resObj.(type) {
	case *appsv1.DaemonSet:
		if err := rer.instanceDevicePlugin(realObj); err != nil {
			return nil, err
		}
		if base.NeedInstallContainerToolkit(rer.GcuResource) {
			base.AddDsNodeSelector(realObj, map[string]string{argsConsts.ContainerToolkitReady: "true"})
		} else if base.NeedInstallDriver(rer.GcuResource) {
			base.AddDsNodeSelector(realObj, map[string]string{consts.DriverInstalledLabel: "true"})
		}
		base.AddCommandArgs(realObj, rer.GcuResource.Spec.DevicePlugin.Command, rer.GcuResource.Spec.DevicePlugin.Args)
	}
	base.AddLabels(resObj, rer.Name, consts.Plugin)
	base.AddOwnereference(resObj, rer.GcuResource)
	logs.Info("read %s(name: %s, namespace: %s) of %s success", resObj.GetObjectKind().GroupVersionKind().Kind,
		resObj.GetName(), resObj.GetNamespace(), consts.Plugin)
	return resObj, nil
}

func (rer *DeployGcuK8sPlugin) instanceDevicePlugin(resObj *appsv1.DaemonSet) error {
	if rer.GcuResource.Spec.DevicePlugin.Namespace != "" {
		resObj.SetNamespace(rer.GcuResource.Spec.DevicePlugin.Namespace)
	}
	DevicePluginSpec := rer.GcuResource.Spec.DevicePlugin
	baseFields := v1.BaseFields{
		Name:            DevicePluginSpec.Name,
		Namespace:       DevicePluginSpec.Namespace,
		Image:           DevicePluginSpec.Image,
		Version:         DevicePluginSpec.Version,
		Repository:      DevicePluginSpec.Repository,
		ImagePullPolicy: DevicePluginSpec.ImagePullPolicy,
		NodeSelector:    DevicePluginSpec.NodeSelector,
		Env:             DevicePluginSpec.Env,
	}
	if err := base.InstaceResObject(resObj, &baseFields); err != nil {
		logs.Error(err, "instance plugin object failed")
		return err
	}
	if err := rer.setInitContainer(resObj); err != nil {
		return err
	}
	switch rer.GcuResource.Spec.DevicePlugin.Variant {
	case "basic", "":
		logs.Info("deploy %s with basic variant", consts.Plugin)
		rer.handleBasic(resObj)
	case "pcieSwitch":
		logs.Info("deploy %s with pcie switch variant", consts.Plugin)
		rer.handlePcieSwitch(resObj)
	case "cpuManager":
		logs.Info("deploy %s with cpu manager variant", consts.Plugin)
		rer.handleCpuManager(resObj)
	case "cdi":
		logs.Info("deploy %s with cdi variant", consts.Plugin)
		rer.handleCdi(resObj)
	default:
		msg := fmt.Sprintf("variant: %s is invalid, only support basic(default), pcieSwitch, cpuManager, cdi",
			rer.GcuResource.Spec.DevicePlugin.Variant)
		logs.Error(msg)
		return gcuErr.NewError(consts.ACTION_FAIL, msg)
	}
	logs.Info("instance ds: %s of %s success", resObj.Name, consts.Plugin)
	return nil
}

func (rer *DeployGcuK8sPlugin) handleBasic(resObj *appsv1.DaemonSet) {
	resObj.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
		RunAsUser: &[]int64{0}[0],
	}
	resObj.Spec.Template.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{
		AllowPrivilegeEscalation: &[]bool{false}[0],
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
			Add:  []corev1.Capability{"SYS_CHROOT"},
		},
	}
}

func (rer *DeployGcuK8sPlugin) handlePcieSwitch(resObj *appsv1.DaemonSet) {
	rer.handleBasic(resObj)
	for i, env := range resObj.Spec.Template.Spec.Containers[0].Env {
		if env.Name == "CUSTOM_REGISTER_RESOURCE" {
			resObj.Spec.Template.Spec.Containers[0].Env[i].Value = "enflame.com/gcu,enflame-tech.com/gcu"
			break
		}
	}
	resObj.Spec.Template.Spec.Containers[0].Env = append(resObj.Spec.Template.Spec.Containers[0].Env, []corev1.EnvVar{
		{
			Name:  "PCIE_SWITCH_AFFINITY",
			Value: "true",
		},
		{
			Name:  "KUBECONFIG",
			Value: "/etc/kubernetes/kubelet.conf",
		},
	}...)
}

func (rer *DeployGcuK8sPlugin) handleCpuManager(resObj *appsv1.DaemonSet) {
	resObj.Spec.Template.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{
		Privileged: &[]bool{true}[0],
	}
	resObj.Spec.Template.Spec.Containers[0].Env = append(resObj.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  "PASS_DEVICE_SPECS",
		Value: "true",
	})
}

func (rer *DeployGcuK8sPlugin) handleCdi(resObj *appsv1.DaemonSet) {
	resObj.Spec.Template.Spec.HostPID = true
	resObj.Spec.Template.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{
		Privileged: &[]bool{true}[0],
	}
	resObj.Spec.Template.Spec.Containers[0].Env = append(resObj.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  "DEVICE_LIST_STRATEGY",
		Value: "cdi-annotations,envvar",
	})
}

func (rer *DeployGcuK8sPlugin) updateStatus(status *base.ProcessStatus, err *error, objectSets *[]client.Object,
	deploy *bool) {
	// create plugin failed, need not modify readyResource
	if *err != nil || !*deploy {
		return
	}
	// check plugin ds run status
	namespacedNames := []*k8sTypes.NamespacedName{}
	for _, resObj := range *objectSets {
		if resObj.GetObjectKind().GroupVersionKind().Kind == object.DaemonSet {
			namespacedNames = append(namespacedNames, &k8sTypes.NamespacedName{
				Namespace: resObj.GetNamespace(),
				Name:      resObj.GetName(),
			})
		}
	}
	isRunning, errCheck := base.DecoratorForCheckController(base.CheckDeamonsetRunning, rer.Client, namespacedNames)
	// check daemonset status failed, return and requeue reconciler
	if errCheck != nil {
		*err = errCheck
		return
	}
	// check daemonset running, plugin deploy ok
	if isRunning {
		status.IsReady = true
		logs.Info("check all resource of %s run status running", consts.Plugin)
	} else {
		msg := fmt.Sprintf("check ds of %s of gr name:%s, uuid:%s is not running", consts.Plugin, rer.Name, rer.UUID)
		logs.Warn(msg)
		*err = gcuErr.NewError(consts.ACTION_RETRY, msg)
	}
}
func (rer *DeployGcuK8sPlugin) setInitContainer(resObj *appsv1.DaemonSet) error {
	envs, err := rer.getEnvs()
	if err != nil {
		return err
	}
	envs = append(envs, corev1.EnvVar{Name: "CDI_ENABLED", Value: getCDIEnabled(resObj.Spec.Template.Spec.Containers[0].Env)})
	resObj.Spec.Template.Spec.InitContainers = append(resObj.Spec.Template.Spec.InitContainers, corev1.Container{
		Name:            "k8s-device-plugin-cdi-manager",
		Image:           resObj.Spec.Template.Spec.Containers[0].Image,
		Command:         []string{"cdi-manager"},
		Env:             envs,
		VolumeMounts:    resObj.Spec.Template.Spec.Containers[0].VolumeMounts,
		ImagePullPolicy: corev1.PullPolicy(rer.GcuResource.Spec.DevicePlugin.ImagePullPolicy),
		SecurityContext: resObj.Spec.Template.Spec.Containers[0].SecurityContext,
	})
	addVolumes(resObj)
	return nil
}

func (rer *DeployGcuK8sPlugin) getEnvs() ([]corev1.EnvVar, error) {
	envs := rer.GcuResource.Spec.DevicePlugin.Env
	nodeInfo := rer.GcuResource.Status.NodesInfo
	content, err := json.Marshal(nodeInfo)
	if err != nil {
		logs.Error(err, "json marshal failed, content: %v", nodeInfo)
		return nil, gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	envs = append(envs, []corev1.EnvVar{
		{Name: consts.NODE_VERSION, Value: string(content)},
		{Name: "LOGPATH", Value: "/var/log/enflame/gcu-operator/k8s-device-plugin-cdi-manager.log"},
		{Name: "NODE_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}},
	}...)
	return envs, nil
}

func getCDIEnabled(envs []corev1.EnvVar) string {
	for _, env := range envs {
		if env.Name == "DEVICE_LIST_STRATEGY" && common.HasValue("cdi-annotations", strings.Split(env.Value, ",")) {
			return "true"
		}
	}
	return "false"
}

func addVolumes(resObj *appsv1.DaemonSet) {
	volumeType := corev1.HostPathDirectoryOrCreate
	resObj.Spec.Template.Spec.Volumes = append(resObj.Spec.Template.Spec.Volumes, []corev1.Volume{
		{
			Name: "gcu-operator-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/log/enflame/gcu-operator",
					Type: &volumeType,
				},
			},
		},
		{
			Name: "containerd-config",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/containerd",
				},
			},
		},
		{
			Name: "docker-config",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/docker",
					Type: &volumeType,
				},
			},
		},
		{
			Name: "var-run",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/run/",
					Type: &volumeType,
				},
			},
		},
		{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}...)
	resObj.Spec.Template.Spec.InitContainers[0].VolumeMounts = append(resObj.Spec.Template.Spec.InitContainers[0].VolumeMounts, []corev1.VolumeMount{
		{
			Name:      "gcu-operator-dir",
			MountPath: "/var/log/enflame/gcu-operator",
		},
		{
			Name:      "containerd-config",
			MountPath: "/etc/containerd",
		},
		{
			Name:      "docker-config",
			MountPath: "/etc/docker",
		},
		{
			Name:      "var-run",
			MountPath: "/var/run/",
		},
		{
			Name:      "data",
			MountPath: "/restart",
		},
	}...)
}
