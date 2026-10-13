// Copyright (c) 2022 Enflame. All Rights Reserved.

package status

const (
	// gcu resource err_code
	DELETING = "deleting"

	DELETE_GFD_SUCCESS = "delete gfd success"
	DELETE_GFD_FAILED  = "delete gfd failed"

	DELETE_GCU_EXPORTER_SUCCESS = "delete gcu exporter success"
	DELETE_GCU_EXPORTER_FAILED  = "delete gcu exporter failed"

	DELETE_NODE_EXPORTER_SUCCESS = "delete node exporter success"
	DELETE_NODE_EXPORTER_FAILED  = "delete node exporter failed"

	DELETE_DEVICE_PLUGIN_SUCCESS = "delete device plugin success"
	DELETE_DEVICE_PLUGIN_FAILED  = "delete device plugin failed"

	DELETE_TOOLKIT_SUCCESS = "delete toolkit success"
	DELETE_TOOLKIT_FAILED  = "delete toolkit failed"

	DELETE_DRIVER_SUCCESS = "delete driver success"
	DELETE_DRIVER_FAILED  = "delete driver failed"

	DELETE_NFD_SUCCESS = "delete nfd success"
	DELETE_NFD_FAILED  = "delete nfd failed"

	DELETE_GCUSHARE_PLUGIN_SUCCESS = "delete gcushare plugin success"
	DELETE_GCUSHARE_PLUGIN_FAILED  = "delete gcushare plugin failed"

	DELETE_GCUSHARE_SCHEDULER_SUCCESS = "delete gcushare scheduler success"
	DELETE_GCUSHARE_SCHEDULER_FAILED  = "delete gcushare scheduler failed"

	DELETE_K8S_DRIVER_MANAGER_SUCCESS = "delete k8s driver manager success"
	DELETE_K8S_DRIVER_MANAGER_FAILED  = "delete k8s driver manager failed"

	CHECK_DELETE_STATUS_FINISH = "check delete all resource finish"
	CHECK_DELETE_STATUS_FAILED = "check delete resource failed"

	// sub resource status
	SUB_RESOURCE_DELETING = "deleting"
	SUB_RESOURCE_DELETED  = "deleted"
)
