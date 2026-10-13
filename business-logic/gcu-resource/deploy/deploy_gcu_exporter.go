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

type DeployGcuExporter struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeployGcuExporter{}

func NewDeployGcuExporter(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeployGcuExporter{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeployGcuExporter) GetProcedureName() string {
	return "DeployGcuExporter"
}

func (rer *DeployGcuExporter) GetProcedureStatus() []string {
	return []string{gcuStatus.DEPLOY_NODE_EXPORTER_SUCCESS, gcuStatus.DEPLOY_GCU_EXPORTER_FAILED}
}

func (rer *DeployGcuExporter) GetProcedureOKStatus() string {
	return gcuStatus.DEPLOY_GCU_EXPORTER_SUCCESS
}

func (rer *DeployGcuExporter) GetProcedureFailStatus() string {
	return gcuStatus.DEPLOY_GCU_EXPORTER_FAILED
}

func (rer *DeployGcuExporter) GetProcedureOKProgress() int {
	return gcuPro.DEPLOY_GCU_EXPORTER_SUCCESS
}

func (rer *DeployGcuExporter) GetProcedureFailProgress() int {
	return gcuPro.DEPLOY_GCU_EXPORTER_FAILED
}

func (rer *DeployGcuExporter) GetProcedureAction() string {
	return consts.DEPLOY
}

func (rer *DeployGcuExporter) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeployGcuExporter) GetClient() client.Client {
	return rer.Client
}

func (rer *DeployGcuExporter) Execute() (err error) {
	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	var objectSets, deploy = []client.Object{}, true
	defer rer.updateStatus(status, &err, &objectSets, &deploy)

	//logic here
	if rer.GcuResource.Spec.GcuExporter == nil || (rer.GcuResource.Spec.GcuExporter.Deploy != nil &&
		!*rer.GcuResource.Spec.GcuExporter.Deploy) {
		logs.Warn("resource %s need not deploy, will skip this procedure", consts.GcuExporter)
		deploy = false
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:         consts.GcuExporter,
		Namespace:    base.GetNamespace(consts.GcuExporter, rer.GcuResource.Spec.GcuExporter.Namespace),
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

func (rer *DeployGcuExporter) deployResources(objectSets []client.Object, status *base.ProcessStatus) error {
	for _, resObj := range objectSets {
		gvk := resObj.GetObjectKind().GroupVersionKind()
		kind, name, namespace := gvk.Kind, resObj.GetName(), resObj.GetNamespace()
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster start, content: %s", kind, name, namespace,
			consts.GcuExporter, common.ChangeInterfaceToString(resObj))
		if err := status.CreateResourceToCluster(resObj); err != nil {
			return err
		}
		resObj.GetObjectKind().SetGroupVersionKind(gvk)
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster success, gr name:%s, uuid:%s",
			kind, name, namespace, consts.GcuExporter, rer.Name, rer.UUID)
	}
	logs.Info("deploy all resource of %s to cluster success, count: %d", consts.GcuExporter, len(objectSets))
	return nil
}

func (rer *DeployGcuExporter) getObjectSets(objectSets *[]client.Object) error {
	files, err := os.ReadDir(base.GcuExporterPath)
	if err != nil {
		logs.Error(err, "read directory: %s failed", base.GcuExporterPath)
		return gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	for _, file := range files {
		resObj, err := rer.readAndParseBlueprint(file.Name())
		if err != nil {
			return err
		}
		*objectSets = append(*objectSets, resObj)
	}
	logs.Info("read and parse all cluster resource of %s finish", consts.GcuExporter)
	return nil
}

func (rer *DeployGcuExporter) readAndParseBlueprint(fileName string) (client.Object, error) {
	filePath := fmt.Sprintf("%s/%s", base.GcuExporterPath, fileName)
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
		if err := rer.instanceGcuExporter(realObj); err != nil {
			return nil, err
		}
		if base.NeedInstallContainerToolkit(rer.GcuResource) {
			base.AddDsNodeSelector(realObj, map[string]string{argsConsts.ContainerToolkitReady: "true"})
		} else if base.NeedInstallDriver(rer.GcuResource) {
			base.AddDsNodeSelector(realObj, map[string]string{consts.DriverInstalledLabel: "true"})
		}
		base.AddCommandArgs(realObj, rer.GcuResource.Spec.GcuExporter.Command, rer.GcuResource.Spec.GcuExporter.Args)
	}
	base.AddLabels(resObj, rer.Name, consts.GcuExporter)
	base.AddOwnereference(resObj, rer.GcuResource)
	logs.Info("read %s(name: %s, namespace: %s) of %s success", resObj.GetObjectKind().GroupVersionKind().Kind,
		resObj.GetName(), resObj.GetNamespace(), consts.GcuExporter)
	return resObj, nil
}

func (rer *DeployGcuExporter) instanceGcuExporter(resObj *appsv1.DaemonSet) error {
	if rer.GcuResource.Spec.GcuExporter.Namespace != "" {
		resObj.SetNamespace(rer.GcuResource.Spec.GcuExporter.Namespace)
	}
	GcuExporterSpec := rer.GcuResource.Spec.GcuExporter
	baseFields := v1.BaseFields{
		Name:            GcuExporterSpec.Name,
		Namespace:       GcuExporterSpec.Namespace,
		Image:           GcuExporterSpec.Image,
		Version:         GcuExporterSpec.Version,
		Repository:      GcuExporterSpec.Repository,
		ImagePullPolicy: GcuExporterSpec.ImagePullPolicy,
		NodeSelector:    GcuExporterSpec.NodeSelector,
		Env:             GcuExporterSpec.Env,
	}
	if err := base.InstaceResObject(resObj, &baseFields); err != nil {
		logs.Error(err, "instance gcu exporter object failed")
		return err
	}
	logs.Info("instance ds: %s of %s success", resObj.Name, consts.GcuExporter)
	return nil
}

func (rer *DeployGcuExporter) updateStatus(status *base.ProcessStatus, err *error, objectSets *[]client.Object,
	deploy *bool) {
	// create gcu exporter failed, need not modify readyResource
	if *err != nil || !*deploy {
		return
	}
	// check gcu exporter ds run status
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
	// check daemonset running, gcu exporter deploy ok
	if isRunning {
		status.IsReady = true
		logs.Info("check all resource of %s run status running", consts.GcuExporter)
	} else {
		msg := fmt.Sprintf("check ds of %s of gr name:%s, uuid:%s is not running", consts.GcuExporter, rer.Name, rer.UUID)
		logs.Warn(msg)
		*err = gcuErr.NewError(consts.ACTION_RETRY, msg)
	}
}
