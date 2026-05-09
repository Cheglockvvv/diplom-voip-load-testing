package control

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/status"
)

const serviceName = "/control.ControlService"

func init() {
	encoding.RegisterCodec(JSONCodec{})
}

type JSONCodec struct{}

func (JSONCodec) Name() string { return "json" }

func (JSONCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (JSONCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

type StartScenarioRequest struct {
	ScenarioJSON string `json:"scenario_json"`
}

type StartScenarioResponse struct {
	RunID   string `json:"run_id"`
	Message string `json:"message"`
}

type StopScenarioRequest struct {
	RunID string `json:"run_id,omitempty"`
}

type StopScenarioResponse struct {
	RunID   string `json:"run_id,omitempty"`
	Message string `json:"message"`
}

type StatusRequest struct {
	RunID string `json:"run_id,omitempty"`
}

type StatusResponse struct {
	RunID   string `json:"run_id,omitempty"`
	State   string `json:"state"`
	Message string `json:"message,omitempty"`
}

type StreamStatusRequest struct {
	RunID string `json:"run_id,omitempty"`
}

type StatusEvent struct {
	RunID    string `json:"run_id,omitempty"`
	State    string `json:"state"`
	Details  string `json:"details,omitempty"`
	UnixTime int64  `json:"unix_time"`
}

type ControlServiceClient interface {
	StartScenario(ctx context.Context, in *StartScenarioRequest, opts ...grpc.CallOption) (*StartScenarioResponse, error)
	StopScenario(ctx context.Context, in *StopScenarioRequest, opts ...grpc.CallOption) (*StopScenarioResponse, error)
	GetStatus(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error)
	StreamStatus(ctx context.Context, in *StreamStatusRequest, opts ...grpc.CallOption) (ControlService_StreamStatusClient, error)
}

type controlServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewControlServiceClient(cc grpc.ClientConnInterface) ControlServiceClient {
	return &controlServiceClient{cc: cc}
}

func (c *controlServiceClient) StartScenario(ctx context.Context, in *StartScenarioRequest, opts ...grpc.CallOption) (*StartScenarioResponse, error) {
	out := new(StartScenarioResponse)
	err := c.cc.Invoke(ctx, serviceName+"/StartScenario", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlServiceClient) StopScenario(ctx context.Context, in *StopScenarioRequest, opts ...grpc.CallOption) (*StopScenarioResponse, error) {
	out := new(StopScenarioResponse)
	err := c.cc.Invoke(ctx, serviceName+"/StopScenario", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlServiceClient) GetStatus(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error) {
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, serviceName+"/GetStatus", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlServiceClient) StreamStatus(ctx context.Context, in *StreamStatusRequest, opts ...grpc.CallOption) (ControlService_StreamStatusClient, error) {
	stream, err := c.cc.NewStream(ctx, &ControlService_ServiceDesc.Streams[0], serviceName+"/StreamStatus", opts...)
	if err != nil {
		return nil, err
	}
	x := &controlServiceStreamStatusClient{stream}
	if err := x.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type ControlService_StreamStatusClient interface {
	Recv() (*StatusEvent, error)
	grpc.ClientStream
}

type controlServiceStreamStatusClient struct {
	grpc.ClientStream
}

func (x *controlServiceStreamStatusClient) Recv() (*StatusEvent, error) {
	m := new(StatusEvent)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type ControlServiceServer interface {
	StartScenario(context.Context, *StartScenarioRequest) (*StartScenarioResponse, error)
	StopScenario(context.Context, *StopScenarioRequest) (*StopScenarioResponse, error)
	GetStatus(context.Context, *StatusRequest) (*StatusResponse, error)
	StreamStatus(*StreamStatusRequest, ControlService_StreamStatusServer) error
}

type UnimplementedControlServiceServer struct{}

func (UnimplementedControlServiceServer) StartScenario(context.Context, *StartScenarioRequest) (*StartScenarioResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StartScenario not implemented")
}
func (UnimplementedControlServiceServer) StopScenario(context.Context, *StopScenarioRequest) (*StopScenarioResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StopScenario not implemented")
}
func (UnimplementedControlServiceServer) GetStatus(context.Context, *StatusRequest) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetStatus not implemented")
}
func (UnimplementedControlServiceServer) StreamStatus(*StreamStatusRequest, ControlService_StreamStatusServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamStatus not implemented")
}

func RegisterControlServiceServer(s grpc.ServiceRegistrar, srv ControlServiceServer) {
	s.RegisterService(&ControlService_ServiceDesc, srv)
}

type ControlService_StreamStatusServer interface {
	Send(*StatusEvent) error
	grpc.ServerStream
}

type controlServiceStreamStatusServer struct {
	grpc.ServerStream
}

func (x *controlServiceStreamStatusServer) Send(m *StatusEvent) error {
	return x.ServerStream.SendMsg(m)
}

func _ControlService_StartScenario_Handler(srv interface{}, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(StartScenarioRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServiceServer).StartScenario(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: serviceName + "/StartScenario",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(ControlServiceServer).StartScenario(ctx, req.(*StartScenarioRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ControlService_StopScenario_Handler(srv interface{}, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(StopScenarioRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServiceServer).StopScenario(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: serviceName + "/StopScenario",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(ControlServiceServer).StopScenario(ctx, req.(*StopScenarioRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ControlService_GetStatus_Handler(srv interface{}, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(StatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlServiceServer).GetStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: serviceName + "/GetStatus",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(ControlServiceServer).GetStatus(ctx, req.(*StatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ControlService_StreamStatus_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(StreamStatusRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ControlServiceServer).StreamStatus(m, &controlServiceStreamStatusServer{stream})
}

var ControlService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "control.ControlService",
	HandlerType: (*ControlServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "StartScenario",
			Handler:    _ControlService_StartScenario_Handler,
		},
		{
			MethodName: "StopScenario",
			Handler:    _ControlService_StopScenario_Handler,
		},
		{
			MethodName: "GetStatus",
			Handler:    _ControlService_GetStatus_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamStatus",
			Handler:       _ControlService_StreamStatus_Handler,
			ServerStreams: true,
		},
	},
}
