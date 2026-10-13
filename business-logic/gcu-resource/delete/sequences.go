// Copyright (c) 2022 Enflame. All Rights Reserved.

package delete

import (
	v1 "gcu-operator/apis/topscloud/v1"
	"gcu-operator/business-logic/base/interfaces"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Procedures(gcuResource *v1.GcuResource, client client.Client) []interfaces.Procedure {
	return []interfaces.Procedure{
		NewDeleteK8sDriverManager(gcuResource, client),
		NewDeleteGcuShareScheduler(gcuResource, client),
		NewDeleteGcuSharePlugin(gcuResource, client),
		NewDeleteGfd(gcuResource, client),
		NewDeleteGcuExporter(gcuResource, client),
		NewDeleteNodeExporter(gcuResource, client),
		NewDeleteDevicePlugin(gcuResource, client),
		NewDeleteToolkit(gcuResource, client),
		NewDeleteDriver(gcuResource, client),
		NewDeleteNfd(gcuResource, client),
		NewCheckDeleteStatus(gcuResource, client),
	}
}
