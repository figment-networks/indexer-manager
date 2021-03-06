// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.23.0
// 	protoc        v3.13.0
// source: proto/indexer.proto

package indexer

import (
	context "context"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type TaskError struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Msg  string `protobuf:"bytes,1,opt,name=msg,proto3" json:"msg,omitempty"`
	Type string `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
}

func (x *TaskError) Reset() {
	*x = TaskError{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_indexer_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TaskError) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TaskError) ProtoMessage() {}

func (x *TaskError) ProtoReflect() protoreflect.Message {
	mi := &file_proto_indexer_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TaskError.ProtoReflect.Descriptor instead.
func (*TaskError) Descriptor() ([]byte, []int) {
	return file_proto_indexer_proto_rawDescGZIP(), []int{0}
}

func (x *TaskError) GetMsg() string {
	if x != nil {
		return x.Msg
	}
	return ""
}

func (x *TaskError) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

type TaskRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Version string `protobuf:"bytes,1,opt,name=version,proto3" json:"version,omitempty"`
	Id      string `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	Type    string `protobuf:"bytes,3,opt,name=type,proto3" json:"type,omitempty"`
	From    string `protobuf:"bytes,4,opt,name=from,proto3" json:"from,omitempty"`
	Payload []byte `protobuf:"bytes,5,opt,name=payload,proto3" json:"payload,omitempty"`
}

func (x *TaskRequest) Reset() {
	*x = TaskRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_indexer_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TaskRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TaskRequest) ProtoMessage() {}

func (x *TaskRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_indexer_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TaskRequest.ProtoReflect.Descriptor instead.
func (*TaskRequest) Descriptor() ([]byte, []int) {
	return file_proto_indexer_proto_rawDescGZIP(), []int{1}
}

func (x *TaskRequest) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *TaskRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *TaskRequest) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *TaskRequest) GetFrom() string {
	if x != nil {
		return x.From
	}
	return ""
}

func (x *TaskRequest) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

type TaskResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Version string     `protobuf:"bytes,1,opt,name=version,proto3" json:"version,omitempty"`
	Id      string     `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	Type    string     `protobuf:"bytes,3,opt,name=type,proto3" json:"type,omitempty"`
	Order   int64      `protobuf:"varint,4,opt,name=order,proto3" json:"order,omitempty"`
	Final   bool       `protobuf:"varint,5,opt,name=final,proto3" json:"final,omitempty"`
	Error   *TaskError `protobuf:"bytes,6,opt,name=error,proto3" json:"error,omitempty"`
	Payload []byte     `protobuf:"bytes,7,opt,name=payload,proto3" json:"payload,omitempty"`
}

func (x *TaskResponse) Reset() {
	*x = TaskResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_indexer_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TaskResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TaskResponse) ProtoMessage() {}

func (x *TaskResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_indexer_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TaskResponse.ProtoReflect.Descriptor instead.
func (*TaskResponse) Descriptor() ([]byte, []int) {
	return file_proto_indexer_proto_rawDescGZIP(), []int{2}
}

func (x *TaskResponse) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *TaskResponse) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *TaskResponse) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *TaskResponse) GetOrder() int64 {
	if x != nil {
		return x.Order
	}
	return 0
}

func (x *TaskResponse) GetFinal() bool {
	if x != nil {
		return x.Final
	}
	return false
}

func (x *TaskResponse) GetError() *TaskError {
	if x != nil {
		return x.Error
	}
	return nil
}

func (x *TaskResponse) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

var File_proto_indexer_proto protoreflect.FileDescriptor

var file_proto_indexer_proto_rawDesc = []byte{
	0x0a, 0x13, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x65, 0x72, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x07, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x65, 0x72, 0x22, 0x31,
	0x0a, 0x09, 0x54, 0x61, 0x73, 0x6b, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x12, 0x10, 0x0a, 0x03, 0x6d,
	0x73, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6d, 0x73, 0x67, 0x12, 0x12, 0x0a,
	0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79, 0x70,
	0x65, 0x22, 0x79, 0x0a, 0x0b, 0x54, 0x61, 0x73, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x79,
	0x70, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x12,
	0x0a, 0x04, 0x66, 0x72, 0x6f, 0x6d, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x66, 0x72,
	0x6f, 0x6d, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18, 0x05, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x22, 0xbc, 0x01, 0x0a,
	0x0c, 0x54, 0x61, 0x73, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a,
	0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07,
	0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x6f,
	0x72, 0x64, 0x65, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52, 0x05, 0x6f, 0x72, 0x64, 0x65,
	0x72, 0x12, 0x14, 0x0a, 0x05, 0x66, 0x69, 0x6e, 0x61, 0x6c, 0x18, 0x05, 0x20, 0x01, 0x28, 0x08,
	0x52, 0x05, 0x66, 0x69, 0x6e, 0x61, 0x6c, 0x12, 0x28, 0x0a, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72,
	0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x65, 0x72,
	0x2e, 0x54, 0x61, 0x73, 0x6b, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6f,
	0x72, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18, 0x07, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x32, 0x4e, 0x0a, 0x0e, 0x49,
	0x6e, 0x64, 0x65, 0x78, 0x65, 0x72, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x3c, 0x0a,
	0x07, 0x54, 0x61, 0x73, 0x6b, 0x52, 0x50, 0x43, 0x12, 0x14, 0x2e, 0x69, 0x6e, 0x64, 0x65, 0x78,
	0x65, 0x72, 0x2e, 0x54, 0x61, 0x73, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x15,
	0x2e, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x65, 0x72, 0x2e, 0x54, 0x61, 0x73, 0x6b, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x28, 0x01, 0x30, 0x01, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_proto_indexer_proto_rawDescOnce sync.Once
	file_proto_indexer_proto_rawDescData = file_proto_indexer_proto_rawDesc
)

func file_proto_indexer_proto_rawDescGZIP() []byte {
	file_proto_indexer_proto_rawDescOnce.Do(func() {
		file_proto_indexer_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_indexer_proto_rawDescData)
	})
	return file_proto_indexer_proto_rawDescData
}

var file_proto_indexer_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_proto_indexer_proto_goTypes = []interface{}{
	(*TaskError)(nil),    // 0: indexer.TaskError
	(*TaskRequest)(nil),  // 1: indexer.TaskRequest
	(*TaskResponse)(nil), // 2: indexer.TaskResponse
}
var file_proto_indexer_proto_depIdxs = []int32{
	0, // 0: indexer.TaskResponse.error:type_name -> indexer.TaskError
	1, // 1: indexer.IndexerService.TaskRPC:input_type -> indexer.TaskRequest
	2, // 2: indexer.IndexerService.TaskRPC:output_type -> indexer.TaskResponse
	2, // [2:3] is the sub-list for method output_type
	1, // [1:2] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_proto_indexer_proto_init() }
func file_proto_indexer_proto_init() {
	if File_proto_indexer_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_proto_indexer_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TaskError); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_indexer_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TaskRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_indexer_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TaskResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proto_indexer_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_indexer_proto_goTypes,
		DependencyIndexes: file_proto_indexer_proto_depIdxs,
		MessageInfos:      file_proto_indexer_proto_msgTypes,
	}.Build()
	File_proto_indexer_proto = out.File
	file_proto_indexer_proto_rawDesc = nil
	file_proto_indexer_proto_goTypes = nil
	file_proto_indexer_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// IndexerServiceClient is the client API for IndexerService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type IndexerServiceClient interface {
	TaskRPC(ctx context.Context, opts ...grpc.CallOption) (IndexerService_TaskRPCClient, error)
}

type indexerServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewIndexerServiceClient(cc grpc.ClientConnInterface) IndexerServiceClient {
	return &indexerServiceClient{cc}
}

func (c *indexerServiceClient) TaskRPC(ctx context.Context, opts ...grpc.CallOption) (IndexerService_TaskRPCClient, error) {
	stream, err := c.cc.NewStream(ctx, &_IndexerService_serviceDesc.Streams[0], "/indexer.IndexerService/TaskRPC", opts...)
	if err != nil {
		return nil, err
	}
	x := &indexerServiceTaskRPCClient{stream}
	return x, nil
}

type IndexerService_TaskRPCClient interface {
	Send(*TaskRequest) error
	Recv() (*TaskResponse, error)
	grpc.ClientStream
}

type indexerServiceTaskRPCClient struct {
	grpc.ClientStream
}

func (x *indexerServiceTaskRPCClient) Send(m *TaskRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *indexerServiceTaskRPCClient) Recv() (*TaskResponse, error) {
	m := new(TaskResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// IndexerServiceServer is the server API for IndexerService service.
type IndexerServiceServer interface {
	TaskRPC(IndexerService_TaskRPCServer) error
}

// UnimplementedIndexerServiceServer can be embedded to have forward compatible implementations.
type UnimplementedIndexerServiceServer struct {
}

func (*UnimplementedIndexerServiceServer) TaskRPC(IndexerService_TaskRPCServer) error {
	return status.Errorf(codes.Unimplemented, "method TaskRPC not implemented")
}

func RegisterIndexerServiceServer(s *grpc.Server, srv IndexerServiceServer) {
	s.RegisterService(&_IndexerService_serviceDesc, srv)
}

func _IndexerService_TaskRPC_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(IndexerServiceServer).TaskRPC(&indexerServiceTaskRPCServer{stream})
}

type IndexerService_TaskRPCServer interface {
	Send(*TaskResponse) error
	Recv() (*TaskRequest, error)
	grpc.ServerStream
}

type indexerServiceTaskRPCServer struct {
	grpc.ServerStream
}

func (x *indexerServiceTaskRPCServer) Send(m *TaskResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *indexerServiceTaskRPCServer) Recv() (*TaskRequest, error) {
	m := new(TaskRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _IndexerService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "indexer.IndexerService",
	HandlerType: (*IndexerServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "TaskRPC",
			Handler:       _IndexerService_TaskRPC_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "proto/indexer.proto",
}
