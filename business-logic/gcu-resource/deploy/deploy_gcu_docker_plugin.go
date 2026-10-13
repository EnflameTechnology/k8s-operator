// Copyright (c) 2022 Enflame. All Rights Reserved.

package deploy

import (
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
	"gcu-operator/pkg/common"
	"gcu-operator/pkg/consts"
	gcuErr "gcu-operator/pkg/error"
	"gcu-operator/pkg/logs"
	"gcu-operator/pkg/object"
)

type DeployGcuRuntime struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeployGcuRuntime{}

func NewDeployGcuRuntime(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeployGcuRuntime{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeployGcuRuntime) GetProcedureName() string {
	return "DeployGcuRuntime"
}

func (rer *DeployGcuRuntime) GetProcedureStatus() []string {
	return []string{gcuStatus.DEPLOY_DRIVER_SUCCESS, gcuStatus.DEPLOY_TOOLKIT_FAILED}
}

func (rer *DeployGcuRuntime) GetProcedureOKStatus() string {
	return gcuStatus.DEPLOY_TOOLKIT_SUCCESS
}

func (rer *DeployGcuRuntime) GetProcedureFailStatus() string {
	return gcuStatus.DEPLOY_TOOLKIT_FAILED
}

func (rer *DeployGcuRuntime) GetProcedureOKProgress() int {
	return gcuPro.DEPLOY_TOOLKIT_SUCCESS
}

func (rer *DeployGcuRuntime) GetProcedureFailProgress() int {
	return gcuPro.DEPLOY_TOOLKIT_FAILED
}

func (rer *DeployGcuRuntime) GetProcedureAction() string {
	return consts.DEPLOY
}

func (rer *DeployGcuRuntime) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeployGcuRuntime) GetClient() client.Client {
	return rer.Client
}

func (rer *DeployGcuRuntime) Execute() (err error) {
	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	var objectSets, deploy = []client.Object{}, true
	defer rer.updateStatus(status, &err, &objectSets, &deploy)

	//logic here
	if !base.NeedInstallContainerToolkit(rer.GcuResource) {
		logs.Warn("resource %s need not deploy, will skip this procedure", consts.Toolkit)
		deploy = false
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:         consts.Toolkit,
		Namespace:    base.GetNamespace(consts.Toolkit, rer.GcuResource.Spec.ContainerToolkit.Namespace),
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

func (rer *DeployGcuRuntime) deployResources(objectSets []client.Object, status *base.ProcessStatus) error {
	for _, resObj := range objectSets {
		gvk := resObj.GetObjectKind().GroupVersionKind()
		kind, name, namespace := gvk.Kind, resObj.GetName(), resObj.GetNamespace()
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster start, content: %s", kind, name, namespace,
			consts.Toolkit, common.ChangeInterfaceToString(resObj))
		if err := status.CreateResourceToCluster(resObj); err != nil {
			return err
		}
		resObj.GetObjectKind().SetGroupVersionKind(gvk)
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster success, gr name:%s, uuid:%s",
			kind, name, namespace, consts.Toolkit, rer.Name, rer.UUID)
	}
	logs.Info("deploy all resource of %s to cluster success, count: %d", consts.Toolkit, len(objectSets))
	return nil
}

func (rer *DeployGcuRuntime) getObjectSets(objectSets *[]client.Object) error {
	files, err := os.ReadDir(base.GcuDockerPluginPath)
	if err != nil {
		logs.Error(err, "read directory: %s failed", base.GcuDockerPluginPath)
		return gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	for _, file := range files {
		resObj, err := rer.readAndParseBlueprint(file.Name())
		if err != nil {
			return err
		}
		*objectSets = append(*objectSets, resObj)
	}
	logs.Info("read and parse all cluster resource of %s finish", consts.Toolkit)
	return nil
}

func (rer *DeployGcuRuntime) readAndParseBlueprint(fileName string) (client.Object, error) {
	filePath := fmt.Sprintf("%s/%s", base.GcuDockerPluginPath, fileName)
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
		if err := rer.instanceToolkit(realObj); err != nil {
			return nil, err
		}
		if base.NeedInstallDriver(rer.GcuResource) {
			base.AddDsNodeSelector(realObj, map[string]string{consts.DriverInstalledLabel: "true"})
		}
		base.AddCommandArgs(realObj, rer.GcuResource.Spec.ContainerToolkit.Command, rer.GcuResource.Spec.ContainerToolkit.Args)
	}
	base.AddLabels(resObj, rer.Name, consts.Toolkit)
	base.AddOwnereference(resObj, rer.GcuResource)
	logs.Info("read %s(name: %s, namespace: %s) of %s success", resObj.GetObjectKind().GroupVersionKind().Kind,
		resObj.GetName(), resObj.GetNamespace(), consts.Toolkit)
	return resObj, nil
}

func (rer *DeployGcuRuntime) instanceToolkit(resObj *appsv1.DaemonSet) error {
	if rer.GcuResource.Spec.ContainerToolkit.Namespace != "" {
		resObj.SetNamespace(rer.GcuResource.Spec.ContainerToolkit.Namespace)
	}
	ToolkitSpec := rer.GcuResource.Spec.ContainerToolkit
	baseFields := v1.BaseFields{
		Name:            ToolkitSpec.Name,
		Namespace:       ToolkitSpec.Namespace,
		Image:           ToolkitSpec.Image,
		Version:         ToolkitSpec.Version,
		Repository:      ToolkitSpec.Repository,
		ImagePullPolicy: ToolkitSpec.ImagePullPolicy,
		NodeSelector:    ToolkitSpec.NodeSelector,
		Env:             ToolkitSpec.Env,
	}
	if err := base.InstaceResObject(resObj, &baseFields); err != nil {
		logs.Error(err, "instance toolkit object failed")
		return err
	}
	if err := rer.setEnv(resObj); err != nil {
		return err
	}
	logs.Info("instance ds: %s of %s success", resObj.Name, consts.Toolkit)
	return nil
}

func (rer *DeployGcuRuntime) setEnv(resObj *appsv1.DaemonSet) error {
	nodeInfo := rer.GcuResource.Status.NodesInfo
	content, err := json.Marshal(nodeInfo)
	if err != nil {
		logs.Error(err, "json marshal failed, content: %v", nodeInfo)
		return gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	base.AddDsEnv(resObj, []corev1.EnvVar{{Name: consts.NODE_VERSION, Value: string(content)}})
	return nil
}

func (rer *DeployGcuRuntime) updateStatus(status *base.ProcessStatus, err *error, objectSets *[]client.Object,
	deploy *bool) {
	// create toolkit failed, need not modify readyResource
	if *err != nil || !*deploy {
		return
	}
	// check toolkit ds run status
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
	// check daemonset running, toolkit deploy ok
	if isRunning {
		status.IsReady = true
		logs.Info("check all resource of %s run status running", consts.Toolkit)
	} else {
		msg := fmt.Sprintf("check ds of %s of gr name:%s, uuid:%s is not running", consts.Toolkit, rer.Name, rer.UUID)
		logs.Warn(msg)
		*err = gcuErr.NewError(consts.ACTION_RETRY, msg)
	}
}
