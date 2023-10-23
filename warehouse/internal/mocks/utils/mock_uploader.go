// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/rudderlabs/rudder-server/warehouse/utils (interfaces: Uploader)

// Package mock_uploader is a generated GoMock package.
package mock_uploader

import (
	context "context"
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"
	model "github.com/rudderlabs/rudder-server/warehouse/internal/model"
	warehouseutils "github.com/rudderlabs/rudder-server/warehouse/utils"
)

// MockUploader is a mock of Uploader interface.
type MockUploader struct {
	ctrl     *gomock.Controller
	recorder *MockUploaderMockRecorder
}

// MockUploaderMockRecorder is the mock recorder for MockUploader.
type MockUploaderMockRecorder struct {
	mock *MockUploader
}

// NewMockUploader creates a new mock instance.
func NewMockUploader(ctrl *gomock.Controller) *MockUploader {
	mock := &MockUploader{ctrl: ctrl}
	mock.recorder = &MockUploaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUploader) EXPECT() *MockUploaderMockRecorder {
	return m.recorder
}

// CanAppend mocks base method.
func (m *MockUploader) CanAppend() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CanAppend")
	ret0, _ := ret[0].(bool)
	return ret0
}

// CanAppend indicates an expected call of CanAppend.
func (mr *MockUploaderMockRecorder) CanAppend() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CanAppend", reflect.TypeOf((*MockUploader)(nil).CanAppend))
}

// GetFirstLastEvent mocks base method.
func (m *MockUploader) GetFirstLastEvent() (time.Time, time.Time) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFirstLastEvent")
	ret0, _ := ret[0].(time.Time)
	ret1, _ := ret[1].(time.Time)
	return ret0, ret1
}

// GetFirstLastEvent indicates an expected call of GetFirstLastEvent.
func (mr *MockUploaderMockRecorder) GetFirstLastEvent() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFirstLastEvent", reflect.TypeOf((*MockUploader)(nil).GetFirstLastEvent))
}

// GetLoadFileGenStartTIme mocks base method.
func (m *MockUploader) GetLoadFileGenStartTIme() time.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLoadFileGenStartTIme")
	ret0, _ := ret[0].(time.Time)
	return ret0
}

// GetLoadFileGenStartTIme indicates an expected call of GetLoadFileGenStartTIme.
func (mr *MockUploaderMockRecorder) GetLoadFileGenStartTIme() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLoadFileGenStartTIme", reflect.TypeOf((*MockUploader)(nil).GetLoadFileGenStartTIme))
}

// GetLoadFileType mocks base method.
func (m *MockUploader) GetLoadFileType() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLoadFileType")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetLoadFileType indicates an expected call of GetLoadFileType.
func (mr *MockUploaderMockRecorder) GetLoadFileType() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLoadFileType", reflect.TypeOf((*MockUploader)(nil).GetLoadFileType))
}

// GetLoadFilesMetadata mocks base method.
func (m *MockUploader) GetLoadFilesMetadata(arg0 context.Context, arg1 warehouseutils.GetLoadFilesOptions) ([]warehouseutils.LoadFile, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLoadFilesMetadata", arg0, arg1)
	ret0, _ := ret[0].([]warehouseutils.LoadFile)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLoadFilesMetadata indicates an expected call of GetLoadFilesMetadata.
func (mr *MockUploaderMockRecorder) GetLoadFilesMetadata(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLoadFilesMetadata", reflect.TypeOf((*MockUploader)(nil).GetLoadFilesMetadata), arg0, arg1)
}

// GetLocalSchema mocks base method.
func (m *MockUploader) GetLocalSchema(arg0 context.Context) (model.Schema, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLocalSchema", arg0)
	ret0, _ := ret[0].(model.Schema)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLocalSchema indicates an expected call of GetLocalSchema.
func (mr *MockUploaderMockRecorder) GetLocalSchema(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLocalSchema", reflect.TypeOf((*MockUploader)(nil).GetLocalSchema), arg0)
}

// GetSampleLoadFileLocation mocks base method.
func (m *MockUploader) GetSampleLoadFileLocation(arg0 context.Context, arg1 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSampleLoadFileLocation", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSampleLoadFileLocation indicates an expected call of GetSampleLoadFileLocation.
func (mr *MockUploaderMockRecorder) GetSampleLoadFileLocation(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSampleLoadFileLocation", reflect.TypeOf((*MockUploader)(nil).GetSampleLoadFileLocation), arg0, arg1)
}

// GetSingleLoadFile mocks base method.
func (m *MockUploader) GetSingleLoadFile(arg0 context.Context, arg1 string) (warehouseutils.LoadFile, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSingleLoadFile", arg0, arg1)
	ret0, _ := ret[0].(warehouseutils.LoadFile)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSingleLoadFile indicates an expected call of GetSingleLoadFile.
func (mr *MockUploaderMockRecorder) GetSingleLoadFile(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSingleLoadFile", reflect.TypeOf((*MockUploader)(nil).GetSingleLoadFile), arg0, arg1)
}

// GetTableSchemaInUpload mocks base method.
func (m *MockUploader) GetTableSchemaInUpload(arg0 string) model.TableSchema {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTableSchemaInUpload", arg0)
	ret0, _ := ret[0].(model.TableSchema)
	return ret0
}

// GetTableSchemaInUpload indicates an expected call of GetTableSchemaInUpload.
func (mr *MockUploaderMockRecorder) GetTableSchemaInUpload(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTableSchemaInUpload", reflect.TypeOf((*MockUploader)(nil).GetTableSchemaInUpload), arg0)
}

// GetTableSchemaInWarehouse mocks base method.
func (m *MockUploader) GetTableSchemaInWarehouse(arg0 string) model.TableSchema {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTableSchemaInWarehouse", arg0)
	ret0, _ := ret[0].(model.TableSchema)
	return ret0
}

// GetTableSchemaInWarehouse indicates an expected call of GetTableSchemaInWarehouse.
func (mr *MockUploaderMockRecorder) GetTableSchemaInWarehouse(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTableSchemaInWarehouse", reflect.TypeOf((*MockUploader)(nil).GetTableSchemaInWarehouse), arg0)
}

// IsWarehouseSchemaEmpty mocks base method.
func (m *MockUploader) IsWarehouseSchemaEmpty() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsWarehouseSchemaEmpty")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsWarehouseSchemaEmpty indicates an expected call of IsWarehouseSchemaEmpty.
func (mr *MockUploaderMockRecorder) IsWarehouseSchemaEmpty() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsWarehouseSchemaEmpty", reflect.TypeOf((*MockUploader)(nil).IsWarehouseSchemaEmpty))
}

// ShouldOnDedupUseNewRecord mocks base method.
func (m *MockUploader) ShouldOnDedupUseNewRecord() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ShouldOnDedupUseNewRecord")
	ret0, _ := ret[0].(bool)
	return ret0
}

// ShouldOnDedupUseNewRecord indicates an expected call of ShouldOnDedupUseNewRecord.
func (mr *MockUploaderMockRecorder) ShouldOnDedupUseNewRecord() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ShouldOnDedupUseNewRecord", reflect.TypeOf((*MockUploader)(nil).ShouldOnDedupUseNewRecord))
}

// UpdateLocalSchema mocks base method.
func (m *MockUploader) UpdateLocalSchema(arg0 context.Context, arg1 model.Schema) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateLocalSchema", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateLocalSchema indicates an expected call of UpdateLocalSchema.
func (mr *MockUploaderMockRecorder) UpdateLocalSchema(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateLocalSchema", reflect.TypeOf((*MockUploader)(nil).UpdateLocalSchema), arg0, arg1)
}

// UseRudderStorage mocks base method.
func (m *MockUploader) UseRudderStorage() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UseRudderStorage")
	ret0, _ := ret[0].(bool)
	return ret0
}

// UseRudderStorage indicates an expected call of UseRudderStorage.
func (mr *MockUploaderMockRecorder) UseRudderStorage() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UseRudderStorage", reflect.TypeOf((*MockUploader)(nil).UseRudderStorage))
}
