# 燧原 GCU Operator (gcu-operator)

> **语言**: [English](README.md) | 中文

## 目录

- [概述](#概述)
- [项目结构](#项目结构)
- [安装](#安装)
  - [前置条件](#前置条件)
  - [从源码构建](#从源码构建)
- [快速开始](#快速开始)
  - [部署 Operator](#部署-operator)
  - [用 GcuResource CR 部署整套栈](#用-gcuresource-cr-部署整套栈)
- [核心特性](#核心特性)
- [被管理的组件](#被管理的组件)
- [版本兼容性](#版本兼容性)
- [硬件支持](#硬件支持)
- [故障排查](#故障排查)
- [许可证](#许可证)

## 概述

`gcu-operator` 是一个 Kubernetes [Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/),
通过一个自定义资源(Custom Resource)部署并管理整套 Enflame GCU(通用计算单元)
云原生栈。无需逐个手动安装组件,你只需 apply 一个 `GcuResource` 自定义资源,
operator 就会把整套栈——驱动、容器运行时、设备插件、特性发现、监控等——
调谐(reconcile)到期望状态。

它基于 [Operator SDK](https://sdk.operatorframework.io/) /
[Kubebuilder](https://book.kubebuilder.io/) 框架构建,定义了 CRD
`GcuResource`(组 `topscloud.enflame.com/v1`),充当 GCU 云原生组件的
**部署总管(orchestrator)**。

该 operator 提供:

- **一个自定义资源,拉起整套栈** —— 一个 `GcuResource` CR 驱动所有 GCU 云原生组件的部署与生命周期。
- **声明式 & 自愈** —— 控制器持续把被管理的组件调谐到 CR 中声明的状态。
- **组件编排** —— 协调式地安装并管理 GCU 驱动、容器工具包、设备插件、特性发现、各类 exporter 与共享组件。

## 项目结构

```
gcu-operator/
├── apis/                     # CRD 类型定义(topscloud/v1: GcuResource)
├── controllers/             # 调谐控制器
├── business-logic/          # 调谐 / 编排逻辑
├── components/              # 被管理组件的定义(driver、container-toolkit、cdi-manager …)
├── enflame-resources/       # 被管理组件的内置资源清单
├── config/                  # Operator 安装清单(CRD、RBAC、manager)
├── deployments/             # Operator 与组件镜像构建资源
├── model/  pkg/  hack/       # 共享模型、辅助函数、工具
├── main.go                  # Operator 入口
├── Makefile                 # Kubebuilder 风格构建目标
├── build.sh                 # 构建 + 打包入口
├── PROJECT                  # Kubebuilder 项目描述文件
└── go.mod                   # Go 模块定义
```

## 安装

### 前置条件

- 一个具备 cluster-admin 权限的 Kubernetes 集群(operator 会创建 CRD 与集群级 RBAC)。
- 带 Enflame GCU 设备的节点。
- 用于从源码构建的 [Go](https://go.dev/) 及可用工具链;构建镜像需 Docker。

### 从源码构建

```bash
# 构建 operator manager 二进制并打包到 dist/
./build.sh all

# 清理构建产物
./build.sh clean
```

`./build.sh all` 执行 `go build` 产出 `manager` 二进制,并连同 `deployments/`
与 `enflame-resources/` 一起打包到 `dist/gcu-operator_<版本>/`。容器镜像由
`deployments/` 下的脚本构建(`build-operator-image.sh`、`build-component-image.sh`)。

## 快速开始

### 部署 Operator

安装 CRD、RBAC 与 operator manager(清单在 `config/` / `deployments/` 下):

```bash
# 安装 CRD 并部署 operator(Kubebuilder 风格)
make install
make deploy
```

### 用 GcuResource CR 部署整套栈

创建一个 `GcuResource` 自定义资源,operator 会据此调谐出整套 GCU 栈:

```bash
kubectl apply -f config/samples/   # 示例 GcuResource CR

# 查看状态
kubectl get gcuresource -A
```

## 核心特性

| 特性 | 说明 |
| --- | --- |
| CRD 驱动 | 一个 `GcuResource` 自定义资源声明整套 GCU 栈 |
| 一键部署 | 安装并串联所有 GCU 云原生组件 |
| 自愈 | 持续把被管理组件调谐到声明状态 |
| Operator SDK / Kubebuilder | 标准 Kubernetes operator 模式(domain `enflame.com`) |
| 生命周期管理 | 处理被管理组件的安装、升级与卸载 |

## 被管理的组件

该 operator 编排 Enflame GCU 云原生组件,包括:GCU 驱动与 driver-manager、
容器工具包、设备插件、特性发现、gcu-exporter / node-exporter 监控组件、
node-feature-discovery,以及 gcushare 的 device-plugin / scheduler-plugin
共享组件(完整集合以你所用 operator 版本的 `components/` 与 `enflame-resources/`
为准)。

## 版本兼容性

operator 负责部署 GCU 云原生组件,因此需与一组兼容的组件版本、以及节点上部署的
Enflame GCU 驱动相匹配。请将其部署在支持该 CRD(`apiextensions.k8s.io/v1`)与
operator 模式的 Kubernetes 集群上。

## 硬件支持

管理面向 Enflame GCU 设备的软件栈。具体支持的设备取决于 GCU 驱动以及 operator
所部署的组件版本。

## 故障排查

- **`GcuResource` 不调谐** —— 查看 operator(manager)Pod 日志,确认 CRD 已安装
  (`kubectl get crd | grep gcuresource`)。
- **被管理组件未出现** —— 确认 operator 具备所需的集群 RBAC,且 CR 引用的组件镜像
  从集群可拉取。
- **apply 后节点仍无 GCU 资源** —— 确认节点已安装 GCU 驱动,并检查 operator 部署的
  各组件(设备插件、容器工具包等)状态。

## 许可证

`gcu-operator` 采用 **Apache License 2.0** 许可证。版权所有 (c) Enflame Technology,
保留所有权利。详见 [LICENSE](LICENSE)。
