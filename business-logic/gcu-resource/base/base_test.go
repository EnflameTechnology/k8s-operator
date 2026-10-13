// Copyright (c) 2022 Enflame. All Rights Reserved.

package base

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCheckGcuResourceHasDeleted(t *testing.T) {
	Convey("test func /business-logic/gcu-resource/base.go/checkGcuResourceHasDeleted", t, func() {
		Convey("1. test cluster gcu resource has been deleted", func() {
			// err := new(error)
			// action := consts.DELETE
			// rer := new(test.MockProcedure)
			// rv := checkGcuResourceHasDeleted(err, action, rer)
			// So(rv, ShouldEqual, false)
		})

		Convey("2. test cluster gcu resource exist", func() {
			// rer := new(test.MockProcedure)
			// patches := gomonkey.ApplyMethod(reflect.TypeOf(rer), "GetProcedureOKStatus", func(_ *test.MockProcedure) string {
			// 	return gcuStatus.CHECK_DELETE_STATUS_FINISH
			// })
			// defer patches.Reset()
			// err := new(error)
			// action := consts.DELETE
			// rv := checkGcuResourceHasDeleted(err, action, rer)
			// So(rv, ShouldEqual, true)
		})
	})
}
