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

type DeleteNodeExporter struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeleteNodeExporter{}

func NewDeleteNodeExporter(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeleteNodeExporter{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeleteNodeExporter) GetProcedureName() string {
	return "DeleteNodeExporter"
}

func (rer *DeleteNodeExporter) GetProcedureStatus() []string {
	return []string{gcuStatus.DELETE_GCU_EXPORTER_SUCCESS, gcuStatus.DELETE_NODE_EXPORTER_FAILED}
}

func (rer *DeleteNodeExporter) GetProcedureOKStatus() string {
	return gcuStatus.DELETE_NODE_EXPORTER_SUCCESS
}

func (rer *DeleteNodeExporter) GetProcedureFailStatus() string {
	return gcuStatus.DELETE_NODE_EXPORTER_FAILED
}

func (rer *DeleteNodeExporter) GetProcedureOKProgress() int {
	return gcuPro.DELETE_NODE_EXPORTER_SUCCESS
}

func (rer *DeleteNodeExporter) GetProcedureFailProgress() int {
	return gcuPro.DELETE_NODE_EXPORTER_FAILED
}

func (rer *DeleteNodeExporter) GetProcedureAction() string {
	return consts.DELETE
}

func (rer *DeleteNodeExporter) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeleteNodeExporter) GetClient() client.Client {
	return rer.Client
}

func (rer *DeleteNodeExporter) Execute() (err error) {
	// logic here
	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	if rer.GcuResource.Spec.NodeExporter == nil || (rer.GcuResource.Spec.NodeExporter.Deploy != nil &&
		!*rer.GcuResource.Spec.NodeExporter.Deploy) {
		logs.Info("resource %s need not delete, because it not deployed", consts.NodeExporter)
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:      consts.NodeExporter,
		Namespace: base.GetNamespace(consts.NodeExporter, rer.GcuResource.Spec.NodeExporter.Namespace),
		Status:    gcuStatus.DELETING,
	}
	resourceStatuses := rer.GcuResource.Status.ResourceStatuses
	for _, resourceStatus := range resourceStatuses {
		if resourceStatus.Name == consts.NodeExporter {
			resourceInfo := resourceStatus.OwnResources
			for _, resInfo := range resourceInfo {
				namespaceName := k8sTypes.NamespacedName{Namespace: resourceStatus.Namespace, Name: resInfo.Name}
				if err := base.DeleteClusterResource(rer.Client, resInfo, &namespaceName); err != nil {
					return err
				}
				logs.Info("delete cluster resource(kind:%s, name:%s, namespace:%s) of %s success",
					resInfo.Kind, resInfo.Name, resourceStatus.Namespace, consts.NodeExporter)
			}
		}
	}
	return nil
}
