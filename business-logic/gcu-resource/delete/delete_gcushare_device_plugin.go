// Copyright (c) 2022 Enflame. All Rights Reserved.

package delete

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "gcu-operator/apis/topscloud/v1"
	"gcu-operator/business-logic/base/interfaces"
	"gcu-operator/business-logic/gcu-resource/base"
	gcuPro "gcu-operator/business-logic/gcu-resource/base/progress"
	gcuStatus "gcu-operator/business-logic/gcu-resource/base/status"
	"gcu-operator/pkg/consts"
	gcuErr "gcu-operator/pkg/error"
	"gcu-operator/pkg/logs"
)

type DeleteGcuSharePlugin struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeleteGcuSharePlugin{}

func NewDeleteGcuSharePlugin(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeleteGcuSharePlugin{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeleteGcuSharePlugin) GetProcedureName() string {
	return "DeleteGcuSharePlugin"
}

func (rer *DeleteGcuSharePlugin) GetProcedureStatus() []string {
	return []string{gcuStatus.DELETE_GCUSHARE_SCHEDULER_SUCCESS, gcuStatus.DELETE_GCUSHARE_PLUGIN_FAILED}
}

func (rer *DeleteGcuSharePlugin) GetProcedureOKStatus() string {
	return gcuStatus.DELETE_GCUSHARE_PLUGIN_SUCCESS
}

func (rer *DeleteGcuSharePlugin) GetProcedureFailStatus() string {
	return gcuStatus.DELETE_GCUSHARE_PLUGIN_FAILED
}

func (rer *DeleteGcuSharePlugin) GetProcedureOKProgress() int {
	return gcuPro.DELETE_GCUSHARE_PLUGIN_SUCCESS
}

func (rer *DeleteGcuSharePlugin) GetProcedureFailProgress() int {
	return gcuPro.DELETE_GCUSHARE_PLUGIN_FAILED
}

func (rer *DeleteGcuSharePlugin) GetProcedureAction() string {
	return consts.DELETE
}

func (rer *DeleteGcuSharePlugin) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeleteGcuSharePlugin) GetClient() client.Client {
	return rer.Client
}

func (rer *DeleteGcuSharePlugin) Execute() (err error) {
	// logic here
	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	if rer.GcuResource.Spec.GCUShare == nil || (rer.GcuResource.Spec.GCUShare.Deploy != nil && !*rer.GcuResource.Spec.GCUShare.Deploy) {
		logs.Info("resource %s need not delete, because it not deployed", consts.GCUSharePlugin)
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:      consts.GCUSharePlugin,
		Namespace: base.GetNamespace(consts.GCUSharePlugin, rer.GcuResource.Spec.GCUShare.DevicePlugin.Namespace),
		Status:    gcuStatus.DELETING,
	}
	resourceStatuses := rer.GcuResource.Status.ResourceStatuses
	for _, resourceStatus := range resourceStatuses {
		if resourceStatus.Name == consts.GCUSharePlugin {
			resourceInfo := resourceStatus.OwnResources
			for _, resInfo := range resourceInfo {
				namespaceName := k8sTypes.NamespacedName{Namespace: resourceStatus.Namespace, Name: resInfo.Name}
				if err := base.DeleteClusterResource(rer.Client, resInfo, &namespaceName); err != nil {
					return err
				}
				logs.Info("delete cluster resource(kind:%s, name:%s, namespace:%s) of %s success",
					resInfo.Kind, resInfo.Name, resourceStatus.Namespace, consts.GCUSharePlugin)
			}
		}
	}
	return rer.resetNodeLabels()
}

func (rer *DeleteGcuSharePlugin) resetNodeLabels() error {
	nodeList := &corev1.NodeList{}
	if err := rer.Client.List(context.TODO(), nodeList); err != nil {
		logs.Error(err, "get cluster node list failed")
		return gcuErr.NewError(consts.ACTION_RETRY, err.Error())
	}
	for i := range nodeList.Items {
		node := &nodeList.Items[i]
		patch := client.MergeFrom(node.DeepCopy())
		_, ok := node.Labels[consts.GcuSharePluginLabel]
		if ok {
			delete(node.Labels, consts.GcuSharePluginLabel)
			if err := rer.Client.Patch(context.TODO(), node, patch); err != nil {
				logs.Error(err, "remove node: %s label %s failed", node.Name, consts.GcuSharePluginLabel)
				return gcuErr.NewError(consts.ACTION_RETRY, err.Error())
			}
			logs.Info("remove node: %s label %s success", node.Name, consts.GcuSharePluginLabel)
		}
		// It is difficult to obtain the config file of the specified node, only try to clear the default resource name
		_, ok = node.Status.Allocatable[consts.GcuCountName]
		if ok {
			patch := client.MergeFrom(node.DeepCopy())
			delete(node.Status.Allocatable, consts.GcuCountName)
			delete(node.Status.Capacity, consts.GcuCountName)
			if err := rer.Client.Status().Patch(context.TODO(), node, patch); err != nil {
				logs.Error(err, "update gcushare node: %s status field %s failed", node.Name, consts.GcuCountName)
				return gcuErr.NewError(consts.ACTION_RETRY, err.Error())
			}
			logs.Info("reset node:%s status field %s success", node.Name, consts.GcuCountName)
		}
	}
	return nil
}
