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

type DeleteGfd struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeleteGfd{}

func NewDeleteGfd(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeleteGfd{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeleteGfd) GetProcedureName() string {
	return "DeleteGfd"
}

func (rer *DeleteGfd) GetProcedureStatus() []string {
	return []string{gcuStatus.DELETE_GCUSHARE_PLUGIN_SUCCESS, gcuStatus.DELETE_GFD_FAILED}
}

func (rer *DeleteGfd) GetProcedureOKStatus() string {
	return gcuStatus.DELETE_GFD_SUCCESS
}

func (rer *DeleteGfd) GetProcedureFailStatus() string {
	return gcuStatus.DELETE_GFD_FAILED
}

func (rer *DeleteGfd) GetProcedureOKProgress() int {
	return gcuPro.DELETE_GFD_SUCCESS
}

func (rer *DeleteGfd) GetProcedureFailProgress() int {
	return gcuPro.DELETE_GFD_FAILED
}

func (rer *DeleteGfd) GetProcedureAction() string {
	return consts.DELETE
}

func (rer *DeleteGfd) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeleteGfd) GetClient() client.Client {
	return rer.Client
}

func (rer *DeleteGfd) Execute() (err error) {
	// logic here
	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	if rer.GcuResource.Spec.Gfd == nil || (rer.GcuResource.Spec.Gfd.Deploy != nil && !*rer.GcuResource.Spec.Gfd.Deploy) {
		logs.Info("resource %s need not delete, because it not deployed", consts.GFD)
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:      consts.GFD,
		Namespace: base.GetNamespace(consts.GFD, rer.GcuResource.Spec.Gfd.Namespace),
		Status:    gcuStatus.DELETING,
	}
	resourceStatuses := rer.GcuResource.Status.ResourceStatuses
	for _, resourceStatus := range resourceStatuses {
		if resourceStatus.Name == consts.GFD {
			resourceInfo := resourceStatus.OwnResources
			for _, resInfo := range resourceInfo {
				namespaceName := k8sTypes.NamespacedName{Namespace: resourceStatus.Namespace, Name: resInfo.Name}
				if err := base.DeleteClusterResource(rer.Client, resInfo, &namespaceName); err != nil {
					return err
				}
				logs.Info("delete cluster resource(kind:%s, name:%s, namespace:%s) of %s success",
					resInfo.Kind, resInfo.Name, resourceStatus.Namespace, consts.GFD)
			}
		}
	}
	return rer.resetNodeLabels()
}

// It is difficult to obtain the config file of the specified node, only try to clear the default labels
func (rer *DeleteGfd) resetNodeLabels() error {
	nodeList := &corev1.NodeList{}
	if err := rer.Client.List(context.TODO(), nodeList); err != nil {
		logs.Error(err, "get cluster node list failed")
		return gcuErr.NewError(consts.ACTION_RETRY, err.Error())
	}
	for i := range nodeList.Items {
		node := &nodeList.Items[i]
		patch := client.MergeFrom(node.DeepCopy())
		hasGfdLabel := false
		for labelKey := range node.Labels {
			if strings.HasPrefix(labelKey, consts.EnflameLabelPrefix) && labelKey != consts.UnifiedDeviceLabelKey &&
				labelKey != consts.GcuSharePluginLabel {
				hasGfdLabel = true
				delete(node.Labels, labelKey)
			}
		}
		if !hasGfdLabel {
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
