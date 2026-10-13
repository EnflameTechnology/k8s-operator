// Copyright (c) 2022 Enflame. All Rights Reserved.

package status

const (
	// gcu resource err_code
	DEPLOYING = "deploying"

	DEPLOY_NFD_SUCCESS = "deploy nfd success"
	DEPLOY_NFD_FAILED  = "deploy nfd failed"

	DEPLOY_DRIVER_SUCCESS = "deploy driver success"
	DEPLOY_DRIVER_FAILED  = "deploy driver failed"

	DEPLOY_TOOLKIT_SUCCESS = "deploy toolkit success"
	DEPLOY_TOOLKIT_FAILED  = "deploy toolkit failed"

	DEPLOY_DEVICE_PLUGIN_SUCCESS = "deploy device plugin success"
	DEPLOY_DEVICE_PLUGIN_FAILED  = "deploy device plugin failed"

	DEPLOY_GCU_EXPORTER_SUCCESS = "deploy gcu exporter success"
	DEPLOY_GCU_EXPORTER_FAILED  = "deploy gcu exporter failed"

	DEPLOY_NODE_EXPORTER_SUCCESS = "deploy node exporter success"
	DEPLOY_NODE_EXPORTER_FAILED  = "deploy node exporter failed"

	DEPLOY_GFD_SUCCESS = "deploy gfd success"
	DEPLOY_GFD_FAILED  = "deploy gfd failed"

	DEPLOY_GCUSHARE_PLUGIN_SUCCESS = "deploy gcushare plugin success"
	DEPLOY_GCUSHARE_PLUGIN_FAILED  = "deploy gcushare plugin failed"

	DEPLOY_GCUSHARE_SCHEDULER_SUCCESS = "deploy gcushare scheduler success"
	DEPLOY_GCUSHARE_SCHEDULER_FAILED  = "deploy gcushare scheduler failed"

	DEPLOY_K8S_DRIVER_MANAGER_SUCCESS = "deploy k8s driver manager success"
	DEPLOY_K8S_DRIVER_MANAGER_FAILED  = "deploy k8s driver manager failed"

	// sub resource status
	SUB_RESOURCE_DEPLOYING = "deploying"
)
