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

type DeleteGcuExporter struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeleteGcuExporter{}

func NewDeleteGcuExporter(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeleteGcuExporter{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeleteGcuExporter) GetProcedureName() string {
	return "DeleteGcuExporter"
}

func (rer *DeleteGcuExporter) GetProcedureStatus() []string {
	return []string{gcuStatus.DELETE_GFD_SUCCESS, gcuStatus.DELETE_GCU_EXPORTER_FAILED}
}

func (rer *DeleteGcuExporter) GetProcedureOKStatus() string {
	return gcuStatus.DELETE_GCU_EXPORTER_SUCCESS
}

func (rer *DeleteGcuExporter) GetProcedureFailStatus() string {
	return gcuStatus.DELETE_GCU_EXPORTER_FAILED
}

func (rer *DeleteGcuExporter) GetProcedureOKProgress() int {
	return gcuPro.DELETE_GCU_EXPORTER_SUCCESS
}

func (rer *DeleteGcuExporter) GetProcedureFailProgress() int {
	return gcuPro.DELETE_GCU_EXPORTER_FAILED
}

func (rer *DeleteGcuExporter) GetProcedureAction() string {
	return consts.DELETE
}

func (rer *DeleteGcuExporter) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeleteGcuExporter) GetClient() client.Client {
	return rer.Client
}

func (rer *DeleteGcuExporter) Execute() (err error) {
	// logic here
	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	if rer.GcuResource.Spec.GcuExporter == nil || (rer.GcuResource.Spec.GcuExporter.Deploy != nil &&
		!*rer.GcuResource.Spec.GcuExporter.Deploy) {
		logs.Info("resource %s need not delete, because it not deployed", consts.GcuExporter)
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:      consts.GcuExporter,
		Namespace: base.GetNamespace(consts.GcuExporter, rer.GcuResource.Spec.GcuExporter.Namespace),
		Status:    gcuStatus.DELETING,
	}
	resourceStatuses := rer.GcuResource.Status.ResourceStatuses
	for _, resourceStatus := range resourceStatuses {
		if resourceStatus.Name == consts.GcuExporter {
			resourceInfo := resourceStatus.OwnResources
			for _, resInfo := range resourceInfo {
				namespaceName := k8sTypes.NamespacedName{Namespace: resourceStatus.Namespace, Name: resInfo.Name}
				if err := base.DeleteClusterResource(rer.Client, resInfo, &namespaceName); err != nil {
					return err
				}
				logs.Info("delete cluster resource(kind:%s, name:%s, namespace:%s) of %s success",
					resInfo.Kind, resInfo.Name, resourceStatus.Namespace, consts.GcuExporter)
			}
		}
	}
	return nil
}
