// Copyright (c) 2022 Enflame. All Rights Reserved.

package delete

import (
	"context"
	"strings"

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

type DeleteNfd struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeleteNfd{}

func NewDeleteNfd(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeleteNfd{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeleteNfd) GetProcedureName() string {
	return "DeleteNfd"
}

func (rer *DeleteNfd) GetProcedureStatus() []string {
	return []string{gcuStatus.DELETE_DRIVER_SUCCESS, gcuStatus.DELETE_NFD_FAILED}
}

func (rer *DeleteNfd) GetProcedureOKStatus() string {
	return gcuStatus.DELETE_NFD_SUCCESS
}

func (rer *DeleteNfd) GetProcedureFailStatus() string {
	return gcuStatus.DELETE_NFD_FAILED
}

func (rer *DeleteNfd) GetProcedureOKProgress() int {
	return gcuPro.DELETE_NFD_SUCCESS
}

func (rer *DeleteNfd) GetProcedureFailProgress() int {
	return gcuPro.DELETE_NFD_FAILED
}

func (rer *DeleteNfd) GetProcedureAction() string {
	return consts.DELETE
}

func (rer *DeleteNfd) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeleteNfd) GetClient() client.Client {
	return rer.Client
}

func (rer *DeleteNfd) Execute() (err error) {
	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)

	// logic here
	if rer.GcuResource.Spec.Nfd == nil || (rer.GcuResource.Spec.Nfd.Deploy != nil && !*rer.GcuResource.Spec.Nfd.Deploy) {
		logs.Info("resource %s need not delete, because it not deployed", consts.NFD)
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:      consts.NFD,
		Namespace: base.GetNamespace(consts.NFD, rer.GcuResource.Spec.Nfd.Namespace),
		Status:    gcuStatus.DELETING,
	}
	resourceStatuses := rer.GcuResource.Status.ResourceStatuses
	for _, resourceStatus := range resourceStatuses {
		if resourceStatus.Name == consts.NFD {
			resourceInfo := resourceStatus.OwnResources
			for _, resInfo := range resourceInfo {
				namespaceName := k8sTypes.NamespacedName{Namespace: resourceStatus.Namespace, Name: resInfo.Name}
				if err := base.DeleteClusterResource(rer.Client, resInfo, &namespaceName); err != nil {
					return err
				}
				logs.Info("delete cluster resource(kind:%s, name:%s, namespace:%s) of %s success",
					resInfo.Kind, resInfo.Name, resourceStatus.Namespace, consts.NFD)
			}
		}
	}
	return rer.resetNodeLabels()
}

func (rer *DeleteNfd) resetNodeLabels() error {
	nodeList := &corev1.NodeList{}
	if err := rer.Client.List(context.TODO(), nodeList); err != nil {
		logs.Error(err, "get cluster node list failed")
		return gcuErr.NewError(consts.ACTION_RETRY, err.Error())
	}
	for i := range nodeList.Items {
		node := &nodeList.Items[i]
		patch := client.MergeFrom(node.DeepCopy())
		hasNfdLabel := false
		for labelKey := range node.Labels {
			if strings.HasPrefix(labelKey, consts.NfdLabelPrefix) {
				hasNfdLabel = true
				delete(node.Labels, labelKey)
			}
		}
		if !hasNfdLabel {
			continue
		}
		if err := rer.Client.Patch(context.TODO(), node, patch); err != nil {
			logs.Error(err, "reset node:%s labels failed", node.Name)
			return gcuErr.NewError(consts.ACTION_RETRY, err.Error())
		}
		logs.Info("reset node:%s labels success", node.Name)
	}
	return nil
}
