package grpc

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"google.golang.org/grpc"
	grpcmd "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/futuramacoder/protopilot/internal/messages"
)

// dynamicCodec implements grpc.Codec for dynamicpb messages.
type dynamicCodec struct{}

func (dynamicCodec) Marshal(v any) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("not a proto.Message: %T", v)
	}
	return proto.Marshal(msg)
}

func (dynamicCodec) Unmarshal(data []byte, v any) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("not a proto.Message: %T", v)
	}
	return proto.Unmarshal(data, msg)
}

func (dynamicCodec) Name() string {
	return "proto"
}

// InvokeUnary performs a unary RPC call and returns a tea.Cmd.
func InvokeUnary(
	conn *grpc.ClientConn,
	method protoreflect.MethodDescriptor,
	req *dynamicpb.Message,
	md map[string]string,
) tea.Cmd {
	return func() tea.Msg {
		// Build full method path: /{package}.{service}/{method}
		svc := method.Parent().(protoreflect.ServiceDescriptor)
		fullMethod := fmt.Sprintf("/%s/%s", svc.FullName(), method.Name())

		// Build context with metadata.
		ctx := context.Background()
		if len(md) > 0 {
			pairs := make([]string, 0, len(md)*2)
			for k, v := range md {
				pairs = append(pairs, k, v)
			}
			ctx = grpcmd.NewOutgoingContext(ctx, grpcmd.Pairs(pairs...))
		}

		// Prepare response message.
		resp := dynamicpb.NewMessage(method.Output())

		// Capture headers and trailers.
		var headers, trailers grpcmd.MD

		// Measure latency.
		start := time.Now()

		err := conn.Invoke(
			ctx,
			fullMethod,
			req,
			resp,
			grpc.ForceCodec(dynamicCodec{}),
			grpc.Header(&headers),
			grpc.Trailer(&trailers),
		)
		latency := time.Since(start)

		if err != nil {
			st, _ := status.FromError(err)
			return messages.ResponseReceivedMsg{
				Status:   st,
				Latency:  latency,
				Headers:  headers,
				Trailers: trailers,
				Err:      err,
			}
		}

		// Marshal response to JSON.
		jsonBytes, jsonErr := protojson.MarshalOptions{
			Multiline: true,
			Indent:    "  ",
		}.Marshal(resp)

		if jsonErr != nil {
			return messages.ResponseReceivedMsg{
				Status:   status.New(0, ""),
				Latency:  latency,
				Headers:  headers,
				Trailers: trailers,
				Err:      fmt.Errorf("failed to marshal response: %w", jsonErr),
			}
		}

		st, _ := status.FromError(nil)
		return messages.ResponseReceivedMsg{
			Body:     jsonBytes,
			Status:   st,
			Latency:  latency,
			Headers:  headers,
			Trailers: trailers,
		}
	}
}
