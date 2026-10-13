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

type DeployGcuDriver struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeployGcuDriver{}

func NewDeployGcuDriver(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeployGcuDriver{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeployGcuDriver) GetProcedureName() string {
	return "DeployGcuDriver"
}

func (rer *DeployGcuDriver) GetProcedureStatus() []string {
	return []string{gcuStatus.DEPLOY_NFD_SUCCESS, gcuStatus.DEPLOY_DRIVER_FAILED}
}

func (rer *DeployGcuDriver) GetProcedureOKStatus() string {
	return gcuStatus.DEPLOY_DRIVER_SUCCESS
}

func (rer *DeployGcuDriver) GetProcedureFailStatus() string {
	return gcuStatus.DEPLOY_DRIVER_FAILED
}

func (rer *DeployGcuDriver) GetProcedureOKProgress() int {
	return gcuPro.DEPLOY_DRIVER_SUCCESS
}

func (rer *DeployGcuDriver) GetProcedureFailProgress() int {
	return gcuPro.DEPLOY_DRIVER_FAILED
}

func (rer *DeployGcuDriver) GetProcedureAction() string {
	return consts.DEPLOY
}

func (rer *DeployGcuDriver) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeployGcuDriver) GetClient() client.Client {
	return rer.Client
}

func (rer *DeployGcuDriver) Execute() (err error) {
	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	var objectSets, deploy = []client.Object{}, true
	defer rer.updateStatus(status, &err, &objectSets, &deploy)

	//logic here
	if !base.NeedInstallDriver(rer.GcuResource) {
		logs.Warn("resource %s need not deploy, will skip this procedure", consts.Driver)
		deploy = false
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:         consts.Driver,
		Namespace:    base.GetNamespace(consts.Driver, rer.GcuResource.Spec.Driver.Namespace),
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

func (rer *DeployGcuDriver) deployResources(objectSets []client.Object, status *base.ProcessStatus) error {
	for _, resObj := range objectSets {
		gvk := resObj.GetObjectKind().GroupVersionKind()
		kind, name, namespace := gvk.Kind, resObj.GetName(), resObj.GetNamespace()
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster start, content: %s", kind, name, namespace,
			consts.Driver, common.ChangeInterfaceToString(resObj))
		if err := status.CreateResourceToCluster(resObj); err != nil {
			return err
		}
		resObj.GetObjectKind().SetGroupVersionKind(gvk)
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster success, gr name:%s, uuid:%s",
			kind, name, namespace, consts.Driver, rer.Name, rer.UUID)
	}
	logs.Info("deploy all resource of %s to cluster success, count: %d", consts.Driver, len(objectSets))
	return nil
}

func (rer *DeployGcuDriver) getObjectSets(objectSets *[]client.Object) error {
	files, err := os.ReadDir(base.GcuDriverPath)
	if err != nil {
		logs.Error(err, "read directory: %s failed", base.GcuDriverPath)
		return gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	for _, file := range files {
		resObj, err := rer.readAndParseBlueprint(file.Name())
		if err != nil {
			return err
		}
		*objectSets = append(*objectSets, resObj)
	}
	logs.Info("read and parse all cluster resource of %s finish", consts.Driver)
	return nil
}

func (rer *DeployGcuDriver) readAndParseBlueprint(fileName string) (client.Object, error) {
	filePath := fmt.Sprintf("%s/%s", base.GcuDriverPath, fileName)
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
		if err := rer.instanceDriver(realObj); err != nil {
			return nil, err
		}
		base.AddCommandArgs(realObj, rer.GcuResource.Spec.Driver.Command, rer.GcuResource.Spec.Driver.Args)
	}
	base.AddLabels(resObj, rer.Name, consts.Driver)
	base.AddOwnereference(resObj, rer.GcuResource)
	logs.Info("read %s(name: %s, namespace: %s) of %s success", resObj.GetObjectKind().GroupVersionKind().Kind,
		resObj.GetName(), resObj.GetNamespace(), consts.Driver)
	return resObj, nil
}

func (rer *DeployGcuDriver) instanceDriver(resObj *appsv1.DaemonSet) error {
	if rer.GcuResource.Spec.Driver.Namespace != "" {
		resObj.SetNamespace(rer.GcuResource.Spec.Driver.Namespace)
	}
	DriverSpec := rer.GcuResource.Spec.Driver
	if DriverSpec.NoS60GCheck != nil && *DriverSpec.NoS60GCheck {
		logs.Info("deploy driver set env NO_S60G_CHECK to true")
		DriverSpec.Env = append(DriverSpec.Env, corev1.EnvVar{
			Name:  "NO_S60G_CHECK",
			Value: "true",
		})
	}
	if DriverSpec.OneShot != nil && *DriverSpec.OneShot {
		logs.Info("deploy driver set env ONE_SHOT to true")
		DriverSpec.Env = append(DriverSpec.Env, corev1.EnvVar{
			Name:  "ONE_SHOT",
			Value: "true",
		})
	}
	baseFields := v1.BaseFields{
		Name:            DriverSpec.Name,
		Namespace:       DriverSpec.Namespace,
		Image:           DriverSpec.Image,
		Version:         DriverSpec.Version,
		Repository:      DriverSpec.Repository,
		ImagePullPolicy: DriverSpec.ImagePullPolicy,
		NodeSelector:    DriverSpec.NodeSelector,
		Env:             DriverSpec.Env,
	}
	if err := base.InstaceResObject(resObj, &baseFields); err != nil {
		logs.Error(err, "instance driver object failed")
		return err
	}
	logs.Info("instance ds: %s of %s success", resObj.Name, consts.Driver)
	return nil
}

func (rer *DeployGcuDriver) updateStatus(status *base.ProcessStatus, err *error, objectSets *[]client.Object,
	deploy *bool) {
	// create driver failed, need not modify readyResource
	if *err != nil || !*deploy {
		return
	}
	// check driver ds run status
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
	// check daemonset running, driver deploy ok
	if isRunning {
		status.IsReady = true
		logs.Info("check all resource of %s run status running", consts.Driver)
	} else {
		msg := fmt.Sprintf("check ds of %s of gr name:%s, uuid:%s is not running", consts.Driver, rer.Name, rer.UUID)
		logs.Warn(msg)
		*err = gcuErr.NewError(consts.ACTION_RETRY, msg)
	}
}
