// Copyright (c) 2022 Enflame. All Rights Reserved.

package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

type DeployGcuSharePlugin struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeployGcuSharePlugin{}

func NewDeployGcuSharePlugin(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeployGcuSharePlugin{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeployGcuSharePlugin) GetProcedureName() string {
	return "DeployGcuSharePlugin"
}

func (rer *DeployGcuSharePlugin) GetProcedureStatus() []string {
	return []string{gcuStatus.DEPLOY_GFD_SUCCESS, gcuStatus.DEPLOY_GCUSHARE_PLUGIN_FAILED}
}

func (rer *DeployGcuSharePlugin) GetProcedureOKStatus() string {
	return gcuStatus.DEPLOY_GCUSHARE_PLUGIN_SUCCESS
}

func (rer *DeployGcuSharePlugin) GetProcedureFailStatus() string {
	return gcuStatus.DEPLOY_GCUSHARE_PLUGIN_FAILED
}

func (rer *DeployGcuSharePlugin) GetProcedureOKProgress() int {
	return gcuPro.DEPLOY_GCUSHARE_PLUGIN_SUCCESS
}

func (rer *DeployGcuSharePlugin) GetProcedureFailProgress() int {
	return gcuPro.DEPLOY_GCUSHARE_PLUGIN_FAILED
}

func (rer *DeployGcuSharePlugin) GetProcedureAction() string {
	return consts.DEPLOY
}

func (rer *DeployGcuSharePlugin) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeployGcuSharePlugin) GetClient() client.Client {
	return rer.Client
}

func (rer *DeployGcuSharePlugin) Execute() (err error) {

	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	var objectSets, deploy = []client.Object{}, true
	defer rer.updateStatus(status, &err, &objectSets, &deploy)

	//logic here
	if rer.GcuResource.Spec.GCUShare == nil || (rer.GcuResource.Spec.GCUShare.Deploy != nil && !*rer.GcuResource.Spec.GCUShare.Deploy) {
		logs.Warn("resource %s need not deploy, will skip this procedure", consts.GCUSharePlugin)
		deploy = false
		return nil
	}
	gcushareNodesList, err := rer.getClusterGcushareNodesList()
	if err != nil {
		return err
	}
	if len(gcushareNodesList) > 0 {
		nodeNames := []string{}
		for _, node := range gcushareNodesList {
			nodeNames = append(nodeNames, node.Name)
		}
		logs.Info("shared GCU node list: %v", nodeNames)
	} else {
		logs.Warn("shared GCU node list is empty")
	}
	if err := rer.checkGcusharePluginCanDeploy(gcushareNodesList); err != nil {
		return err
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:         consts.GCUSharePlugin,
		Namespace:    base.GetNamespace(consts.GCUSharePlugin, rer.GcuResource.Spec.GCUShare.DevicePlugin.Namespace),
		Status:       gcuStatus.DEPLOYING,
		OwnResources: []v1.ResourceInfo{},
	}
	if err = rer.getObjectSets(&objectSets); err != nil {
		return err
	}
	if err = rer.labelClusterGcuShareNodes(gcushareNodesList); err != nil {
		return err
	}
	if err = rer.deployResources(objectSets, status); err != nil {
		return err
	}
	return nil
}

func (rer *DeployGcuSharePlugin) getClusterGcushareNodesList() ([]corev1.Node, error) {
	gcushareNodes := rer.GcuResource.Spec.GCUShare.DevicePlugin.GCUshareNodes
	if len(gcushareNodes) > 0 {
		logs.Info("The input nodes that wants to share the GCU: %v", gcushareNodes)
		gcushareNodesList := []corev1.Node{}
		for _, nodeName := range gcushareNodes {
			node := &corev1.Node{}
			if err := rer.Client.Get(context.TODO(), k8sTypes.NamespacedName{Name: nodeName}, node); err != nil {
				logs.Error(err, "get cluster node: %s detail failed", nodeName)
				return nil, err
			}
			if _, ok := node.GetLabels()[consts.UnifiedDeviceLabelKey]; !ok {
				logs.Warn("the node %s has no GCU device, can not shared GCU, will ignore it", node.Name)
				continue
			}
			gcushareNodesList = append(gcushareNodesList, *node)
		}
		return gcushareNodesList, nil
	}

	logs.Warn("Expect all nodes to adopt shared GCU mode")
	nodeList := &corev1.NodeList{}
	if err := rer.Client.List(context.TODO(), nodeList); err != nil {
		logs.Error(err, "get cluster node list failed")
		return nil, gcuErr.NewError(consts.ACTION_RETRY, err.Error())
	}
	gcushareNodesList := []corev1.Node{}
	for _, node := range nodeList.Items {
		if _, ok := node.GetLabels()[consts.UnifiedDeviceLabelKey]; !ok {
			logs.Warn("the node %s has no GCU device, can not shared GCU, will ignore it", node.Name)
			continue
		}
		gcushareNodesList = append(gcushareNodesList, node)
	}
	return gcushareNodesList, nil
}

func (rer *DeployGcuSharePlugin) labelClusterGcuShareNodes(gcushareNodes []corev1.Node) error {
	for i := range gcushareNodes {
		node := &gcushareNodes[i]
		patch := client.MergeFrom(node.DeepCopy())
		node.Labels[consts.GcuSharePluginLabel] = "true"
		if err := rer.Client.Patch(context.TODO(), node, patch); err != nil {
			logs.Error(err, "update node:%s add label %s failed", node.Name, consts.GcuSharePluginLabel)
			return gcuErr.NewError(consts.ACTION_RETRY, err.Error())
		}
		logs.Info("update node:%s add label %s success", node.Name, consts.GcuSharePluginLabel)
	}
	return nil
}

func (rer *DeployGcuSharePlugin) checkGcusharePluginCanDeploy(gcushareNodes []corev1.Node) error {
	// It is difficult to obtain the config file of the specified node, only use the default resource name
	for _, node := range gcushareNodes {
		value1, ok1 := node.Status.Allocatable[consts.K8sPluginResourceName]
		value2, ok2 := node.Status.Allocatable[consts.K8sPluginVResourceName]
		if (ok1 && value1.String() != "0") || (ok2 && value2.String() != "0") {
			msg := "check k8s device plugin already deployed, so can not deploy gcushare device plugin"
			logs.Error(msg)
			return gcuErr.NewError(consts.ACTION_FAIL, msg)
		}
		// value, ok = node.Status.Capacity[consts.K8sPluginResourceName]
		// if ok && value.String() != "0" {
		// 	msg := "check k8s device plugin already deployed, so can not deploy gcushare device plugin"
		// 	logs.Error(fmt.Errorf(msg))
		// 	return gcuErr.NewError(consts.ACTION_FAIL, msg)
		// }
	}
	if rer.GcuResource.Spec.DevicePlugin == nil || (rer.GcuResource.Spec.DevicePlugin.Deploy != nil &&
		!*rer.GcuResource.Spec.DevicePlugin.Deploy) {
		return nil
	}
	msg := "check k8s device plugin already deployed, so can not deploy gcushare device plugin"
	logs.Error(msg)
	return gcuErr.NewError(consts.ACTION_FAIL, msg)
}

func (rer *DeployGcuSharePlugin) deployResources(objectSets []client.Object, status *base.ProcessStatus) error {
	for _, resObj := range objectSets {
		gvk := resObj.GetObjectKind().GroupVersionKind()
		kind, name, namespace := gvk.Kind, resObj.GetName(), resObj.GetNamespace()
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster start, content: %s", kind, name, namespace,
			consts.GCUSharePlugin, common.ChangeInterfaceToString(resObj))
		if err := status.CreateResourceToCluster(resObj); err != nil {
			return err
		}
		resObj.GetObjectKind().SetGroupVersionKind(gvk)
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster success, gr name:%s, uuid:%s",
			kind, name, namespace, consts.GCUSharePlugin, rer.Name, rer.UUID)
	}
	logs.Info("deploy all resource of %s to cluster success, count: %d", consts.GCUSharePlugin, len(objectSets))
	return nil
}

func (rer *DeployGcuSharePlugin) getObjectSets(objectSets *[]client.Object) error {
	files, err := os.ReadDir(base.GcuSharePluginPath)
	if err != nil {
		logs.Error(err, "read directory: %s failed", base.GcuSharePluginPath)
		return gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	for _, file := range files {
		resObj, err := rer.readAndParseBlueprint(file.Name())
		if err != nil {
			return err
		}
		*objectSets = append(*objectSets, resObj)
	}
	logs.Info("read and parse all cluster resource of %s finish", consts.GCUSharePlugin)
	return nil
}

func (rer *DeployGcuSharePlugin) readAndParseBlueprint(fileName string) (client.Object, error) {
	filePath := fmt.Sprintf("%s/%s", base.GcuSharePluginPath, fileName)
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
		if err := rer.instanceGCUSharePlugin(realObj); err != nil {
			return nil, err
		}
		if base.NeedInstallContainerToolkit(rer.GcuResource) {
			base.AddDsNodeSelector(realObj, map[string]string{argsConsts.ContainerToolkitReady: "true"})
		} else if base.NeedInstallDriver(rer.GcuResource) {
			base.AddDsNodeSelector(realObj, map[string]string{consts.DriverInstalledLabel: "true"})
		}
		base.AddCommandArgs(realObj, rer.GcuResource.Spec.GCUShare.DevicePlugin.Command,
			rer.GcuResource.Spec.GCUShare.DevicePlugin.Args)
	}
	base.AddLabels(resObj, rer.Name, consts.GCUSharePlugin)
	base.AddOwnereference(resObj, rer.GcuResource)
	logs.Info("read %s(name: %s, namespace: %s) of %s success", resObj.GetObjectKind().GroupVersionKind().Kind,
		resObj.GetName(), resObj.GetNamespace(), consts.GCUSharePlugin)
	return resObj, nil
}

func (rer *DeployGcuSharePlugin) instanceGCUSharePlugin(resObj *appsv1.DaemonSet) error {
	if rer.GcuResource.Spec.GCUShare.DevicePlugin.Namespace != "" {
		resObj.SetNamespace(rer.GcuResource.Spec.GCUShare.DevicePlugin.Namespace)
	}
	GcuSharePluginSpec := rer.GcuResource.Spec.GCUShare.DevicePlugin
	baseFields := v1.BaseFields{
		Name:            GcuSharePluginSpec.Name,
		Namespace:       GcuSharePluginSpec.Namespace,
		Image:           GcuSharePluginSpec.Image,
		Version:         GcuSharePluginSpec.Version,
		Repository:      GcuSharePluginSpec.Repository,
		ImagePullPolicy: GcuSharePluginSpec.ImagePullPolicy,
		NodeSelector:    GcuSharePluginSpec.NodeSelector,
		Env:             GcuSharePluginSpec.Env,
	}
	if err := base.InstaceResObject(resObj, &baseFields); err != nil {
		logs.Error(err, "instance gcusharePlugin object failed")
		return err
	}
	rer.setCommandLineParams(resObj, GcuSharePluginSpec)
	logs.Info("instance ds: %s of %s success", resObj.Name, consts.GCUSharePlugin)
	return nil
}

func (rer *DeployGcuSharePlugin) setCommandLineParams(resObj *appsv1.DaemonSet, gcuSharePluginSpec v1.GCUShareDevicePluginSpec) {
	if gcuSharePluginSpec.SliceCount != nil {
		param := fmt.Sprintf("--slice-count=%d", *gcuSharePluginSpec.SliceCount)
		resObj.Spec.Template.Spec.Containers[0].Command = append(resObj.Spec.Template.Spec.Containers[0].Command, param)
		logs.Info("reset gcushare device plugin slice count: %d", *gcuSharePluginSpec.SliceCount)
	}
}

func (rer *DeployGcuSharePlugin) updateStatus(status *base.ProcessStatus, err *error, objectSets *[]client.Object,
	deploy *bool) {
	// create gcusharePlugin failed, need not modify readyResource
	if *err != nil || !*deploy {
		return
	}
	// check gcusharePlugin ds run status
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
	// check daemonset running, gcusharePlugin deploy ok
	if isRunning {
		logs.Info("check all resource of %s run status running", consts.GCUSharePlugin)
	} else {
		msg := fmt.Sprintf("check ds of %s of gr name:%s, uuid:%s is not running", consts.GCUSharePlugin, rer.Name, rer.UUID)
		logs.Warn(msg)
		*err = gcuErr.NewError(consts.ACTION_RETRY, msg)
	}
}
