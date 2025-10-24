/*
Package pb is a reverse proxy.

It translates gRPC into RESTful JSON APIs.
*/
package pb

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/v2/utilities"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	_ codes.Code
	_ io.Reader
	_ status.Status
	_ = errors.New
	_ = runtime.String
	_ = utilities.NewDoubleArray
	_ = metadata.Join
)

func request_A2AService_SendMessage_0(ctx context.Context, marshaler runtime.Marshaler, client A2AServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var (
		protoReq SendMessageRequest
		metadata runtime.ServerMetadata
	)
	if err := marshaler.NewDecoder(req.Body).Decode(&protoReq); err != nil && !errors.Is(err, io.EOF) {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	if req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
	}
	msg, err := client.SendMessage(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err
}

func local_request_A2AService_SendMessage_0(ctx context.Context, marshaler runtime.Marshaler, server A2AServiceServer, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var (
		protoReq SendMessageRequest
		metadata runtime.ServerMetadata
	)
	if err := marshaler.NewDecoder(req.Body).Decode(&protoReq); err != nil && !errors.Is(err, io.EOF) {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	msg, err := server.SendMessage(ctx, &protoReq)
	return msg, metadata, err
}

func request_A2AService_SendStreamingMessage_0(ctx context.Context, marshaler runtime.Marshaler, client A2AServiceClient, req *http.Request, pathParams map[string]string) (A2AService_SendStreamingMessageClient, runtime.ServerMetadata, error) {
	var (
		protoReq SendMessageRequest
		metadata runtime.ServerMetadata
	)
	if err := marshaler.NewDecoder(req.Body).Decode(&protoReq); err != nil && !errors.Is(err, io.EOF) {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	if req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
	}
	stream, err := client.SendStreamingMessage(ctx, &protoReq)
	if err != nil {
		return nil, metadata, err
	}
	header, err := stream.Header()
	if err != nil {
		return nil, metadata, err
	}
	metadata.HeaderMD = header
	return stream, metadata, nil
}

var filter_A2AService_GetTask_0 = &utilities.DoubleArray{Encoding: map[string]int{"name": 0}, Base: []int{1, 1, 0}, Check: []int{0, 1, 2}}

func request_A2AService_GetTask_0(ctx context.Context, marshaler runtime.Marshaler, client A2AServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var (
		protoReq GetTaskRequest
		metadata runtime.ServerMetadata
		err      error
	)
	if req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
	}
	val, ok := pathParams["name"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "name")
	}
	protoReq.Name, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "name", err)
	}
	if err := req.ParseForm(); err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	if err := runtime.PopulateQueryParameters(&protoReq, req.Form, filter_A2AService_GetTask_0); err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	msg, err := client.GetTask(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err
}

func local_request_A2AService_GetTask_0(ctx context.Context, marshaler runtime.Marshaler, server A2AServiceServer, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var (
		protoReq GetTaskRequest
		metadata runtime.ServerMetadata
		err      error
	)
	val, ok := pathParams["name"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "name")
	}
	protoReq.Name, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "name", err)
	}
	if err := req.ParseForm(); err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	if err := runtime.PopulateQueryParameters(&protoReq, req.Form, filter_A2AService_GetTask_0); err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	msg, err := server.GetTask(ctx, &protoReq)
	return msg, metadata, err
}

func request_A2AService_CancelTask_0(ctx context.Context, marshaler runtime.Marshaler, client A2AServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var (
		protoReq CancelTaskRequest
		metadata runtime.ServerMetadata
		err      error
	)
	if err := marshaler.NewDecoder(req.Body).Decode(&protoReq); err != nil && !errors.Is(err, io.EOF) {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	if req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
	}
	val, ok := pathParams["name"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "name")
	}
	protoReq.Name, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "name", err)
	}
	msg, err := client.CancelTask(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err
}

func local_request_A2AService_CancelTask_0(ctx context.Context, marshaler runtime.Marshaler, server A2AServiceServer, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var (
		protoReq CancelTaskRequest
		metadata runtime.ServerMetadata
		err      error
	)
	if err := marshaler.NewDecoder(req.Body).Decode(&protoReq); err != nil && !errors.Is(err, io.EOF) {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	val, ok := pathParams["name"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "name")
	}
	protoReq.Name, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "name", err)
	}
	msg, err := server.CancelTask(ctx, &protoReq)
	return msg, metadata, err
}

func request_A2AService_TaskSubscription_0(ctx context.Context, marshaler runtime.Marshaler, client A2AServiceClient, req *http.Request, pathParams map[string]string) (A2AService_TaskSubscriptionClient, runtime.ServerMetadata, error) {
	var (
		protoReq TaskSubscriptionRequest
		metadata runtime.ServerMetadata
		err      error
	)
	if req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
	}
	val, ok := pathParams["name"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "name")
	}
	protoReq.Name, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "name", err)
	}
	stream, err := client.TaskSubscription(ctx, &protoReq)
	if err != nil {
		return nil, metadata, err
	}
	header, err := stream.Header()
	if err != nil {
		return nil, metadata, err
	}
	metadata.HeaderMD = header
	return stream, metadata, nil
}

var filter_A2AService_CreateTaskPushNotificationConfig_0 = &utilities.DoubleArray{Encoding: map[string]int{"config": 0, "parent": 1}, Base: []int{1, 1, 2, 0, 0}, Check: []int{0, 1, 1, 2, 3}}

func request_A2AService_CreateTaskPushNotificationConfig_0(ctx context.Context, marshaler runtime.Marshaler, client A2AServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var (
		protoReq CreateTaskPushNotificationConfigRequest
		metadata runtime.ServerMetadata
		err      error
	)
	if err := marshaler.NewDecoder(req.Body).Decode(&protoReq.Config); err != nil && !errors.Is(err, io.EOF) {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	if req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
	}
	val, ok := pathParams["parent"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "parent")
	}
	protoReq.Parent, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "parent", err)
	}
	if err := req.ParseForm(); err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	if err := runtime.PopulateQueryParameters(&protoReq, req.Form, filter_A2AService_CreateTaskPushNotificationConfig_0); err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	msg, err := client.CreateTaskPushNotificationConfig(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err
}

func local_request_A2AService_CreateTaskPushNotificationConfig_0(ctx context.Context, marshaler runtime.Marshaler, server A2AServiceServer, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var (
		protoReq CreateTaskPushNotificationConfigRequest
		metadata runtime.ServerMetadata
		err      error
	)
	if err := marshaler.NewDecoder(req.Body).Decode(&protoReq.Config); err != nil && !errors.Is(err, io.EOF) {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	val, ok := pathParams["parent"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "parent")
	}
	protoReq.Parent, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "parent", err)
	}
	if err := req.ParseForm(); err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	if err := runtime.PopulateQueryParameters(&protoReq, req.Form, filter_A2AService_CreateTaskPushNotificationConfig_0); err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	msg, err := server.CreateTaskPushNotificationConfig(ctx, &protoReq)
	return msg, metadata, err
}

func request_A2AService_GetTaskPushNotificationConfig_0(ctx context.Context, marshaler runtime.Marshaler, client A2AServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var (
		protoReq GetTaskPushNotificationConfigRequest
		metadata runtime.ServerMetadata
		err      error
	)
	if req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
	}
	val, ok := pathParams["name"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "name")
	}
	protoReq.Name, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "name", err)
	}
	msg, err := client.GetTaskPushNotificationConfig(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err
}

func local_request_A2AService_GetTaskPushNotificationConfig_0(ctx context.Context, marshaler runtime.Marshaler, server A2AServiceServer, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var (
		protoReq GetTaskPushNotificationConfigRequest
		metadata runtime.ServerMetadata
		err      error
	)
	val, ok := pathParams["name"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "name")
	}
	protoReq.Name, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "name", err)
	}
	msg, err := server.GetTaskPushNotificationConfig(ctx, &protoReq)
	return msg, metadata, err
}

var filter_A2AService_ListTaskPushNotificationConfig_0 = &utilities.DoubleArray{Encoding: map[string]int{"parent": 0}, Base: []int{1, 1, 0}, Check: []int{0, 1, 2}}

func request_A2AService_ListTaskPushNotificationConfig_0(ctx context.Context, marshaler runtime.Marshaler, client A2AServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var (
		protoReq ListTaskPushNotificationConfigRequest
		metadata runtime.ServerMetadata
		err      error
	)
	if req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
	}
	val, ok := pathParams["parent"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "parent")
	}
	protoReq.Parent, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "parent", err)
	}
	if err := req.ParseForm(); err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	if err := runtime.PopulateQueryParameters(&protoReq, req.Form, filter_A2AService_ListTaskPushNotificationConfig_0); err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	msg, err := client.ListTaskPushNotificationConfig(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err
}

func local_request_A2AService_ListTaskPushNotificationConfig_0(ctx context.Context, marshaler runtime.Marshaler, server A2AServiceServer, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var (
		protoReq ListTaskPushNotificationConfigRequest
		metadata runtime.ServerMetadata
		err      error
	)
	val, ok := pathParams["parent"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "parent")
	}
	protoReq.Parent, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "parent", err)
	}
	if err := req.ParseForm(); err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	if err := runtime.PopulateQueryParameters(&protoReq, req.Form, filter_A2AService_ListTaskPushNotificationConfig_0); err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	msg, err := server.ListTaskPushNotificationConfig(ctx, &protoReq)
	return msg, metadata, err
}

func request_A2AService_GetAgentCard_0(ctx context.Context, marshaler runtime.Marshaler, client A2AServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var (
		protoReq GetAgentCardRequest
		metadata runtime.ServerMetadata
	)
	if req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
	}
	msg, err := client.GetAgentCard(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err
}

func local_request_A2AService_GetAgentCard_0(ctx context.Context, marshaler runtime.Marshaler, server A2AServiceServer, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var (
		protoReq GetAgentCardRequest
		metadata runtime.ServerMetadata
	)
	msg, err := server.GetAgentCard(ctx, &protoReq)
	return msg, metadata, err
}

func request_A2AService_DeleteTaskPushNotificationConfig_0(ctx context.Context, marshaler runtime.Marshaler, client A2AServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var (
		protoReq DeleteTaskPushNotificationConfigRequest
		metadata runtime.ServerMetadata
		err      error
	)
	if req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
	}
	val, ok := pathParams["name"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "name")
	}
	protoReq.Name, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "name", err)
	}
	msg, err := client.DeleteTaskPushNotificationConfig(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err
}

func local_request_A2AService_DeleteTaskPushNotificationConfig_0(ctx context.Context, marshaler runtime.Marshaler, server A2AServiceServer, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var (
		protoReq DeleteTaskPushNotificationConfigRequest
		metadata runtime.ServerMetadata
		err      error
	)
	val, ok := pathParams["name"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "name")
	}
	protoReq.Name, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "name", err)
	}
	msg, err := server.DeleteTaskPushNotificationConfig(ctx, &protoReq)
	return msg, metadata, err
}

func RegisterA2AServiceHandlerServer(ctx context.Context, mux *runtime.ServeMux, server A2AServiceServer) error {
	mux.Handle(http.MethodPost, pattern_A2AService_SendMessage_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		var stream runtime.ServerTransportStream
		ctx = grpc.NewContextWithServerTransportStream(ctx, &stream)
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateIncomingContext(ctx, mux, req, "/a2a.v1.A2AService/SendMessage", runtime.WithHTTPPathPattern("/v1/message:send"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := local_request_A2AService_SendMessage_0(annotatedContext, inboundMarshaler, server, req, pathParams)
		md.HeaderMD, md.TrailerMD = metadata.Join(md.HeaderMD, stream.Header()), metadata.Join(md.TrailerMD, stream.Trailer())
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_SendMessage_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})

	mux.Handle(http.MethodPost, pattern_A2AService_SendStreamingMessage_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		err := status.Error(codes.Unimplemented, "streaming calls are not yet supported in the in-process transport")
		_, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
	})
	mux.Handle(http.MethodGet, pattern_A2AService_GetTask_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		var stream runtime.ServerTransportStream
		ctx = grpc.NewContextWithServerTransportStream(ctx, &stream)
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateIncomingContext(ctx, mux, req, "/a2a.v1.A2AService/GetTask", runtime.WithHTTPPathPattern("/v1/{name=tasks/*}"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := local_request_A2AService_GetTask_0(annotatedContext, inboundMarshaler, server, req, pathParams)
		md.HeaderMD, md.TrailerMD = metadata.Join(md.HeaderMD, stream.Header()), metadata.Join(md.TrailerMD, stream.Trailer())
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_GetTask_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})
	mux.Handle(http.MethodPost, pattern_A2AService_CancelTask_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		var stream runtime.ServerTransportStream
		ctx = grpc.NewContextWithServerTransportStream(ctx, &stream)
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateIncomingContext(ctx, mux, req, "/a2a.v1.A2AService/CancelTask", runtime.WithHTTPPathPattern("/v1/{name=tasks/*}:cancel"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := local_request_A2AService_CancelTask_0(annotatedContext, inboundMarshaler, server, req, pathParams)
		md.HeaderMD, md.TrailerMD = metadata.Join(md.HeaderMD, stream.Header()), metadata.Join(md.TrailerMD, stream.Trailer())
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_CancelTask_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})

	mux.Handle(http.MethodGet, pattern_A2AService_TaskSubscription_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		err := status.Error(codes.Unimplemented, "streaming calls are not yet supported in the in-process transport")
		_, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
	})
	mux.Handle(http.MethodPost, pattern_A2AService_CreateTaskPushNotificationConfig_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		var stream runtime.ServerTransportStream
		ctx = grpc.NewContextWithServerTransportStream(ctx, &stream)
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateIncomingContext(ctx, mux, req, "/a2a.v1.A2AService/CreateTaskPushNotificationConfig", runtime.WithHTTPPathPattern("/v1/{parent=tasks/*/pushNotificationConfigs}"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := local_request_A2AService_CreateTaskPushNotificationConfig_0(annotatedContext, inboundMarshaler, server, req, pathParams)
		md.HeaderMD, md.TrailerMD = metadata.Join(md.HeaderMD, stream.Header()), metadata.Join(md.TrailerMD, stream.Trailer())
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_CreateTaskPushNotificationConfig_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})
	mux.Handle(http.MethodGet, pattern_A2AService_GetTaskPushNotificationConfig_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		var stream runtime.ServerTransportStream
		ctx = grpc.NewContextWithServerTransportStream(ctx, &stream)
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateIncomingContext(ctx, mux, req, "/a2a.v1.A2AService/GetTaskPushNotificationConfig", runtime.WithHTTPPathPattern("/v1/{name=tasks/*/pushNotificationConfigs/*}"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := local_request_A2AService_GetTaskPushNotificationConfig_0(annotatedContext, inboundMarshaler, server, req, pathParams)
		md.HeaderMD, md.TrailerMD = metadata.Join(md.HeaderMD, stream.Header()), metadata.Join(md.TrailerMD, stream.Trailer())
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_GetTaskPushNotificationConfig_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})
	mux.Handle(http.MethodGet, pattern_A2AService_ListTaskPushNotificationConfig_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		var stream runtime.ServerTransportStream
		ctx = grpc.NewContextWithServerTransportStream(ctx, &stream)
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateIncomingContext(ctx, mux, req, "/a2a.v1.A2AService/ListTaskPushNotificationConfig", runtime.WithHTTPPathPattern("/v1/{parent=tasks/*}/pushNotificationConfigs"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := local_request_A2AService_ListTaskPushNotificationConfig_0(annotatedContext, inboundMarshaler, server, req, pathParams)
		md.HeaderMD, md.TrailerMD = metadata.Join(md.HeaderMD, stream.Header()), metadata.Join(md.TrailerMD, stream.Trailer())
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_ListTaskPushNotificationConfig_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})
	mux.Handle(http.MethodGet, pattern_A2AService_GetAgentCard_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		var stream runtime.ServerTransportStream
		ctx = grpc.NewContextWithServerTransportStream(ctx, &stream)
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateIncomingContext(ctx, mux, req, "/a2a.v1.A2AService/GetAgentCard", runtime.WithHTTPPathPattern("/v1/card"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := local_request_A2AService_GetAgentCard_0(annotatedContext, inboundMarshaler, server, req, pathParams)
		md.HeaderMD, md.TrailerMD = metadata.Join(md.HeaderMD, stream.Header()), metadata.Join(md.TrailerMD, stream.Trailer())
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_GetAgentCard_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})
	mux.Handle(http.MethodDelete, pattern_A2AService_DeleteTaskPushNotificationConfig_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		var stream runtime.ServerTransportStream
		ctx = grpc.NewContextWithServerTransportStream(ctx, &stream)
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateIncomingContext(ctx, mux, req, "/a2a.v1.A2AService/DeleteTaskPushNotificationConfig", runtime.WithHTTPPathPattern("/v1/{name=tasks/*/pushNotificationConfigs/*}"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := local_request_A2AService_DeleteTaskPushNotificationConfig_0(annotatedContext, inboundMarshaler, server, req, pathParams)
		md.HeaderMD, md.TrailerMD = metadata.Join(md.HeaderMD, stream.Header()), metadata.Join(md.TrailerMD, stream.Trailer())
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_DeleteTaskPushNotificationConfig_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})

	return nil
}

func RegisterA2AServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) (err error) {
	conn, err := grpc.NewClient(endpoint, opts...)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if cerr := conn.Close(); cerr != nil {
				grpclog.Errorf("Failed to close conn to %s: %v", endpoint, cerr)
			}
			return
		}
		go func() {
			<-ctx.Done()
			if cerr := conn.Close(); cerr != nil {
				grpclog.Errorf("Failed to close conn to %s: %v", endpoint, cerr)
			}
		}()
	}()
	return RegisterA2AServiceHandler(ctx, mux, conn)
}

func RegisterA2AServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return RegisterA2AServiceHandlerClient(ctx, mux, NewA2AServiceClient(conn))
}

func RegisterA2AServiceHandlerClient(ctx context.Context, mux *runtime.ServeMux, client A2AServiceClient) error {
	mux.Handle(http.MethodPost, pattern_A2AService_SendMessage_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateContext(ctx, mux, req, "/a2a.v1.A2AService/SendMessage", runtime.WithHTTPPathPattern("/v1/message:send"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := request_A2AService_SendMessage_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_SendMessage_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})
	mux.Handle(http.MethodPost, pattern_A2AService_SendStreamingMessage_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateContext(ctx, mux, req, "/a2a.v1.A2AService/SendStreamingMessage", runtime.WithHTTPPathPattern("/v1/message:stream"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := request_A2AService_SendStreamingMessage_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_SendStreamingMessage_0(annotatedContext, mux, outboundMarshaler, w, req, func() (proto.Message, error) { return resp.Recv() }, mux.GetForwardResponseOptions()...)
	})
	mux.Handle(http.MethodGet, pattern_A2AService_GetTask_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateContext(ctx, mux, req, "/a2a.v1.A2AService/GetTask", runtime.WithHTTPPathPattern("/v1/{name=tasks/*}"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := request_A2AService_GetTask_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_GetTask_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})
	mux.Handle(http.MethodPost, pattern_A2AService_CancelTask_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateContext(ctx, mux, req, "/a2a.v1.A2AService/CancelTask", runtime.WithHTTPPathPattern("/v1/{name=tasks/*}:cancel"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := request_A2AService_CancelTask_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_CancelTask_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})
	mux.Handle(http.MethodGet, pattern_A2AService_TaskSubscription_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateContext(ctx, mux, req, "/a2a.v1.A2AService/TaskSubscription", runtime.WithHTTPPathPattern("/v1/{name=tasks/*}:subscribe"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := request_A2AService_TaskSubscription_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_TaskSubscription_0(annotatedContext, mux, outboundMarshaler, w, req, func() (proto.Message, error) { return resp.Recv() }, mux.GetForwardResponseOptions()...)
	})
	mux.Handle(http.MethodPost, pattern_A2AService_CreateTaskPushNotificationConfig_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateContext(ctx, mux, req, "/a2a.v1.A2AService/CreateTaskPushNotificationConfig", runtime.WithHTTPPathPattern("/v1/{parent=tasks/*/pushNotificationConfigs}"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := request_A2AService_CreateTaskPushNotificationConfig_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_CreateTaskPushNotificationConfig_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})
	mux.Handle(http.MethodGet, pattern_A2AService_GetTaskPushNotificationConfig_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateContext(ctx, mux, req, "/a2a.v1.A2AService/GetTaskPushNotificationConfig", runtime.WithHTTPPathPattern("/v1/{name=tasks/*/pushNotificationConfigs/*}"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := request_A2AService_GetTaskPushNotificationConfig_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_GetTaskPushNotificationConfig_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})
	mux.Handle(http.MethodGet, pattern_A2AService_ListTaskPushNotificationConfig_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateContext(ctx, mux, req, "/a2a.v1.A2AService/ListTaskPushNotificationConfig", runtime.WithHTTPPathPattern("/v1/{parent=tasks/*}/pushNotificationConfigs"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := request_A2AService_ListTaskPushNotificationConfig_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_ListTaskPushNotificationConfig_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})
	mux.Handle(http.MethodGet, pattern_A2AService_GetAgentCard_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateContext(ctx, mux, req, "/a2a.v1.A2AService/GetAgentCard", runtime.WithHTTPPathPattern("/v1/card"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := request_A2AService_GetAgentCard_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_GetAgentCard_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})
	mux.Handle(http.MethodDelete, pattern_A2AService_DeleteTaskPushNotificationConfig_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateContext(ctx, mux, req, "/a2a.v1.A2AService/DeleteTaskPushNotificationConfig", runtime.WithHTTPPathPattern("/v1/{name=tasks/*/pushNotificationConfigs/*}"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := request_A2AService_DeleteTaskPushNotificationConfig_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		forward_A2AService_DeleteTaskPushNotificationConfig_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
	})
	return nil
}

var (
	pattern_A2AService_SendMessage_0                      = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1}, []string{"v1", "message"}, "send"))
	pattern_A2AService_SendStreamingMessage_0             = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1}, []string{"v1", "message"}, "stream"))
	pattern_A2AService_GetTask_0                          = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 1, 0, 4, 2, 5, 2}, []string{"v1", "tasks", "name"}, ""))
	pattern_A2AService_CancelTask_0                       = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 1, 0, 4, 2, 5, 2}, []string{"v1", "tasks", "name"}, "cancel"))
	pattern_A2AService_TaskSubscription_0                 = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 1, 0, 4, 2, 5, 2}, []string{"v1", "tasks", "name"}, "subscribe"))
	pattern_A2AService_CreateTaskPushNotificationConfig_0 = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 1, 0, 2, 2, 4, 3, 5, 3}, []string{"v1", "tasks", "pushNotificationConfigs", "parent"}, ""))
	pattern_A2AService_GetTaskPushNotificationConfig_0    = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 1, 0, 2, 2, 1, 0, 4, 4, 5, 3}, []string{"v1", "tasks", "pushNotificationConfigs", "name"}, ""))
	pattern_A2AService_ListTaskPushNotificationConfig_0   = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 1, 0, 4, 2, 5, 2, 2, 3}, []string{"v1", "tasks", "parent", "pushNotificationConfigs"}, ""))
	pattern_A2AService_GetAgentCard_0                     = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1}, []string{"v1", "card"}, ""))
	pattern_A2AService_DeleteTaskPushNotificationConfig_0 = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 1, 0, 2, 2, 1, 0, 4, 4, 5, 3}, []string{"v1", "tasks", "pushNotificationConfigs", "name"}, ""))
)

var (
	forward_A2AService_SendMessage_0                      = runtime.ForwardResponseMessage
	forward_A2AService_SendStreamingMessage_0             = runtime.ForwardResponseStream
	forward_A2AService_GetTask_0                          = runtime.ForwardResponseMessage
	forward_A2AService_CancelTask_0                       = runtime.ForwardResponseMessage
	forward_A2AService_TaskSubscription_0                 = runtime.ForwardResponseStream
	forward_A2AService_CreateTaskPushNotificationConfig_0 = runtime.ForwardResponseMessage
	forward_A2AService_GetTaskPushNotificationConfig_0    = runtime.ForwardResponseMessage
	forward_A2AService_ListTaskPushNotificationConfig_0   = runtime.ForwardResponseMessage
	forward_A2AService_GetAgentCard_0                     = runtime.ForwardResponseMessage
	forward_A2AService_DeleteTaskPushNotificationConfig_0 = runtime.ForwardResponseMessage
)
