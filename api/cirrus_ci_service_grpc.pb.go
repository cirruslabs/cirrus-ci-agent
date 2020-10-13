// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package api

import (
	context "context"
	empty "github.com/golang/protobuf/ptypes/empty"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion7

// CirrusConfigurationEvaluatorServiceClient is the client API for CirrusConfigurationEvaluatorService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type CirrusConfigurationEvaluatorServiceClient interface {
	EvaluateConfig(ctx context.Context, in *EvaluateConfigRequest, opts ...grpc.CallOption) (*EvaluateConfigResponse, error)
}

type cirrusConfigurationEvaluatorServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewCirrusConfigurationEvaluatorServiceClient(cc grpc.ClientConnInterface) CirrusConfigurationEvaluatorServiceClient {
	return &cirrusConfigurationEvaluatorServiceClient{cc}
}

func (c *cirrusConfigurationEvaluatorServiceClient) EvaluateConfig(ctx context.Context, in *EvaluateConfigRequest, opts ...grpc.CallOption) (*EvaluateConfigResponse, error) {
	out := new(EvaluateConfigResponse)
	err := c.cc.Invoke(ctx, "/org.cirruslabs.ci.services.cirruscigrpc.CirrusConfigurationEvaluatorService/EvaluateConfig", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CirrusConfigurationEvaluatorServiceServer is the server API for CirrusConfigurationEvaluatorService service.
// All implementations must embed UnimplementedCirrusConfigurationEvaluatorServiceServer
// for forward compatibility
type CirrusConfigurationEvaluatorServiceServer interface {
	EvaluateConfig(context.Context, *EvaluateConfigRequest) (*EvaluateConfigResponse, error)
	mustEmbedUnimplementedCirrusConfigurationEvaluatorServiceServer()
}

// UnimplementedCirrusConfigurationEvaluatorServiceServer must be embedded to have forward compatible implementations.
type UnimplementedCirrusConfigurationEvaluatorServiceServer struct {
}

func (UnimplementedCirrusConfigurationEvaluatorServiceServer) EvaluateConfig(context.Context, *EvaluateConfigRequest) (*EvaluateConfigResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method EvaluateConfig not implemented")
}
func (UnimplementedCirrusConfigurationEvaluatorServiceServer) mustEmbedUnimplementedCirrusConfigurationEvaluatorServiceServer() {
}

// UnsafeCirrusConfigurationEvaluatorServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to CirrusConfigurationEvaluatorServiceServer will
// result in compilation errors.
type UnsafeCirrusConfigurationEvaluatorServiceServer interface {
	mustEmbedUnimplementedCirrusConfigurationEvaluatorServiceServer()
}

func RegisterCirrusConfigurationEvaluatorServiceServer(s *grpc.Server, srv CirrusConfigurationEvaluatorServiceServer) {
	s.RegisterService(&_CirrusConfigurationEvaluatorService_serviceDesc, srv)
}

func _CirrusConfigurationEvaluatorService_EvaluateConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EvaluateConfigRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CirrusConfigurationEvaluatorServiceServer).EvaluateConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.cirruslabs.ci.services.cirruscigrpc.CirrusConfigurationEvaluatorService/EvaluateConfig",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CirrusConfigurationEvaluatorServiceServer).EvaluateConfig(ctx, req.(*EvaluateConfigRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _CirrusConfigurationEvaluatorService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "org.cirruslabs.ci.services.cirruscigrpc.CirrusConfigurationEvaluatorService",
	HandlerType: (*CirrusConfigurationEvaluatorServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "EvaluateConfig",
			Handler:    _CirrusConfigurationEvaluatorService_EvaluateConfig_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "cirrus_ci_service.proto",
}

// CirrusCIServiceClient is the client API for CirrusCIService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type CirrusCIServiceClient interface {
	InitialCommands(ctx context.Context, in *InitialCommandsRequest, opts ...grpc.CallOption) (*CommandsResponse, error)
	ReportSingleCommand(ctx context.Context, in *ReportSingleCommandRequest, opts ...grpc.CallOption) (*ReportSingleCommandResponse, error)
	ReportAnnotations(ctx context.Context, in *ReportAnnotationsCommandRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	StreamLogs(ctx context.Context, opts ...grpc.CallOption) (CirrusCIService_StreamLogsClient, error)
	SaveLogs(ctx context.Context, opts ...grpc.CallOption) (CirrusCIService_SaveLogsClient, error)
	UploadCache(ctx context.Context, opts ...grpc.CallOption) (CirrusCIService_UploadCacheClient, error)
	UploadArtifacts(ctx context.Context, opts ...grpc.CallOption) (CirrusCIService_UploadArtifactsClient, error)
	DownloadCache(ctx context.Context, in *DownloadCacheRequest, opts ...grpc.CallOption) (CirrusCIService_DownloadCacheClient, error)
	CacheInfo(ctx context.Context, in *CacheInfoRequest, opts ...grpc.CallOption) (*CacheInfoResponse, error)
	Ping(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*empty.Empty, error)
	Heartbeat(ctx context.Context, in *HeartbeatRequest, opts ...grpc.CallOption) (*HeartbeatResponse, error)
	ReportStopHook(ctx context.Context, in *ReportStopHookRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	ReportAgentError(ctx context.Context, in *ReportAgentProblemRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	ReportAgentWarning(ctx context.Context, in *ReportAgentProblemRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	ReportAgentSignal(ctx context.Context, in *ReportAgentSignalRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	ReportAgentLogs(ctx context.Context, in *ReportAgentLogsRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	ParseConfig(ctx context.Context, in *ParseConfigRequest, opts ...grpc.CallOption) (*ParseConfigResponse, error)
}

type cirrusCIServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewCirrusCIServiceClient(cc grpc.ClientConnInterface) CirrusCIServiceClient {
	return &cirrusCIServiceClient{cc}
}

func (c *cirrusCIServiceClient) InitialCommands(ctx context.Context, in *InitialCommandsRequest, opts ...grpc.CallOption) (*CommandsResponse, error) {
	out := new(CommandsResponse)
	err := c.cc.Invoke(ctx, "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/InitialCommands", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cirrusCIServiceClient) ReportSingleCommand(ctx context.Context, in *ReportSingleCommandRequest, opts ...grpc.CallOption) (*ReportSingleCommandResponse, error) {
	out := new(ReportSingleCommandResponse)
	err := c.cc.Invoke(ctx, "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/ReportSingleCommand", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cirrusCIServiceClient) ReportAnnotations(ctx context.Context, in *ReportAnnotationsCommandRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/ReportAnnotations", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cirrusCIServiceClient) StreamLogs(ctx context.Context, opts ...grpc.CallOption) (CirrusCIService_StreamLogsClient, error) {
	stream, err := c.cc.NewStream(ctx, &_CirrusCIService_serviceDesc.Streams[0], "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/StreamLogs", opts...)
	if err != nil {
		return nil, err
	}
	x := &cirrusCIServiceStreamLogsClient{stream}
	return x, nil
}

type CirrusCIService_StreamLogsClient interface {
	Send(*LogEntry) error
	CloseAndRecv() (*UploadLogsResponse, error)
	grpc.ClientStream
}

type cirrusCIServiceStreamLogsClient struct {
	grpc.ClientStream
}

func (x *cirrusCIServiceStreamLogsClient) Send(m *LogEntry) error {
	return x.ClientStream.SendMsg(m)
}

func (x *cirrusCIServiceStreamLogsClient) CloseAndRecv() (*UploadLogsResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(UploadLogsResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *cirrusCIServiceClient) SaveLogs(ctx context.Context, opts ...grpc.CallOption) (CirrusCIService_SaveLogsClient, error) {
	stream, err := c.cc.NewStream(ctx, &_CirrusCIService_serviceDesc.Streams[1], "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/SaveLogs", opts...)
	if err != nil {
		return nil, err
	}
	x := &cirrusCIServiceSaveLogsClient{stream}
	return x, nil
}

type CirrusCIService_SaveLogsClient interface {
	Send(*LogEntry) error
	CloseAndRecv() (*UploadLogsResponse, error)
	grpc.ClientStream
}

type cirrusCIServiceSaveLogsClient struct {
	grpc.ClientStream
}

func (x *cirrusCIServiceSaveLogsClient) Send(m *LogEntry) error {
	return x.ClientStream.SendMsg(m)
}

func (x *cirrusCIServiceSaveLogsClient) CloseAndRecv() (*UploadLogsResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(UploadLogsResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *cirrusCIServiceClient) UploadCache(ctx context.Context, opts ...grpc.CallOption) (CirrusCIService_UploadCacheClient, error) {
	stream, err := c.cc.NewStream(ctx, &_CirrusCIService_serviceDesc.Streams[2], "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/UploadCache", opts...)
	if err != nil {
		return nil, err
	}
	x := &cirrusCIServiceUploadCacheClient{stream}
	return x, nil
}

type CirrusCIService_UploadCacheClient interface {
	Send(*CacheEntry) error
	CloseAndRecv() (*UploadCacheResponse, error)
	grpc.ClientStream
}

type cirrusCIServiceUploadCacheClient struct {
	grpc.ClientStream
}

func (x *cirrusCIServiceUploadCacheClient) Send(m *CacheEntry) error {
	return x.ClientStream.SendMsg(m)
}

func (x *cirrusCIServiceUploadCacheClient) CloseAndRecv() (*UploadCacheResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(UploadCacheResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *cirrusCIServiceClient) UploadArtifacts(ctx context.Context, opts ...grpc.CallOption) (CirrusCIService_UploadArtifactsClient, error) {
	stream, err := c.cc.NewStream(ctx, &_CirrusCIService_serviceDesc.Streams[3], "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/UploadArtifacts", opts...)
	if err != nil {
		return nil, err
	}
	x := &cirrusCIServiceUploadArtifactsClient{stream}
	return x, nil
}

type CirrusCIService_UploadArtifactsClient interface {
	Send(*ArtifactEntry) error
	CloseAndRecv() (*UploadArtifactsResponse, error)
	grpc.ClientStream
}

type cirrusCIServiceUploadArtifactsClient struct {
	grpc.ClientStream
}

func (x *cirrusCIServiceUploadArtifactsClient) Send(m *ArtifactEntry) error {
	return x.ClientStream.SendMsg(m)
}

func (x *cirrusCIServiceUploadArtifactsClient) CloseAndRecv() (*UploadArtifactsResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(UploadArtifactsResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *cirrusCIServiceClient) DownloadCache(ctx context.Context, in *DownloadCacheRequest, opts ...grpc.CallOption) (CirrusCIService_DownloadCacheClient, error) {
	stream, err := c.cc.NewStream(ctx, &_CirrusCIService_serviceDesc.Streams[4], "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/DownloadCache", opts...)
	if err != nil {
		return nil, err
	}
	x := &cirrusCIServiceDownloadCacheClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type CirrusCIService_DownloadCacheClient interface {
	Recv() (*DataChunk, error)
	grpc.ClientStream
}

type cirrusCIServiceDownloadCacheClient struct {
	grpc.ClientStream
}

func (x *cirrusCIServiceDownloadCacheClient) Recv() (*DataChunk, error) {
	m := new(DataChunk)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *cirrusCIServiceClient) CacheInfo(ctx context.Context, in *CacheInfoRequest, opts ...grpc.CallOption) (*CacheInfoResponse, error) {
	out := new(CacheInfoResponse)
	err := c.cc.Invoke(ctx, "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/CacheInfo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cirrusCIServiceClient) Ping(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/Ping", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cirrusCIServiceClient) Heartbeat(ctx context.Context, in *HeartbeatRequest, opts ...grpc.CallOption) (*HeartbeatResponse, error) {
	out := new(HeartbeatResponse)
	err := c.cc.Invoke(ctx, "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/Heartbeat", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cirrusCIServiceClient) ReportStopHook(ctx context.Context, in *ReportStopHookRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/ReportStopHook", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cirrusCIServiceClient) ReportAgentError(ctx context.Context, in *ReportAgentProblemRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/ReportAgentError", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cirrusCIServiceClient) ReportAgentWarning(ctx context.Context, in *ReportAgentProblemRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/ReportAgentWarning", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cirrusCIServiceClient) ReportAgentSignal(ctx context.Context, in *ReportAgentSignalRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/ReportAgentSignal", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cirrusCIServiceClient) ReportAgentLogs(ctx context.Context, in *ReportAgentLogsRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/ReportAgentLogs", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cirrusCIServiceClient) ParseConfig(ctx context.Context, in *ParseConfigRequest, opts ...grpc.CallOption) (*ParseConfigResponse, error) {
	out := new(ParseConfigResponse)
	err := c.cc.Invoke(ctx, "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/ParseConfig", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CirrusCIServiceServer is the server API for CirrusCIService service.
// All implementations must embed UnimplementedCirrusCIServiceServer
// for forward compatibility
type CirrusCIServiceServer interface {
	InitialCommands(context.Context, *InitialCommandsRequest) (*CommandsResponse, error)
	ReportSingleCommand(context.Context, *ReportSingleCommandRequest) (*ReportSingleCommandResponse, error)
	ReportAnnotations(context.Context, *ReportAnnotationsCommandRequest) (*empty.Empty, error)
	StreamLogs(CirrusCIService_StreamLogsServer) error
	SaveLogs(CirrusCIService_SaveLogsServer) error
	UploadCache(CirrusCIService_UploadCacheServer) error
	UploadArtifacts(CirrusCIService_UploadArtifactsServer) error
	DownloadCache(*DownloadCacheRequest, CirrusCIService_DownloadCacheServer) error
	CacheInfo(context.Context, *CacheInfoRequest) (*CacheInfoResponse, error)
	Ping(context.Context, *empty.Empty) (*empty.Empty, error)
	Heartbeat(context.Context, *HeartbeatRequest) (*HeartbeatResponse, error)
	ReportStopHook(context.Context, *ReportStopHookRequest) (*empty.Empty, error)
	ReportAgentError(context.Context, *ReportAgentProblemRequest) (*empty.Empty, error)
	ReportAgentWarning(context.Context, *ReportAgentProblemRequest) (*empty.Empty, error)
	ReportAgentSignal(context.Context, *ReportAgentSignalRequest) (*empty.Empty, error)
	ReportAgentLogs(context.Context, *ReportAgentLogsRequest) (*empty.Empty, error)
	ParseConfig(context.Context, *ParseConfigRequest) (*ParseConfigResponse, error)
	mustEmbedUnimplementedCirrusCIServiceServer()
}

// UnimplementedCirrusCIServiceServer must be embedded to have forward compatible implementations.
type UnimplementedCirrusCIServiceServer struct {
}

func (UnimplementedCirrusCIServiceServer) InitialCommands(context.Context, *InitialCommandsRequest) (*CommandsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InitialCommands not implemented")
}
func (UnimplementedCirrusCIServiceServer) ReportSingleCommand(context.Context, *ReportSingleCommandRequest) (*ReportSingleCommandResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReportSingleCommand not implemented")
}
func (UnimplementedCirrusCIServiceServer) ReportAnnotations(context.Context, *ReportAnnotationsCommandRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReportAnnotations not implemented")
}
func (UnimplementedCirrusCIServiceServer) StreamLogs(CirrusCIService_StreamLogsServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamLogs not implemented")
}
func (UnimplementedCirrusCIServiceServer) SaveLogs(CirrusCIService_SaveLogsServer) error {
	return status.Errorf(codes.Unimplemented, "method SaveLogs not implemented")
}
func (UnimplementedCirrusCIServiceServer) UploadCache(CirrusCIService_UploadCacheServer) error {
	return status.Errorf(codes.Unimplemented, "method UploadCache not implemented")
}
func (UnimplementedCirrusCIServiceServer) UploadArtifacts(CirrusCIService_UploadArtifactsServer) error {
	return status.Errorf(codes.Unimplemented, "method UploadArtifacts not implemented")
}
func (UnimplementedCirrusCIServiceServer) DownloadCache(*DownloadCacheRequest, CirrusCIService_DownloadCacheServer) error {
	return status.Errorf(codes.Unimplemented, "method DownloadCache not implemented")
}
func (UnimplementedCirrusCIServiceServer) CacheInfo(context.Context, *CacheInfoRequest) (*CacheInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CacheInfo not implemented")
}
func (UnimplementedCirrusCIServiceServer) Ping(context.Context, *empty.Empty) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ping not implemented")
}
func (UnimplementedCirrusCIServiceServer) Heartbeat(context.Context, *HeartbeatRequest) (*HeartbeatResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Heartbeat not implemented")
}
func (UnimplementedCirrusCIServiceServer) ReportStopHook(context.Context, *ReportStopHookRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReportStopHook not implemented")
}
func (UnimplementedCirrusCIServiceServer) ReportAgentError(context.Context, *ReportAgentProblemRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReportAgentError not implemented")
}
func (UnimplementedCirrusCIServiceServer) ReportAgentWarning(context.Context, *ReportAgentProblemRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReportAgentWarning not implemented")
}
func (UnimplementedCirrusCIServiceServer) ReportAgentSignal(context.Context, *ReportAgentSignalRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReportAgentSignal not implemented")
}
func (UnimplementedCirrusCIServiceServer) ReportAgentLogs(context.Context, *ReportAgentLogsRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReportAgentLogs not implemented")
}
func (UnimplementedCirrusCIServiceServer) ParseConfig(context.Context, *ParseConfigRequest) (*ParseConfigResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ParseConfig not implemented")
}
func (UnimplementedCirrusCIServiceServer) mustEmbedUnimplementedCirrusCIServiceServer() {}

// UnsafeCirrusCIServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to CirrusCIServiceServer will
// result in compilation errors.
type UnsafeCirrusCIServiceServer interface {
	mustEmbedUnimplementedCirrusCIServiceServer()
}

func RegisterCirrusCIServiceServer(s *grpc.Server, srv CirrusCIServiceServer) {
	s.RegisterService(&_CirrusCIService_serviceDesc, srv)
}

func _CirrusCIService_InitialCommands_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitialCommandsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CirrusCIServiceServer).InitialCommands(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/InitialCommands",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CirrusCIServiceServer).InitialCommands(ctx, req.(*InitialCommandsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CirrusCIService_ReportSingleCommand_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReportSingleCommandRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CirrusCIServiceServer).ReportSingleCommand(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/ReportSingleCommand",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CirrusCIServiceServer).ReportSingleCommand(ctx, req.(*ReportSingleCommandRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CirrusCIService_ReportAnnotations_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReportAnnotationsCommandRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CirrusCIServiceServer).ReportAnnotations(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/ReportAnnotations",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CirrusCIServiceServer).ReportAnnotations(ctx, req.(*ReportAnnotationsCommandRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CirrusCIService_StreamLogs_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(CirrusCIServiceServer).StreamLogs(&cirrusCIServiceStreamLogsServer{stream})
}

type CirrusCIService_StreamLogsServer interface {
	SendAndClose(*UploadLogsResponse) error
	Recv() (*LogEntry, error)
	grpc.ServerStream
}

type cirrusCIServiceStreamLogsServer struct {
	grpc.ServerStream
}

func (x *cirrusCIServiceStreamLogsServer) SendAndClose(m *UploadLogsResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *cirrusCIServiceStreamLogsServer) Recv() (*LogEntry, error) {
	m := new(LogEntry)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _CirrusCIService_SaveLogs_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(CirrusCIServiceServer).SaveLogs(&cirrusCIServiceSaveLogsServer{stream})
}

type CirrusCIService_SaveLogsServer interface {
	SendAndClose(*UploadLogsResponse) error
	Recv() (*LogEntry, error)
	grpc.ServerStream
}

type cirrusCIServiceSaveLogsServer struct {
	grpc.ServerStream
}

func (x *cirrusCIServiceSaveLogsServer) SendAndClose(m *UploadLogsResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *cirrusCIServiceSaveLogsServer) Recv() (*LogEntry, error) {
	m := new(LogEntry)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _CirrusCIService_UploadCache_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(CirrusCIServiceServer).UploadCache(&cirrusCIServiceUploadCacheServer{stream})
}

type CirrusCIService_UploadCacheServer interface {
	SendAndClose(*UploadCacheResponse) error
	Recv() (*CacheEntry, error)
	grpc.ServerStream
}

type cirrusCIServiceUploadCacheServer struct {
	grpc.ServerStream
}

func (x *cirrusCIServiceUploadCacheServer) SendAndClose(m *UploadCacheResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *cirrusCIServiceUploadCacheServer) Recv() (*CacheEntry, error) {
	m := new(CacheEntry)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _CirrusCIService_UploadArtifacts_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(CirrusCIServiceServer).UploadArtifacts(&cirrusCIServiceUploadArtifactsServer{stream})
}

type CirrusCIService_UploadArtifactsServer interface {
	SendAndClose(*UploadArtifactsResponse) error
	Recv() (*ArtifactEntry, error)
	grpc.ServerStream
}

type cirrusCIServiceUploadArtifactsServer struct {
	grpc.ServerStream
}

func (x *cirrusCIServiceUploadArtifactsServer) SendAndClose(m *UploadArtifactsResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *cirrusCIServiceUploadArtifactsServer) Recv() (*ArtifactEntry, error) {
	m := new(ArtifactEntry)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _CirrusCIService_DownloadCache_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(DownloadCacheRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(CirrusCIServiceServer).DownloadCache(m, &cirrusCIServiceDownloadCacheServer{stream})
}

type CirrusCIService_DownloadCacheServer interface {
	Send(*DataChunk) error
	grpc.ServerStream
}

type cirrusCIServiceDownloadCacheServer struct {
	grpc.ServerStream
}

func (x *cirrusCIServiceDownloadCacheServer) Send(m *DataChunk) error {
	return x.ServerStream.SendMsg(m)
}

func _CirrusCIService_CacheInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CacheInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CirrusCIServiceServer).CacheInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/CacheInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CirrusCIServiceServer).CacheInfo(ctx, req.(*CacheInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CirrusCIService_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(empty.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CirrusCIServiceServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/Ping",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CirrusCIServiceServer).Ping(ctx, req.(*empty.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _CirrusCIService_Heartbeat_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HeartbeatRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CirrusCIServiceServer).Heartbeat(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/Heartbeat",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CirrusCIServiceServer).Heartbeat(ctx, req.(*HeartbeatRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CirrusCIService_ReportStopHook_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReportStopHookRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CirrusCIServiceServer).ReportStopHook(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/ReportStopHook",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CirrusCIServiceServer).ReportStopHook(ctx, req.(*ReportStopHookRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CirrusCIService_ReportAgentError_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReportAgentProblemRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CirrusCIServiceServer).ReportAgentError(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/ReportAgentError",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CirrusCIServiceServer).ReportAgentError(ctx, req.(*ReportAgentProblemRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CirrusCIService_ReportAgentWarning_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReportAgentProblemRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CirrusCIServiceServer).ReportAgentWarning(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/ReportAgentWarning",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CirrusCIServiceServer).ReportAgentWarning(ctx, req.(*ReportAgentProblemRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CirrusCIService_ReportAgentSignal_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReportAgentSignalRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CirrusCIServiceServer).ReportAgentSignal(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/ReportAgentSignal",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CirrusCIServiceServer).ReportAgentSignal(ctx, req.(*ReportAgentSignalRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CirrusCIService_ReportAgentLogs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReportAgentLogsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CirrusCIServiceServer).ReportAgentLogs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/ReportAgentLogs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CirrusCIServiceServer).ReportAgentLogs(ctx, req.(*ReportAgentLogsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CirrusCIService_ParseConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ParseConfigRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CirrusCIServiceServer).ParseConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService/ParseConfig",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CirrusCIServiceServer).ParseConfig(ctx, req.(*ParseConfigRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _CirrusCIService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "org.cirruslabs.ci.services.cirruscigrpc.CirrusCIService",
	HandlerType: (*CirrusCIServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "InitialCommands",
			Handler:    _CirrusCIService_InitialCommands_Handler,
		},
		{
			MethodName: "ReportSingleCommand",
			Handler:    _CirrusCIService_ReportSingleCommand_Handler,
		},
		{
			MethodName: "ReportAnnotations",
			Handler:    _CirrusCIService_ReportAnnotations_Handler,
		},
		{
			MethodName: "CacheInfo",
			Handler:    _CirrusCIService_CacheInfo_Handler,
		},
		{
			MethodName: "Ping",
			Handler:    _CirrusCIService_Ping_Handler,
		},
		{
			MethodName: "Heartbeat",
			Handler:    _CirrusCIService_Heartbeat_Handler,
		},
		{
			MethodName: "ReportStopHook",
			Handler:    _CirrusCIService_ReportStopHook_Handler,
		},
		{
			MethodName: "ReportAgentError",
			Handler:    _CirrusCIService_ReportAgentError_Handler,
		},
		{
			MethodName: "ReportAgentWarning",
			Handler:    _CirrusCIService_ReportAgentWarning_Handler,
		},
		{
			MethodName: "ReportAgentSignal",
			Handler:    _CirrusCIService_ReportAgentSignal_Handler,
		},
		{
			MethodName: "ReportAgentLogs",
			Handler:    _CirrusCIService_ReportAgentLogs_Handler,
		},
		{
			MethodName: "ParseConfig",
			Handler:    _CirrusCIService_ParseConfig_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamLogs",
			Handler:       _CirrusCIService_StreamLogs_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "SaveLogs",
			Handler:       _CirrusCIService_SaveLogs_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "UploadCache",
			Handler:       _CirrusCIService_UploadCache_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "UploadArtifacts",
			Handler:       _CirrusCIService_UploadArtifacts_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "DownloadCache",
			Handler:       _CirrusCIService_DownloadCache_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "cirrus_ci_service.proto",
}
