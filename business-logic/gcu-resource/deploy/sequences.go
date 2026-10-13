// Copyright (c) 2022 Enflame. All Rights Reserved.

package deploy

import (
	v1 "gcu-operator/apis/topscloud/v1"
	"gcu-operator/business-logic/base/interfaces"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Procedures(gcuResource *v1.GcuResource, client client.Client) []interfaces.Procedure {
	return []interfaces.Procedure{
		NewDeployNfd(gcuResource, client),
		NewDeployGcuDriver(gcuResource, client),
		NewDeployGcuRuntime(gcuResource, client),
		NewDeployGcuK8sPlugin(gcuResource, client),
		NewDeployNodeExporter(gcuResource, client),
		NewDeployGcuExporter(gcuResource, client),
		NewDeployGcuFeatureDiscovery(gcuResource, client),
		NewDeployGcuSharePlugin(gcuResource, client),
		NewDeployGcuShareScheduler(gcuResource, client),
		NewDeployK8sDriverManager(gcuResource, client),
	}
}
