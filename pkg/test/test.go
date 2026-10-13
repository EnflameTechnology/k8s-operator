// Copyright (c) 2022 Enflame. All Rights Reserved.

package test

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MockProcedure struct {
}

func (rer *MockProcedure) GetProcedureName() string {
	return ""
}

func (rer *MockProcedure) GetProcedureStatus() []string {
	return []string{}
}

func (rer *MockProcedure) GetProcedureOKStatus() string {
	return ""
}

func (rer *MockProcedure) GetProcedureFailStatus() string {
	return ""
}

func (rer *MockProcedure) GetProcedureOKProgress() int {
	return 0
}

func (rer *MockProcedure) GetProcedureFailProgress() int {
	return 0
}

func (rer *MockProcedure) Execute() error {
	return nil
}

type MockManagerClient struct{}

func (rer *MockManagerClient) Scheme() *runtime.Scheme {
	return nil
}

func (rer *MockManagerClient) RESTMapper() meta.RESTMapper {
	return nil
}

func (rer *MockManagerClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return nil
}

func (rer *MockManagerClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return nil
}

func (rer *MockManagerClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return nil
}

func (rer *MockManagerClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return nil
}

func (rer *MockManagerClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return nil
}

func (rer *MockManagerClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}

func (rer *MockManagerClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return nil
}

func (rer *MockManagerClient) Status() client.StatusWriter {
	return &MockStatusWriter{}
}

type MockStatusWriter struct{}

func (rer *MockStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return nil
}

func (rer *MockStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}
