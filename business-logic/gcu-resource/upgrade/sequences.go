// Copyright (c) 2022 Enflame. All Rights Reserved.

package upgrade

import (
	v1 "gcu-operator/apis/topscloud/v1"
	"gcu-operator/business-logic/base/interfaces"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Procedures(gcuResource *v1.GcuResource, client client.Client) []interfaces.Procedure {
	return []interfaces.Procedure{}
}
