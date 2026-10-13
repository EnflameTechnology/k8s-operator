// Copyright (c) 2022 Enflame. All Rights Reserved.

package delete

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	v1 "gcu-operator/apis/topscloud/v1"
	gcuStatus "gcu-operator/business-logic/gcu-resource/base/status"
	"gcu-operator/pkg/consts"
	"gcu-operator/pkg/object"
	"gcu-operator/pkg/test"
)

func TestCheckGcuResourceHasDeleted(t *testing.T) {
	Convey("test func /business-logic/gcu-resource/delete/check_delete_status.go/updateResourceDeleteStatus", t, func() {
		gcuResource := &v1.GcuResource{}
		client := &test.MockManagerClient{}
		checkIns := NewCheckDeleteStatus(gcuResource, client).(*CheckDeleteStatus)
		Convey("1. test updateResourceDeleteStatus success", func() {
			resourceStatus := &[]v1.ResourceStatus{
				{
					Name:   consts.Driver,
					Status: gcuStatus.SUB_RESOURCE_DELETED,
					OwnResources: []v1.ResourceInfo{
						{
							Group:   "",
							Version: "v1",
							Kind:    object.Pod,
							Name:    "zxx",
						},
					},
				},
			}
			checkIns.GcuResource.Status.ResourceStatuses = []v1.ResourceStatus{
				{
					Name:      consts.Driver,
					Namespace: "kube-system",
					Status:    gcuStatus.DELETING,
				},
				{
					Name:      consts.Plugin,
					Namespace: "kube-system",
					Status:    gcuStatus.DELETING,
				},
			}
			err := fmt.Errorf("aaa")
			checkIns.updateResourceDeleteStatus(resourceStatus, &err)
			So(checkIns.GcuResource.Status.ResourceStatuses[0].Name, ShouldEqual, consts.Driver)
			So(checkIns.GcuResource.Status.ResourceStatuses[0].Namespace, ShouldEqual, "kube-system")
			So(checkIns.GcuResource.Status.ResourceStatuses[0].Status, ShouldEqual, gcuStatus.SUB_RESOURCE_DELETED)
			So(len(checkIns.GcuResource.Status.ResourceStatuses[0].OwnResources), ShouldEqual, 1)
			So(checkIns.GcuResource.Status.ResourceStatuses[0].OwnResources[0].Name, ShouldEqual, "zxx")

			So(checkIns.GcuResource.Status.ResourceStatuses[1].Name, ShouldEqual, consts.Plugin)
			So(checkIns.GcuResource.Status.ResourceStatuses[1].Namespace, ShouldEqual, "kube-system")
			So(checkIns.GcuResource.Status.ResourceStatuses[1].Status, ShouldEqual, gcuStatus.DELETING)
		})
	})
}
