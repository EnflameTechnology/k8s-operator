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
	argsConsts "gcu-operator/components/common/consts"
	"gcu-operator/pkg/consts"
	gcuErr "gcu-operator/pkg/error"
	"gcu-operator/pkg/logs"
	"gcu-operator/pkg/object"
)

type DeleteToolkit struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeleteToolkit{}

func NewDeleteToolkit(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeleteToolkit{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeleteToolkit) GetProcedureName() string {
	return "DeleteToolkit"
}

func (rer *DeleteToolkit) GetProcedureStatus() []string {
	return []string{gcuStatus.DELETE_DEVICE_PLUGIN_SUCCESS, gcuStatus.DELETE_TOOLKIT_FAILED}
}

func (rer *DeleteToolkit) GetProcedureOKStatus() string {
	return gcuStatus.DELETE_TOOLKIT_SUCCESS
}

func (rer *DeleteToolkit) GetProcedureFailStatus() string {
	return gcuStatus.DELETE_TOOLKIT_FAILED
}

func (rer *DeleteToolkit) GetProcedureOKProgress() int {
	return gcuPro.DELETE_TOOLKIT_SUCCESS
}

func (rer *DeleteToolkit) GetProcedureFailProgress() int {
	return gcuPro.DELETE_TOOLKIT_FAILED
}

func (rer *DeleteToolkit) GetProcedureAction() string {
	return consts.DELETE
}

func (rer *DeleteToolkit) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeleteToolkit) GetClient() client.Client {
	return rer.Client
}

func (rer *DeleteToolkit) Execute() (err error) {
	// logic here
	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	if rer.GcuResource.Spec.ContainerToolkit == nil || (rer.GcuResource.Spec.ContainerToolkit.Deploy != nil &&
		!*rer.GcuResource.Spec.ContainerToolkit.Deploy) {
		logs.Info("resource %s need not delete, because it not deployed", consts.Toolkit)
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:      consts.Toolkit,
		Namespace: base.GetNamespace(consts.Toolkit, rer.GcuResource.Spec.ContainerToolkit.Namespace),
		Status:    gcuStatus.DELETING,
	}
	resourceStatuses := rer.GcuResource.Status.ResourceStatuses
	for _, resourceStatus := range resourceStatuses {
		if resourceStatus.Name == consts.Toolkit {
			resourceInfo := resourceStatus.OwnResources
			for _, resInfo := range resourceInfo {
				if resInfo.Kind == object.DaemonSet && rer.removeNodeLabel() != nil {
					return gcuErr.NewError(consts.ACTION_FAIL, err.Error())
				}
				namespaceName := k8sTypes.NamespacedName{Namespace: resourceStatus.Namespace, Name: resInfo.Name}
				if err := base.DeleteClusterResource(rer.Client, resInfo, &namespaceName); err != nil {
					return err
				}
				logs.Info("delete cluster resource(kind:%s, name:%s, namespace:%s) of %s success",
					resInfo.Kind, resInfo.Name, resourceStatus.Namespace, consts.Toolkit)
			}
		}
	}
	return nil
}

func (rer *DeleteToolkit) removeNodeLabel() error {
	labelSelector := map[string]string{
		argsConsts.ContainerToolkitReady: "true",
	}
	nodeList := &corev1.NodeList{}
	err := rer.Client.List(context.TODO(), nodeList, client.MatchingLabels(labelSelector))
	if err != nil {
		logs.Error(err, "list nodes with label %s failed", argsConsts.ContainerToolkitReady)
		return err
	}
	for i := range nodeList.Items {
		node := &nodeList.Items[i]
		patch := client.MergeFrom(node.DeepCopy())
		delete(node.Labels, argsConsts.ContainerToolkitReady)
		if err := rer.Client.Patch(context.TODO(), node, patch); err != nil {
			logs.Error(err, "remove node label %s from node %s failed",
				argsConsts.ContainerToolkitReady, node.Name)
			return err
		}
		logs.Info("remove node label %s from node %s success",
			argsConsts.ContainerToolkitReady, node.Name)
	}
	return nil
}
