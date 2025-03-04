// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             (unknown)
// source: v1/idp_service.proto

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// IdentityProviderServiceClient is the client API for IdentityProviderService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type IdentityProviderServiceClient interface {
	GetIdentityProvider(ctx context.Context, in *GetIdentityProviderRequest, opts ...grpc.CallOption) (*IdentityProvider, error)
	ListIdentityProviders(ctx context.Context, in *ListIdentityProvidersRequest, opts ...grpc.CallOption) (*ListIdentityProvidersResponse, error)
	CreateIdentityProvider(ctx context.Context, in *CreateIdentityProviderRequest, opts ...grpc.CallOption) (*IdentityProvider, error)
	UpdateIdentityProvider(ctx context.Context, in *UpdateIdentityProviderRequest, opts ...grpc.CallOption) (*IdentityProvider, error)
	DeleteIdentityProvider(ctx context.Context, in *DeleteIdentityProviderRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	UndeleteIdentityProvider(ctx context.Context, in *UndeleteIdentityProviderRequest, opts ...grpc.CallOption) (*IdentityProvider, error)
	TestIdentityProvider(ctx context.Context, in *TestIdentityProviderRequest, opts ...grpc.CallOption) (*TestIdentityProviderResponse, error)
}

type identityProviderServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewIdentityProviderServiceClient(cc grpc.ClientConnInterface) IdentityProviderServiceClient {
	return &identityProviderServiceClient{cc}
}

func (c *identityProviderServiceClient) GetIdentityProvider(ctx context.Context, in *GetIdentityProviderRequest, opts ...grpc.CallOption) (*IdentityProvider, error) {
	out := new(IdentityProvider)
	err := c.cc.Invoke(ctx, "/bytebase.v1.IdentityProviderService/GetIdentityProvider", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *identityProviderServiceClient) ListIdentityProviders(ctx context.Context, in *ListIdentityProvidersRequest, opts ...grpc.CallOption) (*ListIdentityProvidersResponse, error) {
	out := new(ListIdentityProvidersResponse)
	err := c.cc.Invoke(ctx, "/bytebase.v1.IdentityProviderService/ListIdentityProviders", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *identityProviderServiceClient) CreateIdentityProvider(ctx context.Context, in *CreateIdentityProviderRequest, opts ...grpc.CallOption) (*IdentityProvider, error) {
	out := new(IdentityProvider)
	err := c.cc.Invoke(ctx, "/bytebase.v1.IdentityProviderService/CreateIdentityProvider", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *identityProviderServiceClient) UpdateIdentityProvider(ctx context.Context, in *UpdateIdentityProviderRequest, opts ...grpc.CallOption) (*IdentityProvider, error) {
	out := new(IdentityProvider)
	err := c.cc.Invoke(ctx, "/bytebase.v1.IdentityProviderService/UpdateIdentityProvider", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *identityProviderServiceClient) DeleteIdentityProvider(ctx context.Context, in *DeleteIdentityProviderRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/bytebase.v1.IdentityProviderService/DeleteIdentityProvider", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *identityProviderServiceClient) UndeleteIdentityProvider(ctx context.Context, in *UndeleteIdentityProviderRequest, opts ...grpc.CallOption) (*IdentityProvider, error) {
	out := new(IdentityProvider)
	err := c.cc.Invoke(ctx, "/bytebase.v1.IdentityProviderService/UndeleteIdentityProvider", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *identityProviderServiceClient) TestIdentityProvider(ctx context.Context, in *TestIdentityProviderRequest, opts ...grpc.CallOption) (*TestIdentityProviderResponse, error) {
	out := new(TestIdentityProviderResponse)
	err := c.cc.Invoke(ctx, "/bytebase.v1.IdentityProviderService/TestIdentityProvider", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// IdentityProviderServiceServer is the server API for IdentityProviderService service.
// All implementations must embed UnimplementedIdentityProviderServiceServer
// for forward compatibility
type IdentityProviderServiceServer interface {
	GetIdentityProvider(context.Context, *GetIdentityProviderRequest) (*IdentityProvider, error)
	ListIdentityProviders(context.Context, *ListIdentityProvidersRequest) (*ListIdentityProvidersResponse, error)
	CreateIdentityProvider(context.Context, *CreateIdentityProviderRequest) (*IdentityProvider, error)
	UpdateIdentityProvider(context.Context, *UpdateIdentityProviderRequest) (*IdentityProvider, error)
	DeleteIdentityProvider(context.Context, *DeleteIdentityProviderRequest) (*emptypb.Empty, error)
	UndeleteIdentityProvider(context.Context, *UndeleteIdentityProviderRequest) (*IdentityProvider, error)
	TestIdentityProvider(context.Context, *TestIdentityProviderRequest) (*TestIdentityProviderResponse, error)
	mustEmbedUnimplementedIdentityProviderServiceServer()
}

// UnimplementedIdentityProviderServiceServer must be embedded to have forward compatible implementations.
type UnimplementedIdentityProviderServiceServer struct {
}

func (UnimplementedIdentityProviderServiceServer) GetIdentityProvider(context.Context, *GetIdentityProviderRequest) (*IdentityProvider, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetIdentityProvider not implemented")
}
func (UnimplementedIdentityProviderServiceServer) ListIdentityProviders(context.Context, *ListIdentityProvidersRequest) (*ListIdentityProvidersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListIdentityProviders not implemented")
}
func (UnimplementedIdentityProviderServiceServer) CreateIdentityProvider(context.Context, *CreateIdentityProviderRequest) (*IdentityProvider, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateIdentityProvider not implemented")
}
func (UnimplementedIdentityProviderServiceServer) UpdateIdentityProvider(context.Context, *UpdateIdentityProviderRequest) (*IdentityProvider, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateIdentityProvider not implemented")
}
func (UnimplementedIdentityProviderServiceServer) DeleteIdentityProvider(context.Context, *DeleteIdentityProviderRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteIdentityProvider not implemented")
}
func (UnimplementedIdentityProviderServiceServer) UndeleteIdentityProvider(context.Context, *UndeleteIdentityProviderRequest) (*IdentityProvider, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UndeleteIdentityProvider not implemented")
}
func (UnimplementedIdentityProviderServiceServer) TestIdentityProvider(context.Context, *TestIdentityProviderRequest) (*TestIdentityProviderResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TestIdentityProvider not implemented")
}
func (UnimplementedIdentityProviderServiceServer) mustEmbedUnimplementedIdentityProviderServiceServer() {
}

// UnsafeIdentityProviderServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to IdentityProviderServiceServer will
// result in compilation errors.
type UnsafeIdentityProviderServiceServer interface {
	mustEmbedUnimplementedIdentityProviderServiceServer()
}

func RegisterIdentityProviderServiceServer(s grpc.ServiceRegistrar, srv IdentityProviderServiceServer) {
	s.RegisterService(&IdentityProviderService_ServiceDesc, srv)
}

func _IdentityProviderService_GetIdentityProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetIdentityProviderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IdentityProviderServiceServer).GetIdentityProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bytebase.v1.IdentityProviderService/GetIdentityProvider",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IdentityProviderServiceServer).GetIdentityProvider(ctx, req.(*GetIdentityProviderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IdentityProviderService_ListIdentityProviders_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListIdentityProvidersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IdentityProviderServiceServer).ListIdentityProviders(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bytebase.v1.IdentityProviderService/ListIdentityProviders",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IdentityProviderServiceServer).ListIdentityProviders(ctx, req.(*ListIdentityProvidersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IdentityProviderService_CreateIdentityProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateIdentityProviderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IdentityProviderServiceServer).CreateIdentityProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bytebase.v1.IdentityProviderService/CreateIdentityProvider",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IdentityProviderServiceServer).CreateIdentityProvider(ctx, req.(*CreateIdentityProviderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IdentityProviderService_UpdateIdentityProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateIdentityProviderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IdentityProviderServiceServer).UpdateIdentityProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bytebase.v1.IdentityProviderService/UpdateIdentityProvider",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IdentityProviderServiceServer).UpdateIdentityProvider(ctx, req.(*UpdateIdentityProviderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IdentityProviderService_DeleteIdentityProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteIdentityProviderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IdentityProviderServiceServer).DeleteIdentityProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bytebase.v1.IdentityProviderService/DeleteIdentityProvider",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IdentityProviderServiceServer).DeleteIdentityProvider(ctx, req.(*DeleteIdentityProviderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IdentityProviderService_UndeleteIdentityProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UndeleteIdentityProviderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IdentityProviderServiceServer).UndeleteIdentityProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bytebase.v1.IdentityProviderService/UndeleteIdentityProvider",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IdentityProviderServiceServer).UndeleteIdentityProvider(ctx, req.(*UndeleteIdentityProviderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IdentityProviderService_TestIdentityProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TestIdentityProviderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IdentityProviderServiceServer).TestIdentityProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bytebase.v1.IdentityProviderService/TestIdentityProvider",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IdentityProviderServiceServer).TestIdentityProvider(ctx, req.(*TestIdentityProviderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// IdentityProviderService_ServiceDesc is the grpc.ServiceDesc for IdentityProviderService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var IdentityProviderService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "bytebase.v1.IdentityProviderService",
	HandlerType: (*IdentityProviderServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetIdentityProvider",
			Handler:    _IdentityProviderService_GetIdentityProvider_Handler,
		},
		{
			MethodName: "ListIdentityProviders",
			Handler:    _IdentityProviderService_ListIdentityProviders_Handler,
		},
		{
			MethodName: "CreateIdentityProvider",
			Handler:    _IdentityProviderService_CreateIdentityProvider_Handler,
		},
		{
			MethodName: "UpdateIdentityProvider",
			Handler:    _IdentityProviderService_UpdateIdentityProvider_Handler,
		},
		{
			MethodName: "DeleteIdentityProvider",
			Handler:    _IdentityProviderService_DeleteIdentityProvider_Handler,
		},
		{
			MethodName: "UndeleteIdentityProvider",
			Handler:    _IdentityProviderService_UndeleteIdentityProvider_Handler,
		},
		{
			MethodName: "TestIdentityProvider",
			Handler:    _IdentityProviderService_TestIdentityProvider_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "v1/idp_service.proto",
}
