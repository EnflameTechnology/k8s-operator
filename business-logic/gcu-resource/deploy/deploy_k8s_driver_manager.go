// Copyright (c) 2022 Enflame. All Rights Reserved.

package deploy

import (
	"encoding/json"
	"fmt"
	"os"

	appsv1 "k8s.io/api/apps/v1"
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

type DeployK8sDriverManager struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeployK8sDriverManager{}

func NewDeployK8sDriverManager(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeployK8sDriverManager{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeployK8sDriverManager) GetProcedureName() string {
	return "DeployK8sDriverManager"
}

func (rer *DeployK8sDriverManager) GetProcedureStatus() []string {
	return []string{gcuStatus.DEPLOY_GCUSHARE_SCHEDULER_SUCCESS, gcuStatus.DEPLOY_K8S_DRIVER_MANAGER_FAILED}
}

func (rer *DeployK8sDriverManager) GetProcedureOKStatus() string {
	return gcuStatus.DEPLOY_K8S_DRIVER_MANAGER_SUCCESS
}

func (rer *DeployK8sDriverManager) GetProcedureFailStatus() string {
	return gcuStatus.DEPLOY_K8S_DRIVER_MANAGER_FAILED
}

func (rer *DeployK8sDriverManager) GetProcedureOKProgress() int {
	return gcuPro.DEPLOY_K8S_DRIVER_MANAGER_SUCCESS
}

func (rer *DeployK8sDriverManager) GetProcedureFailProgress() int {
	return gcuPro.DEPLOY_K8S_DRIVER_MANAGER_FAILED
}

func (rer *DeployK8sDriverManager) GetProcedureAction() string {
	return consts.DEPLOY
}

func (rer *DeployK8sDriverManager) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeployK8sDriverManager) GetClient() client.Client {
	return rer.Client
}

func (rer *DeployK8sDriverManager) Execute() (err error) {

	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	var objectSets, deploy = []client.Object{}, true
	defer rer.updateStatus(status, &err, &objectSets, &deploy)

	//logic here
	if rer.GcuResource.Spec.K8sDriverManager == nil || (rer.GcuResource.Spec.K8sDriverManager.Deploy != nil &&
		!*rer.GcuResource.Spec.K8sDriverManager.Deploy) {
		logs.Warn("resource %s need not deploy, will skip this procedure", consts.K8sDriverManager)
		deploy = false
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:         consts.K8sDriverManager,
		Namespace:    base.GetNamespace(consts.K8sDriverManager, rer.GcuResource.Spec.K8sDriverManager.Namespace),
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

func (rer *DeployK8sDriverManager) deployResources(objectSets []client.Object, status *base.ProcessStatus) error {
	for _, resObj := range objectSets {
		gvk := resObj.GetObjectKind().GroupVersionKind()
		kind, name, namespace := gvk.Kind, resObj.GetName(), resObj.GetNamespace()
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster start, content: %s", kind, name, namespace,
			consts.K8sDriverManager, common.ChangeInterfaceToString(resObj))
		if err := status.CreateResourceToCluster(resObj); err != nil {
			return err
		}
		resObj.GetObjectKind().SetGroupVersionKind(gvk)
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster success, gr name:%s, uuid:%s",
			kind, name, namespace, consts.K8sDriverManager, rer.Name, rer.UUID)
	}
	logs.Info("deploy all resource of %s to cluster success, count: %d", consts.K8sDriverManager, len(objectSets))
	return nil
}

func (rer *DeployK8sDriverManager) getObjectSets(objectSets *[]client.Object) error {
	files, err := os.ReadDir(base.K8sDriverManagerPath)
	if err != nil {
		logs.Error(err, "read directory: %s failed", base.K8sDriverManagerPath)
		return gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	for _, file := range files {
		resObj, err := rer.readAndParseBlueprint(file.Name())
		if err != nil {
			return err
		}
		*objectSets = append(*objectSets, resObj)
	}
	logs.Info("read and parse all cluster resource of %s finish", consts.K8sDriverManager)
	return nil
}

func (rer *DeployK8sDriverManager) readAndParseBlueprint(fileName string) (client.Object, error) {
	filePath := fmt.Sprintf("%s/%s", base.K8sDriverManagerPath, fileName)
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
		if err := rer.instanceK8sDriverManager(realObj); err != nil {
			return nil, err
		}
		if base.NeedInstallDriver(rer.GcuResource) {
			base.AddDsNodeSelector(realObj, map[string]string{consts.DriverInstalledLabel: "true"})
		}
		base.AddCommandArgs(realObj, rer.GcuResource.Spec.K8sDriverManager.Command, rer.GcuResource.Spec.K8sDriverManager.Args)
	}
	base.AddLabels(resObj, rer.Name, consts.K8sDriverManager)
	base.AddOwnereference(resObj, rer.GcuResource)
	logs.Info("read %s(name: %s, namespace: %s) of %s success", resObj.GetObjectKind().GroupVersionKind().Kind,
		resObj.GetName(), resObj.GetNamespace(), consts.K8sDriverManager)
	return resObj, nil
}

func (rer *DeployK8sDriverManager) instanceK8sDriverManager(resObj *appsv1.DaemonSet) error {
	if rer.GcuResource.Spec.K8sDriverManager.Namespace != "" {
		resObj.SetNamespace(rer.GcuResource.Spec.K8sDriverManager.Namespace)
	}
	K8sDriverManagerSpec := rer.GcuResource.Spec.K8sDriverManager
	baseFields := v1.BaseFields{
		Name:            K8sDriverManagerSpec.Name,
		Namespace:       K8sDriverManagerSpec.Namespace,
		Image:           K8sDriverManagerSpec.Image,
		Version:         K8sDriverManagerSpec.Version,
		Repository:      K8sDriverManagerSpec.Repository,
		ImagePullPolicy: K8sDriverManagerSpec.ImagePullPolicy,
		NodeSelector:    K8sDriverManagerSpec.NodeSelector,
		Env:             K8sDriverManagerSpec.Env,
	}
	if err := base.InstaceResObject(resObj, &baseFields); err != nil {
		logs.Error(err, "instance k8s driver manager object failed")
		return err
	}
	logs.Info("instance ds: %s of %s success", resObj.Name, consts.K8sDriverManager)
	return nil
}

func (rer *DeployK8sDriverManager) updateStatus(status *base.ProcessStatus, err *error, objectSets *[]client.Object,
	deploy *bool) {
	// create k8s driver manager failed, need not modify readyResource
	if *err != nil || !*deploy {
		return
	}
	// check k8s driver manager ds run status
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
	// check daemonset running, k8s driver manager deploy ok
	if isRunning {

		status.IsReady = true
		logs.Info("check all resource of %s run status running", consts.K8sDriverManager)
	} else {
		msg := fmt.Sprintf("check ds of %s of gr name:%s, uuid:%s is not running", consts.K8sDriverManager, rer.Name, rer.UUID)
		logs.Warn(msg)
		*err = gcuErr.NewError(consts.ACTION_RETRY, msg)
	}
}
