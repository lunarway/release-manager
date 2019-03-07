// Code generated by protoc-gen-go. DO NOT EDIT.
// source: protos/releasemanager.proto

/*
Package grpc is a generated protocol buffer package.

It is generated from these files:
	protos/releasemanager.proto

It has these top-level messages:
	PromoteRequest
	PromoteResponse
	StatusRequest
	Environment
	StatusResponse
*/
package grpc

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc1 "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type PromoteRequest struct {
	Service     string `protobuf:"bytes,1,opt,name=service" json:"service,omitempty"`
	Environment string `protobuf:"bytes,2,opt,name=environment" json:"environment,omitempty"`
}

func (m *PromoteRequest) Reset()                    { *m = PromoteRequest{} }
func (m *PromoteRequest) String() string            { return proto.CompactTextString(m) }
func (*PromoteRequest) ProtoMessage()               {}
func (*PromoteRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *PromoteRequest) GetService() string {
	if m != nil {
		return m.Service
	}
	return ""
}

func (m *PromoteRequest) GetEnvironment() string {
	if m != nil {
		return m.Environment
	}
	return ""
}

type PromoteResponse struct {
	Service     string `protobuf:"bytes,1,opt,name=service" json:"service,omitempty"`
	Environment string `protobuf:"bytes,2,opt,name=environment" json:"environment,omitempty"`
	Status      string `protobuf:"bytes,3,opt,name=status" json:"status,omitempty"`
}

func (m *PromoteResponse) Reset()                    { *m = PromoteResponse{} }
func (m *PromoteResponse) String() string            { return proto.CompactTextString(m) }
func (*PromoteResponse) ProtoMessage()               {}
func (*PromoteResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *PromoteResponse) GetService() string {
	if m != nil {
		return m.Service
	}
	return ""
}

func (m *PromoteResponse) GetEnvironment() string {
	if m != nil {
		return m.Environment
	}
	return ""
}

func (m *PromoteResponse) GetStatus() string {
	if m != nil {
		return m.Status
	}
	return ""
}

type StatusRequest struct {
	Service string `protobuf:"bytes,1,opt,name=service" json:"service,omitempty"`
}

func (m *StatusRequest) Reset()                    { *m = StatusRequest{} }
func (m *StatusRequest) String() string            { return proto.CompactTextString(m) }
func (*StatusRequest) ProtoMessage()               {}
func (*StatusRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *StatusRequest) GetService() string {
	if m != nil {
		return m.Service
	}
	return ""
}

type Environment struct {
	Tag       string `protobuf:"bytes,1,opt,name=tag" json:"tag,omitempty"`
	Committer string `protobuf:"bytes,2,opt,name=committer" json:"committer,omitempty"`
	Author    string `protobuf:"bytes,3,opt,name=author" json:"author,omitempty"`
	Message   string `protobuf:"bytes,4,opt,name=message" json:"message,omitempty"`
	Date      int64  `protobuf:"varint,5,opt,name=date" json:"date,omitempty"`
}

func (m *Environment) Reset()                    { *m = Environment{} }
func (m *Environment) String() string            { return proto.CompactTextString(m) }
func (*Environment) ProtoMessage()               {}
func (*Environment) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *Environment) GetTag() string {
	if m != nil {
		return m.Tag
	}
	return ""
}

func (m *Environment) GetCommitter() string {
	if m != nil {
		return m.Committer
	}
	return ""
}

func (m *Environment) GetAuthor() string {
	if m != nil {
		return m.Author
	}
	return ""
}

func (m *Environment) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func (m *Environment) GetDate() int64 {
	if m != nil {
		return m.Date
	}
	return 0
}

type StatusResponse struct {
	Dev     *Environment `protobuf:"bytes,1,opt,name=dev" json:"dev,omitempty"`
	Staging *Environment `protobuf:"bytes,2,opt,name=staging" json:"staging,omitempty"`
	Prod    *Environment `protobuf:"bytes,3,opt,name=prod" json:"prod,omitempty"`
}

func (m *StatusResponse) Reset()                    { *m = StatusResponse{} }
func (m *StatusResponse) String() string            { return proto.CompactTextString(m) }
func (*StatusResponse) ProtoMessage()               {}
func (*StatusResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *StatusResponse) GetDev() *Environment {
	if m != nil {
		return m.Dev
	}
	return nil
}

func (m *StatusResponse) GetStaging() *Environment {
	if m != nil {
		return m.Staging
	}
	return nil
}

func (m *StatusResponse) GetProd() *Environment {
	if m != nil {
		return m.Prod
	}
	return nil
}

func init() {
	proto.RegisterType((*PromoteRequest)(nil), "releasemanager.PromoteRequest")
	proto.RegisterType((*PromoteResponse)(nil), "releasemanager.PromoteResponse")
	proto.RegisterType((*StatusRequest)(nil), "releasemanager.StatusRequest")
	proto.RegisterType((*Environment)(nil), "releasemanager.Environment")
	proto.RegisterType((*StatusResponse)(nil), "releasemanager.StatusResponse")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc1.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc1.SupportPackageIsVersion4

// Client API for ReleaseManager service

type ReleaseManagerClient interface {
	Promote(ctx context.Context, in *PromoteRequest, opts ...grpc1.CallOption) (*PromoteResponse, error)
	Status(ctx context.Context, in *StatusRequest, opts ...grpc1.CallOption) (*StatusResponse, error)
}

type releaseManagerClient struct {
	cc *grpc1.ClientConn
}

func NewReleaseManagerClient(cc *grpc1.ClientConn) ReleaseManagerClient {
	return &releaseManagerClient{cc}
}

func (c *releaseManagerClient) Promote(ctx context.Context, in *PromoteRequest, opts ...grpc1.CallOption) (*PromoteResponse, error) {
	out := new(PromoteResponse)
	err := grpc1.Invoke(ctx, "/releasemanager.ReleaseManager/Promote", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *releaseManagerClient) Status(ctx context.Context, in *StatusRequest, opts ...grpc1.CallOption) (*StatusResponse, error) {
	out := new(StatusResponse)
	err := grpc1.Invoke(ctx, "/releasemanager.ReleaseManager/Status", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for ReleaseManager service

type ReleaseManagerServer interface {
	Promote(context.Context, *PromoteRequest) (*PromoteResponse, error)
	Status(context.Context, *StatusRequest) (*StatusResponse, error)
}

func RegisterReleaseManagerServer(s *grpc1.Server, srv ReleaseManagerServer) {
	s.RegisterService(&_ReleaseManager_serviceDesc, srv)
}

func _ReleaseManager_Promote_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc1.UnaryServerInterceptor) (interface{}, error) {
	in := new(PromoteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ReleaseManagerServer).Promote(ctx, in)
	}
	info := &grpc1.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/releasemanager.ReleaseManager/Promote",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ReleaseManagerServer).Promote(ctx, req.(*PromoteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ReleaseManager_Status_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc1.UnaryServerInterceptor) (interface{}, error) {
	in := new(StatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ReleaseManagerServer).Status(ctx, in)
	}
	info := &grpc1.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/releasemanager.ReleaseManager/Status",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ReleaseManagerServer).Status(ctx, req.(*StatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _ReleaseManager_serviceDesc = grpc1.ServiceDesc{
	ServiceName: "releasemanager.ReleaseManager",
	HandlerType: (*ReleaseManagerServer)(nil),
	Methods: []grpc1.MethodDesc{
		{
			MethodName: "Promote",
			Handler:    _ReleaseManager_Promote_Handler,
		},
		{
			MethodName: "Status",
			Handler:    _ReleaseManager_Status_Handler,
		},
	},
	Streams:  []grpc1.StreamDesc{},
	Metadata: "protos/releasemanager.proto",
}

func init() { proto.RegisterFile("protos/releasemanager.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 337 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x92, 0xcf, 0x4a, 0xc3, 0x40,
	0x10, 0xc6, 0x8d, 0x89, 0x29, 0x9d, 0x62, 0x94, 0x39, 0x48, 0x68, 0xb5, 0x96, 0x9c, 0xea, 0xc1,
	0x16, 0x2a, 0xbe, 0x80, 0xe0, 0x41, 0xa8, 0x20, 0xf1, 0xe6, 0x6d, 0x6d, 0x87, 0x18, 0x30, 0xd9,
	0xb8, 0x3b, 0xed, 0x0b, 0xf8, 0x30, 0xe2, 0x5b, 0x4a, 0x76, 0x37, 0xda, 0x16, 0x4a, 0x0e, 0xde,
	0xe6, 0xcf, 0xb7, 0xdf, 0xfe, 0x76, 0x67, 0x60, 0x50, 0x29, 0xc9, 0x52, 0x4f, 0x15, 0xbd, 0x93,
	0xd0, 0x54, 0x88, 0x52, 0x64, 0xa4, 0x26, 0xa6, 0x8a, 0xd1, 0x76, 0x35, 0x99, 0x43, 0xf4, 0xa4,
	0x64, 0x21, 0x99, 0x52, 0xfa, 0x58, 0x91, 0x66, 0x8c, 0xa1, 0xa3, 0x49, 0xad, 0xf3, 0x05, 0xc5,
	0xde, 0xc8, 0x1b, 0x77, 0xd3, 0x26, 0xc5, 0x11, 0xf4, 0xa8, 0x5c, 0xe7, 0x4a, 0x96, 0x05, 0x95,
	0x1c, 0x1f, 0x9a, 0xee, 0x66, 0x29, 0x21, 0x38, 0xf9, 0x75, 0xd3, 0x95, 0x2c, 0x35, 0xfd, 0xc7,
	0x0e, 0xcf, 0x20, 0xd4, 0x2c, 0x78, 0xa5, 0x63, 0xdf, 0x34, 0x5d, 0x96, 0x5c, 0xc1, 0xf1, 0xb3,
	0x89, 0x5a, 0x99, 0x93, 0x4f, 0x0f, 0x7a, 0xf7, 0x1b, 0x96, 0xa7, 0xe0, 0xb3, 0xc8, 0x9c, 0xaa,
	0x0e, 0xf1, 0x1c, 0xba, 0x0b, 0x59, 0x14, 0x39, 0x33, 0x29, 0x07, 0xf1, 0x57, 0xa8, 0x11, 0xc4,
	0x8a, 0xdf, 0xa4, 0x6a, 0x10, 0x6c, 0x56, 0xdf, 0x58, 0x90, 0xd6, 0x22, 0xa3, 0x38, 0xb0, 0x37,
	0xba, 0x14, 0x11, 0x82, 0xa5, 0x60, 0x8a, 0x8f, 0x46, 0xde, 0xd8, 0x4f, 0x4d, 0x9c, 0x7c, 0x79,
	0x10, 0x35, 0xc4, 0xee, 0x5f, 0xae, 0xc1, 0x5f, 0xd2, 0xda, 0x80, 0xf4, 0x66, 0x83, 0xc9, 0xce,
	0xb0, 0x36, 0x90, 0xd3, 0x5a, 0x87, 0xb7, 0xd0, 0xd1, 0x2c, 0xb2, 0xbc, 0xcc, 0x0c, 0x63, 0xcb,
	0x91, 0x46, 0x8b, 0x53, 0x08, 0x2a, 0x25, 0x97, 0x06, 0xbe, 0xe5, 0x8c, 0x11, 0xce, 0xbe, 0x3d,
	0x88, 0x52, 0x2b, 0x7a, 0xb4, 0x22, 0x9c, 0x43, 0xc7, 0x0d, 0x15, 0x87, 0xbb, 0x06, 0xdb, 0xbb,
	0xd3, 0xbf, 0xdc, 0xdb, 0xb7, 0xaf, 0x4e, 0x0e, 0xf0, 0x01, 0x42, 0xfb, 0x13, 0x78, 0xb1, 0x2b,
	0xde, 0x9a, 0x69, 0x7f, 0xb8, 0xaf, 0xdd, 0x58, 0xdd, 0x85, 0x2f, 0x41, 0xa6, 0xaa, 0xc5, 0x6b,
	0x68, 0x56, 0xfb, 0xe6, 0x27, 0x00, 0x00, 0xff, 0xff, 0x0b, 0xb3, 0xe9, 0x41, 0xf9, 0x02, 0x00,
	0x00,
}
