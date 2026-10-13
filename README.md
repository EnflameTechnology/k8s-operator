# Enflame GCU Operator (gcu-operator)

> **Language**: English | [中文](README_zh.md)

## Table of Contents

- [Overview](#overview)
- [Project Structure](#project-structure)
- [Installation](#installation)
  - [Prerequisites](#prerequisites)
  - [Build from Source](#build-from-source)
- [Quick Start](#quick-start)
  - [Deploy the Operator](#deploy-the-operator)
  - [Deploy the Stack with a GcuResource CR](#deploy-the-stack-with-a-gcuresource-cr)
- [Key Features](#key-features)
- [Managed Components](#managed-components)
- [Version Compatibility](#version-compatibility)
- [Hardware Support](#hardware-support)
- [Trouble Shooting](#trouble-shooting)
- [License](#license)

## Overview

`gcu-operator` is a Kubernetes [Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
that deploys and manages the full Enflame GCU (General Compute Unit)
cloud-native stack through a single Custom Resource. Instead of installing each
component by hand, you apply one `GcuResource` custom resource and the operator
reconciles the whole stack — driver, container runtime, device plugin, feature
discovery, monitoring and more — to the desired state.

It is built with the [Operator SDK](https://sdk.operatorframework.io/) /
[Kubebuilder](https://book.kubebuilder.io/) framework, defines the CRD
`GcuResource` (group `topscloud.enflame.com/v1`), and acts as the **deployment
orchestrator** for the GCU cloud-native components.

The operator delivers:

- **One custom resource, whole stack** — a single `GcuResource` CR drives the deployment and lifecycle of every GCU cloud-native component.
- **Declarative & self-healing** — the controller continuously reconciles the managed components to the state declared in the CR.
- **Component orchestration** — installs and manages the GCU driver, container toolkit, device plugin, feature discovery, exporters and sharing components in a coordinated way.

## Project Structure

```
gcu-operator/
├── apis/                     # CRD type definitions (topscloud/v1: GcuResource)
├── controllers/             # Reconcile controllers
├── business-logic/          # Reconciliation / orchestration logic
├── components/              # Managed component definitions (driver, container-toolkit, cdi-manager, ...)
├── enflame-resources/       # Bundled resource manifests for the managed components
├── config/                  # Operator install manifests (CRD, RBAC, manager)
├── deployments/             # Operator & component image build assets
├── model/  pkg/  hack/       # Shared models, helpers, tooling
├── main.go                  # Operator entry point
├── Makefile                 # Kubebuilder-style build targets
├── build.sh                 # Build + package entry point
├── PROJECT                  # Kubebuilder project descriptor
└── go.mod                   # Go module definition
```

## Installation

### Prerequisites

- A Kubernetes cluster with cluster-admin access (the operator creates CRDs and cluster-scoped RBAC).
- Nodes with Enflame GCU devices.
- [Go](https://go.dev/) and a working toolchain for building from source; Docker for building images.

### Build from Source

```bash
# Build the operator manager binary and produce a release package under dist/
./build.sh all

# Clean build artifacts
./build.sh clean
```

`./build.sh all` runs `go build` to produce the `manager` binary and packages
it together with `deployments/` and `enflame-resources/` under
`dist/gcu-operator_<version>/`. Container images are built via the scripts under
`deployments/` (`build-operator-image.sh`, `build-component-image.sh`).

## Quick Start

### Deploy the Operator

Install the CRD, RBAC and the operator manager (manifests under `config/` /
`deployments/`):

```bash
# Install CRDs and deploy the operator (Kubebuilder-style)
make install
make deploy
```

### Deploy the Stack with a GcuResource CR

Create a `GcuResource` custom resource; the operator reconciles the full GCU
stack according to it:

```bash
kubectl apply -f config/samples/   # sample GcuResource CR

# Inspect status
kubectl get gcuresource -A
```

## Key Features

| Feature | Description |
| --- | --- |
| CRD-driven | A single `GcuResource` custom resource declares the whole GCU stack |
| One-click deploy | Installs and wires together all GCU cloud-native components |
| Self-healing | Continuously reconciles managed components to the declared state |
| Operator SDK / Kubebuilder | Standard Kubernetes operator pattern (domain `enflame.com`) |
| Lifecycle management | Handles install, upgrade and teardown of the managed components |

## Managed Components

The operator orchestrates the Enflame GCU cloud-native components, including the
GCU driver and driver-manager, the container toolkit, the device plugin,
feature discovery, the gcu-exporter / node-exporter monitoring components,
node-feature-discovery, and the gcushare device-plugin / scheduler-plugin
sharing components (see `components/` and `enflame-resources/` for the full set
managed by your operator version).

## Version Compatibility

The operator deploys the GCU cloud-native components, so it must be matched to a
compatible set of component versions and to the Enflame GCU driver deployed on
your nodes. Deploy it on a Kubernetes cluster whose version supports the CRD
(`apiextensions.k8s.io/v1`) and operator pattern.

## Hardware Support

Manages the software stack for Enflame GCU devices. The exact devices supported
depend on the GCU driver and the component versions the operator deploys.

## Trouble Shooting

- **`GcuResource` not reconciling** — check the operator (manager) pod logs and
  confirm the CRD is installed (`kubectl get crd | grep gcuresource`).
- **Managed components not appearing** — verify the operator has the required
  cluster RBAC, and that the referenced component images are reachable from the
  cluster.
- **Nodes still missing GCU resources after apply** — confirm the GCU driver is
  installed on the nodes and inspect the status of the components the operator
  deployed (device plugin, container toolkit, etc.).

## License

`gcu-operator` is licensed under the **Apache License 2.0**. Copyright (c)
Enflame Technology. All Rights Reserved. See [LICENSE](LICENSE).
