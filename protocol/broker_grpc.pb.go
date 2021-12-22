// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package protocol

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// PricesClient is the client API for Prices service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PricesClient interface {
	SubAll(ctx context.Context, in *Request, opts ...grpc.CallOption) (Prices_SubAllClient, error)
}

type pricesClient struct {
	cc grpc.ClientConnInterface
}

func NewPricesClient(cc grpc.ClientConnInterface) PricesClient {
	return &pricesClient{cc}
}

func (c *pricesClient) SubAll(ctx context.Context, in *Request, opts ...grpc.CallOption) (Prices_SubAllClient, error) {
	stream, err := c.cc.NewStream(ctx, &Prices_ServiceDesc.Streams[0], "/pgrpc.Prices/SubAll", opts...)
	if err != nil {
		return nil, err
	}
	x := &pricesSubAllClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Prices_SubAllClient interface {
	Recv() (*Stock, error)
	grpc.ClientStream
}

type pricesSubAllClient struct {
	grpc.ClientStream
}

func (x *pricesSubAllClient) Recv() (*Stock, error) {
	m := new(Stock)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// PricesServer is the server API for Prices service.
// All implementations must embed UnimplementedPricesServer
// for forward compatibility
type PricesServer interface {
	SubAll(*Request, Prices_SubAllServer) error
	mustEmbedUnimplementedPricesServer()
}

// UnimplementedPricesServer must be embedded to have forward compatible implementations.
type UnimplementedPricesServer struct {
}

func (UnimplementedPricesServer) SubAll(*Request, Prices_SubAllServer) error {
	return status.Errorf(codes.Unimplemented, "method SubAll not implemented")
}
func (UnimplementedPricesServer) mustEmbedUnimplementedPricesServer() {}

// UnsafePricesServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PricesServer will
// result in compilation errors.
type UnsafePricesServer interface {
	mustEmbedUnimplementedPricesServer()
}

func RegisterPricesServer(s grpc.ServiceRegistrar, srv PricesServer) {
	s.RegisterService(&Prices_ServiceDesc, srv)
}

func _Prices_SubAll_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Request)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(PricesServer).SubAll(m, &pricesSubAllServer{stream})
}

type Prices_SubAllServer interface {
	Send(*Stock) error
	grpc.ServerStream
}

type pricesSubAllServer struct {
	grpc.ServerStream
}

func (x *pricesSubAllServer) Send(m *Stock) error {
	return x.ServerStream.SendMsg(m)
}

// Prices_ServiceDesc is the grpc.ServiceDesc for Prices service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Prices_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "pgrpc.Prices",
	HandlerType: (*PricesServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "SubAll",
			Handler:       _Prices_SubAll_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "protocol/broker.proto",
}

// SwopsClient is the client API for Swops service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type SwopsClient interface {
	Swop(ctx context.Context, opts ...grpc.CallOption) (Swops_SwopClient, error)
}

type swopsClient struct {
	cc grpc.ClientConnInterface
}

func NewSwopsClient(cc grpc.ClientConnInterface) SwopsClient {
	return &swopsClient{cc}
}

func (c *swopsClient) Swop(ctx context.Context, opts ...grpc.CallOption) (Swops_SwopClient, error) {
	stream, err := c.cc.NewStream(ctx, &Swops_ServiceDesc.Streams[0], "/pgrpc.Swops/Swop", opts...)
	if err != nil {
		return nil, err
	}
	x := &swopsSwopClient{stream}
	return x, nil
}

type Swops_SwopClient interface {
	Send(*Application) error
	Recv() (*Response, error)
	grpc.ClientStream
}

type swopsSwopClient struct {
	grpc.ClientStream
}

func (x *swopsSwopClient) Send(m *Application) error {
	return x.ClientStream.SendMsg(m)
}

func (x *swopsSwopClient) Recv() (*Response, error) {
	m := new(Response)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// SwopsServer is the server API for Swops service.
// All implementations must embed UnimplementedSwopsServer
// for forward compatibility
type SwopsServer interface {
	Swop(Swops_SwopServer) error
	mustEmbedUnimplementedSwopsServer()
}

// UnimplementedSwopsServer must be embedded to have forward compatible implementations.
type UnimplementedSwopsServer struct {
}

func (UnimplementedSwopsServer) Swop(Swops_SwopServer) error {
	return status.Errorf(codes.Unimplemented, "method Swop not implemented")
}
func (UnimplementedSwopsServer) mustEmbedUnimplementedSwopsServer() {}

// UnsafeSwopsServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SwopsServer will
// result in compilation errors.
type UnsafeSwopsServer interface {
	mustEmbedUnimplementedSwopsServer()
}

func RegisterSwopsServer(s grpc.ServiceRegistrar, srv SwopsServer) {
	s.RegisterService(&Swops_ServiceDesc, srv)
}

func _Swops_Swop_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(SwopsServer).Swop(&swopsSwopServer{stream})
}

type Swops_SwopServer interface {
	Send(*Response) error
	Recv() (*Application, error)
	grpc.ServerStream
}

type swopsSwopServer struct {
	grpc.ServerStream
}

func (x *swopsSwopServer) Send(m *Response) error {
	return x.ServerStream.SendMsg(m)
}

func (x *swopsSwopServer) Recv() (*Application, error) {
	m := new(Application)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Swops_ServiceDesc is the grpc.ServiceDesc for Swops service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Swops_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "pgrpc.Swops",
	HandlerType: (*SwopsServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Swop",
			Handler:       _Swops_Swop_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "protocol/broker.proto",
}