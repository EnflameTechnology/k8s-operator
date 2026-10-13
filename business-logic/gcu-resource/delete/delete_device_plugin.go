// Copyright (c) 2022 Enflame. All Rights Reserved.

package delete

import (
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "gcu-operator/apis/topscloud/v1"
	"gcu-operator/business-logic/base/interfaces"
	"gcu-operator/business-logic/gcu-resource/base"
	gcuPro "gcu-operator/business-logic/gcu-resource/base/progress"
	gcuStatus "gcu-operator/business-logic/gcu-resource/base/status"
	"gcu-operator/pkg/consts"
	"gcu-operator/pkg/logs"
)

type DeleteDevicePlugin struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeleteDevicePlugin{}

func NewDeleteDevicePlugin(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeleteDevicePlugin{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeleteDevicePlugin) GetProcedureName() string {
	return "DeleteDevicePlugin"
}

func (rer *DeleteDevicePlugin) GetProcedureStatus() []string {
	return []string{gcuStatus.DELETE_NODE_EXPORTER_SUCCESS, gcuStatus.DELETE_DEVICE_PLUGIN_FAILED}
}

func (rer *DeleteDevicePlugin) GetProcedureOKStatus() string {
	return gcuStatus.DELETE_DEVICE_PLUGIN_SUCCESS
}

func (rer *DeleteDevicePlugin) GetProcedureFailStatus() string {
	return gcuStatus.DELETE_DEVICE_PLUGIN_FAILED
}

func (rer *DeleteDevicePlugin) GetProcedureOKProgress() int {
	return gcuPro.DELETE_DEVICE_PLUGIN_SUCCESS
}

func (rer *DeleteDevicePlugin) GetProcedureFailProgress() int {
	return gcuPro.DELETE_DEVICE_PLUGIN_FAILED
}

func (rer *DeleteDevicePlugin) GetProcedureAction() string {
	return consts.DELETE
}

func (rer *DeleteDevicePlugin) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeleteDevicePlugin) GetClient() client.Client {
	return rer.Client
}

func (rer *DeleteDevicePlugin) Execute() (err error) {
	// logic here
	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	if rer.GcuResource.Spec.DevicePlugin == nil || (rer.GcuResource.Spec.DevicePlugin.Deploy != nil &&
		!*rer.GcuResource.Spec.DevicePlugin.Deploy) {
		logs.Info("resource %s need not delete, because it not deployed", consts.Plugin)
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:      consts.Plugin,
		Namespace: base.GetNamespace(consts.Plugin, rer.GcuResource.Spec.DevicePlugin.Namespace),
		Status:    gcuStatus.DELETING,
	}
	resourceStatuses := rer.GcuResource.Status.ResourceStatuses
	for _, resourceStatus := range resourceStatuses {
		if resourceStatus.Name == consts.Plugin {
			resourceInfo := resourceStatus.OwnResources
			for _, resInfo := range resourceInfo {
				namespaceName := k8sTypes.NamespacedName{Namespace: resourceStatus.Namespace, Name: resInfo.Name}
				if err := base.DeleteClusterResource(rer.Client, resInfo, &namespaceName); err != nil {
					return err
				}
				logs.Info("delete cluster resource(kind:%s, name:%s, namespace:%s) of %s success",
					resInfo.Kind, resInfo.Name, resourceStatus.Namespace, consts.Plugin)
			}
		}
	}
	return nil
}
