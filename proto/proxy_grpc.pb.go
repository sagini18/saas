// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.4.0
// - protoc             v5.27.1
// source: proto/proxy.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.62.0 or later.
const _ = grpc.SupportPackageIsVersion8

const (
	Proxy_Subscribe_FullMethodName      = "/proto.Proxy/Subscribe"
	Proxy_StreamCommands_FullMethodName = "/proto.Proxy/StreamCommands"
)

// ProxyClient is the client API for Proxy service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ProxyClient interface {
	Subscribe(ctx context.Context, in *SubscriptionRequest, opts ...grpc.CallOption) (*SubscriptionResponse, error)
	StreamCommands(ctx context.Context, opts ...grpc.CallOption) (Proxy_StreamCommandsClient, error)
}

type proxyClient struct {
	cc grpc.ClientConnInterface
}

func NewProxyClient(cc grpc.ClientConnInterface) ProxyClient {
	return &proxyClient{cc}
}

func (c *proxyClient) Subscribe(ctx context.Context, in *SubscriptionRequest, opts ...grpc.CallOption) (*SubscriptionResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SubscriptionResponse)
	err := c.cc.Invoke(ctx, Proxy_Subscribe_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *proxyClient) StreamCommands(ctx context.Context, opts ...grpc.CallOption) (Proxy_StreamCommandsClient, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Proxy_ServiceDesc.Streams[0], Proxy_StreamCommands_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &proxyStreamCommandsClient{ClientStream: stream}
	return x, nil
}

type Proxy_StreamCommandsClient interface {
	Send(*InitialRequest) error
	Recv() (*Response, error)
	grpc.ClientStream
}

type proxyStreamCommandsClient struct {
	grpc.ClientStream
}

func (x *proxyStreamCommandsClient) Send(m *InitialRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *proxyStreamCommandsClient) Recv() (*Response, error) {
	m := new(Response)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ProxyServer is the server API for Proxy service.
// All implementations must embed UnimplementedProxyServer
// for forward compatibility
type ProxyServer interface {
	Subscribe(context.Context, *SubscriptionRequest) (*SubscriptionResponse, error)
	StreamCommands(Proxy_StreamCommandsServer) error
	mustEmbedUnimplementedProxyServer()
}

// UnimplementedProxyServer must be embedded to have forward compatible implementations.
type UnimplementedProxyServer struct {
}

func (UnimplementedProxyServer) Subscribe(context.Context, *SubscriptionRequest) (*SubscriptionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Subscribe not implemented")
}
func (UnimplementedProxyServer) StreamCommands(Proxy_StreamCommandsServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamCommands not implemented")
}
func (UnimplementedProxyServer) mustEmbedUnimplementedProxyServer() {}

// UnsafeProxyServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ProxyServer will
// result in compilation errors.
type UnsafeProxyServer interface {
	mustEmbedUnimplementedProxyServer()
}

func RegisterProxyServer(s grpc.ServiceRegistrar, srv ProxyServer) {
	s.RegisterService(&Proxy_ServiceDesc, srv)
}

func _Proxy_Subscribe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SubscriptionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProxyServer).Subscribe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Proxy_Subscribe_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProxyServer).Subscribe(ctx, req.(*SubscriptionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Proxy_StreamCommands_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ProxyServer).StreamCommands(&proxyStreamCommandsServer{ServerStream: stream})
}

type Proxy_StreamCommandsServer interface {
	Send(*Response) error
	Recv() (*InitialRequest, error)
	grpc.ServerStream
}

type proxyStreamCommandsServer struct {
	grpc.ServerStream
}

func (x *proxyStreamCommandsServer) Send(m *Response) error {
	return x.ServerStream.SendMsg(m)
}

func (x *proxyStreamCommandsServer) Recv() (*InitialRequest, error) {
	m := new(InitialRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Proxy_ServiceDesc is the grpc.ServiceDesc for Proxy service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Proxy_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.Proxy",
	HandlerType: (*ProxyServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Subscribe",
			Handler:    _Proxy_Subscribe_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamCommands",
			Handler:       _Proxy_StreamCommands_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "proto/proxy.proto",
}
