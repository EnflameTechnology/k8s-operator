// Copyright (c) 2022 Enflame. All Rights Reserved.

package executor

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	gcuv1 "gcu-operator/apis/topscloud/v1"
	"gcu-operator/business-logic/base/interfaces"
	"gcu-operator/business-logic/gcu-resource/delete"
	"gcu-operator/business-logic/gcu-resource/deploy"
	"gcu-operator/business-logic/gcu-resource/rollback"
	"gcu-operator/business-logic/gcu-resource/upgrade"
	"gcu-operator/pkg/common"
	"gcu-operator/pkg/consts"
	gcuErr "gcu-operator/pkg/error"
	"gcu-operator/pkg/logs"
)

type Executor struct {
	Procedures  []interfaces.Procedure
	GcuResource *gcuv1.GcuResource
	Ctx         context.Context
	Client      client.Client
	Action      string
}

func NewExecutor(gcuResource *gcuv1.GcuResource, ctx context.Context, client client.Client,
	action string) *Executor {
	return &Executor{
		GcuResource: gcuResource,
		Ctx:         ctx,
		Client:      client,
		Action:      action,
	}
}

func (rer *Executor) Execute() error {
	rer.initProcedures()
	condition := common.FindConditionByActionType(rer.Action, rer.GcuResource.Status.Conditions)
	if condition.Action == "" {
		msg := fmt.Sprintf("can not find condition by action:%s", rer.Action)
		return gcuErr.NewError(consts.ACTION_FAIL, msg)
	}
	status := condition.Status
	if rer.needNotExecuteProcedures(status) {
		return nil
	}
	getPro := false
	for index := 0; index < len(rer.Procedures); index++ {
		if !getPro && rer.statusInProcedure(status, rer.Procedures[index]) {
			logs.Info("start %s gcuResource(name:%s, uuid:%s), status:%s", rer.Action,
				rer.GcuResource.Name, rer.GcuResource.UID, status)
			getPro = true
		}
		if getPro {
			if err := rer.Procedures[index].Execute(); err != nil {
				if gcuErr, ok := err.(*gcuErr.Error); ok && gcuErr.Code == consts.ACTION_RETRY {
					logs.Warn("execute procedure: %s failed, status: %s, error: %s", rer.Procedures[index].GetProcedureName(),
						rer.Procedures[index].GetProcedureFailStatus(), gcuErr.Error())
				} else {
					logs.Error(err, "execute procedure: %s failed, status: %s", rer.Procedures[index].GetProcedureName(),
						rer.Procedures[index].GetProcedureFailStatus())
				}
				return err
			}
		}
	}
	if !getPro {
		// if status == gcuStatus.SUCCESS {
		// 	return nil
		// }
		msg := fmt.Sprintf("can not find procedure by action:%s, status:%s", rer.Action, status)
		logs.Error(fmt.Errorf(msg), "gcuResource(name:%s, uuid:%s)", rer.GcuResource.Name, rer.GcuResource.UID)
		return gcuErr.NewError(consts.ACTION_FAIL, msg)
	}
	logs.Info("all procedures of gcuResource(name:%s, uuid:%s) execute success",
		rer.GcuResource.Name, rer.GcuResource.UID)
	return nil
}

func (rer *Executor) statusInProcedure(status string, procedure interfaces.Procedure) bool {
	for _, proStatus := range procedure.GetProcedureStatus() {
		if proStatus == status {
			return true
		}
	}
	return false
}

func (rer *Executor) initProcedures() {
	if rer.Action == consts.DEPLOY {
		rer.Procedures = deploy.Procedures(rer.GcuResource, rer.Client)
	} else if rer.Action == consts.DELETE {
		rer.Procedures = delete.Procedures(rer.GcuResource, rer.Client)
	} else if rer.Action == consts.UPGRADE {
		rer.Procedures = upgrade.Procedures(rer.GcuResource, rer.Client)
	} else {
		rer.Procedures = rollback.Procedures(rer.GcuResource, rer.Client)
	}
}

func (rer *Executor) needNotExecuteProcedures(status string) bool {
	// status is last progress success value, exit procedure
	// This branch is usually not reachable, but client-go has a delay and there is a possibility of entry
	if status == rer.Procedures[len(rer.Procedures)-1].GetProcedureOKStatus() {
		return true
	}
	return false
}
