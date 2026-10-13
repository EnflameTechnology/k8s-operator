// Copyright (c) 2022 Enflame. All Rights Reserved.

package interfaces

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "gcu-operator/apis/topscloud/v1"
)

type Procedure interface {
	GetProcedureName() string
	GetProcedureAction() string
	GetProcedureStatus() []string
	GetProcedureOKStatus() string
	GetProcedureFailStatus() string
	GetProcedureOKProgress() int
	GetProcedureFailProgress() int
	GetGCUResource() *v1.GcuResource
	GetClient() client.Client
	Execute() error
}
