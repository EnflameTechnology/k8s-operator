// Copyright (c) 2022 Enflame. All Rights Reserved.

package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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

type DeployGcuShareScheduler struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeployGcuShareScheduler{}

func NewDeployGcuShareScheduler(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeployGcuShareScheduler{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeployGcuShareScheduler) GetProcedureName() string {
	return "DeployGcuShareScheduler"
}

func (rer *DeployGcuShareScheduler) GetProcedureStatus() []string {
	return []string{gcuStatus.DEPLOY_GCUSHARE_PLUGIN_SUCCESS, gcuStatus.DEPLOY_GCUSHARE_SCHEDULER_FAILED}
}

func (rer *DeployGcuShareScheduler) GetProcedureOKStatus() string {
	return gcuStatus.DEPLOY_GCUSHARE_SCHEDULER_SUCCESS
}

func (rer *DeployGcuShareScheduler) GetProcedureFailStatus() string {
	return gcuStatus.DEPLOY_GCUSHARE_SCHEDULER_FAILED
}

func (rer *DeployGcuShareScheduler) GetProcedureOKProgress() int {
	return gcuPro.DEPLOY_GCUSHARE_SCHEDULER_SUCCESS
}

func (rer *DeployGcuShareScheduler) GetProcedureFailProgress() int {
	return gcuPro.DEPLOY_GCUSHARE_SCHEDULER_FAILED
}

func (rer *DeployGcuShareScheduler) GetProcedureAction() string {
	return consts.DEPLOY
}

func (rer *DeployGcuShareScheduler) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeployGcuShareScheduler) GetClient() client.Client {
	return rer.Client
}

func (rer *DeployGcuShareScheduler) Execute() (err error) {

	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	var objectSets, deploy = []client.Object{}, true
	defer rer.updateStatus(status, &err, &objectSets, &deploy)

	//logic here
	if rer.GcuResource.Spec.GCUShare == nil || (rer.GcuResource.Spec.GCUShare.Deploy != nil && !*rer.GcuResource.Spec.GCUShare.Deploy) {
		logs.Warn("resource %s need not deploy, will skip this procedure", consts.GCUShareScheduler)
		deploy = false
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:         consts.GCUShareScheduler,
		Namespace:    base.GetNamespace(consts.GCUShareScheduler, rer.GcuResource.Spec.GCUShare.SchedulerPlugin.Namespace),
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

func (rer *DeployGcuShareScheduler) deployResources(objectSets []client.Object, status *base.ProcessStatus) error {
	for _, resObj := range objectSets {
		gvk := resObj.GetObjectKind().GroupVersionKind()
		kind, name, namespace := gvk.Kind, resObj.GetName(), resObj.GetNamespace()
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster start, content: %s", kind, name, namespace,
			consts.GCUShareScheduler, common.ChangeInterfaceToString(resObj))
		if err := status.CreateResourceToCluster(resObj); err != nil {
			return err
		}
		resObj.GetObjectKind().SetGroupVersionKind(gvk)
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster success, gr name:%s, uuid:%s",
			kind, name, namespace, consts.GCUShareScheduler, rer.Name, rer.UUID)
	}
	logs.Info("deploy all resource of %s to cluster success, count: %d", consts.GCUShareScheduler, len(objectSets))
	return nil
}

func (rer *DeployGcuShareScheduler) getObjectSets(objectSets *[]client.Object) error {
	files, err := os.ReadDir(base.GcuShareSchedulerPath)
	if err != nil {
		logs.Error(err, "read directory: %s failed", base.GcuShareSchedulerPath)
		return gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	for _, file := range files {
		resObj, err := rer.readAndParseBlueprint(file.Name())
		if err != nil {
			return err
		}
		*objectSets = append(*objectSets, resObj)
	}
	logs.Info("read and parse all cluster resource of %s finish", consts.GCUShareScheduler)
	return nil
}

func (rer *DeployGcuShareScheduler) readAndParseBlueprint(fileName string) (client.Object, error) {
	filePath := fmt.Sprintf("%s/%s", base.GcuShareSchedulerPath, fileName)
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
	case *corev1.ConfigMap:
		if err := rer.handleCondigMap(realObj); err != nil {
			return nil, err
		}
	case *appsv1.Deployment:
		if err := rer.instanceScheduler(realObj); err != nil {
			return nil, err
		}
		base.AddCommandArgs(realObj, rer.GcuResource.Spec.GCUShare.SchedulerPlugin.Command, rer.GcuResource.Spec.GCUShare.SchedulerPlugin.Args)
	}
	base.AddLabels(resObj, rer.Name, consts.GCUShareScheduler)
	base.AddOwnereference(resObj, rer.GcuResource)
	logs.Info("read %s(name: %s, namespace: %s) of %s success", resObj.GetObjectKind().GroupVersionKind().Kind,
		resObj.GetName(), resObj.GetNamespace(), consts.GCUShareScheduler)
	return resObj, nil
}

func (rer *DeployGcuShareScheduler) handleCondigMap(realObj *corev1.ConfigMap) error {
	kubeletVersion := ""
	for nodeName, nodeInfo := range rer.GcuResource.Status.NodesInfo {
		if nodeInfo.IsGCUNode {
			kubeletVersion = nodeInfo.KubeletVersion
			logs.Info("detected node: %s kubelet version: %s", nodeName, kubeletVersion)
			break
		}
	}
	versionGe, err := base.VersionGreaterOrEqual(kubeletVersion, "v1.24.0")
	if err != nil {
		return gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	if versionGe {
		return nil
	}
	logs.Warn("kubeconfiguration will use group version: kubescheduler.config.k8s.io/v1beta1")
	realObj.Data[consts.GCUShareConfigFile] = strings.ReplaceAll(realObj.Data[consts.GCUShareConfigFile],
		"apiVersion: kubescheduler.config.k8s.io/v1", "apiVersion: kubescheduler.config.k8s.io/v1beta1")
	return nil
}

func (rer *DeployGcuShareScheduler) instanceScheduler(resObj *appsv1.Deployment) error {
	if rer.GcuResource.Spec.GCUShare.SchedulerPlugin.Namespace != "" {
		resObj.SetNamespace(rer.GcuResource.Spec.GCUShare.SchedulerPlugin.Namespace)
	}
	SchedulerSpec := rer.GcuResource.Spec.GCUShare.SchedulerPlugin
	baseFields := v1.BaseFields{
		Name:            SchedulerSpec.Name,
		Namespace:       SchedulerSpec.Namespace,
		Image:           SchedulerSpec.Image,
		Version:         SchedulerSpec.Version,
		Repository:      SchedulerSpec.Repository,
		ImagePullPolicy: SchedulerSpec.ImagePullPolicy,
		NodeSelector:    SchedulerSpec.NodeSelector,
		Env:             SchedulerSpec.Env,
	}
	if SchedulerSpec.Replicas != nil {
		resObj.Spec.Replicas = SchedulerSpec.Replicas
	}
	if err := base.InstaceResObject(resObj, &baseFields); err != nil {
		logs.Error(err, "instance gcushare scheduler object failed")
		return err
	}
	logs.Info("instance deployment: %s of %s success", resObj.Name, consts.GCUShareScheduler)
	return nil
}

func (rer *DeployGcuShareScheduler) updateStatus(status *base.ProcessStatus, err *error, objectSets *[]client.Object,
	deploy *bool) {
	// create gcushare scheduler failed, need not modify readyResource
	if *err != nil || !*deploy {
		return
	}
	// check gcushare scheduler ds run status
	namespacedNames := []*k8sTypes.NamespacedName{}
	for _, resObj := range *objectSets {
		if resObj.GetObjectKind().GroupVersionKind().Kind == object.Deployment {
			namespacedNames = append(namespacedNames, &k8sTypes.NamespacedName{
				Namespace: resObj.GetNamespace(),
				Name:      resObj.GetName(),
			})
		}
	}
	isRunning, errCheck := base.DecoratorForCheckController(base.CheckDeploymentsRunning, rer.Client, namespacedNames)
	// check daemonset status failed, return and requeue reconciler
	if errCheck != nil {
		*err = errCheck
		return
	}
	// check daemonset running, gcushare scheduler deploy ok
	if isRunning {
		status.IsReady = true
		logs.Info("check all resource of %s run status running", consts.GCUShareScheduler)
	} else {
		msg := fmt.Sprintf("check ds of %s of gr name:%s, uuid:%s is not running", consts.GCUShareScheduler, rer.Name, rer.UUID)
		logs.Warn(msg)
		*err = gcuErr.NewError(consts.ACTION_RETRY, msg)
	}
}
