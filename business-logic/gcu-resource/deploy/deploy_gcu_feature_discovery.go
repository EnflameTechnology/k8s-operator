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
	argsConsts "gcu-operator/components/common/consts"
	"gcu-operator/pkg/common"
	"gcu-operator/pkg/consts"
	gcuErr "gcu-operator/pkg/error"
	"gcu-operator/pkg/logs"
	"gcu-operator/pkg/object"
)

type DeployGcuFeatureDiscovery struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeployGcuFeatureDiscovery{}

func NewDeployGcuFeatureDiscovery(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeployGcuFeatureDiscovery{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeployGcuFeatureDiscovery) GetProcedureName() string {
	return "DeployGcuFeatureDiscovery"
}

func (rer *DeployGcuFeatureDiscovery) GetProcedureStatus() []string {
	return []string{gcuStatus.DEPLOY_GCU_EXPORTER_SUCCESS, gcuStatus.DEPLOY_GFD_FAILED}
}

func (rer *DeployGcuFeatureDiscovery) GetProcedureOKStatus() string {
	return gcuStatus.DEPLOY_GFD_SUCCESS
}

func (rer *DeployGcuFeatureDiscovery) GetProcedureFailStatus() string {
	return gcuStatus.DEPLOY_GFD_FAILED
}

func (rer *DeployGcuFeatureDiscovery) GetProcedureOKProgress() int {
	return gcuPro.DEPLOY_GFD_SUCCESS
}

func (rer *DeployGcuFeatureDiscovery) GetProcedureFailProgress() int {
	return gcuPro.DEPLOY_GFD_FAILED
}

func (rer *DeployGcuFeatureDiscovery) GetProcedureAction() string {
	return consts.DEPLOY
}

func (rer *DeployGcuFeatureDiscovery) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeployGcuFeatureDiscovery) GetClient() client.Client {
	return rer.Client
}

func (rer *DeployGcuFeatureDiscovery) Execute() (err error) {
	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	var objectSets, deploy = []client.Object{}, true
	defer rer.updateStatus(status, &err, &objectSets, &deploy)

	//logic here
	if rer.GcuResource.Spec.Gfd == nil || (rer.GcuResource.Spec.Gfd.Deploy != nil && !*rer.GcuResource.Spec.Gfd.Deploy) {
		logs.Warn("resource %s need not deploy, will skip this procedure", consts.GFD)
		deploy = false
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:         consts.GFD,
		Namespace:    base.GetNamespace(consts.GFD, rer.GcuResource.Spec.Gfd.Namespace),
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

func (rer *DeployGcuFeatureDiscovery) deployResources(objectSets []client.Object, status *base.ProcessStatus) error {
	for _, resObj := range objectSets {
		gvk := resObj.GetObjectKind().GroupVersionKind()
		kind, name, namespace := gvk.Kind, resObj.GetName(), resObj.GetNamespace()
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster start, content: %s", kind, name, namespace,
			consts.GFD, common.ChangeInterfaceToString(resObj))
		if err := status.CreateResourceToCluster(resObj); err != nil {
			return err
		}
		resObj.GetObjectKind().SetGroupVersionKind(gvk)
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster success, gr name:%s, uuid:%s",
			kind, name, namespace, consts.GFD, rer.Name, rer.UUID)
	}
	logs.Info("deploy all resource of %s to cluster success, count: %d", consts.GFD, len(objectSets))
	return nil
}

func (rer *DeployGcuFeatureDiscovery) getObjectSets(objectSets *[]client.Object) error {
	files, err := os.ReadDir(base.GcuFeatureDiscoveryPath)
	if err != nil {
		logs.Error(err, "read directory: %s failed", base.GcuFeatureDiscoveryPath)
		return gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	for _, file := range files {
		resObj, err := rer.readAndParseBlueprint(file.Name())
		if err != nil {
			return err
		}
		*objectSets = append(*objectSets, resObj)
	}
	logs.Info("read and parse all cluster resource of %s finish", consts.GFD)
	return nil
}

func (rer *DeployGcuFeatureDiscovery) readAndParseBlueprint(fileName string) (client.Object, error) {
	filePath := fmt.Sprintf("%s/%s", base.GcuFeatureDiscoveryPath, fileName)
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
		if err := rer.instanceGfd(realObj); err != nil {
			return nil, err
		}
		if base.NeedInstallContainerToolkit(rer.GcuResource) {
			base.AddDsNodeSelector(realObj, map[string]string{argsConsts.ContainerToolkitReady: "true"})
		} else if base.NeedInstallDriver(rer.GcuResource) {
			base.AddDsNodeSelector(realObj, map[string]string{consts.DriverInstalledLabel: "true"})
		}
		base.AddCommandArgs(realObj, rer.GcuResource.Spec.Gfd.Command, rer.GcuResource.Spec.Gfd.Args)
	}
	base.AddLabels(resObj, rer.Name, consts.GFD)
	base.AddOwnereference(resObj, rer.GcuResource)
	logs.Info("read %s(name: %s, namespace: %s) of %s success", resObj.GetObjectKind().GroupVersionKind().Kind,
		resObj.GetName(), resObj.GetNamespace(), consts.GFD)
	return resObj, nil
}

func (rer *DeployGcuFeatureDiscovery) instanceGfd(resObj *appsv1.DaemonSet) error {
	if rer.GcuResource.Spec.Gfd.Namespace != "" {
		resObj.SetNamespace(rer.GcuResource.Spec.Gfd.Namespace)
	}
	GfdSpec := rer.GcuResource.Spec.Gfd
	baseFields := v1.BaseFields{
		Name:            GfdSpec.Name,
		Namespace:       GfdSpec.Namespace,
		Image:           GfdSpec.Image,
		Version:         GfdSpec.Version,
		Repository:      GfdSpec.Repository,
		ImagePullPolicy: GfdSpec.ImagePullPolicy,
		NodeSelector:    GfdSpec.NodeSelector,
		Env:             GfdSpec.Env,
	}
	if err := base.InstaceResObject(resObj, &baseFields); err != nil {
		logs.Error(err, "instance gfd object failed")
		return err
	}
	logs.Info("instance ds: %s of %s success", resObj.Name, consts.GFD)
	return nil
}

func (rer *DeployGcuFeatureDiscovery) updateStatus(status *base.ProcessStatus, err *error, objectSets *[]client.Object,
	deploy *bool) {
	// create gfd failed, need not modify readyResource
	if *err != nil || !*deploy {
		return
	}
	// check gfd ds run status
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
	// check daemonset running, gfd deploy ok
	if isRunning {
		status.IsReady = true
		logs.Info("check all resource of %s run status running", consts.GFD)
	} else {
		msg := fmt.Sprintf("check ds of %s of gr name:%s, uuid:%s is not running", consts.GFD, rer.Name, rer.UUID)
		logs.Warn(msg)
		*err = gcuErr.NewError(consts.ACTION_RETRY, msg)
	}
}
