package interceptor

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"google.golang.org/grpc"
	grpcmetadata "google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/operator-framework/operator-registry/pkg/api"
	"github.com/operator-framework/operator-registry/pkg/server"
)

func StreamSetCacheDigest(etag string) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !strings.HasPrefix(info.FullMethod, "/api.Registry/") {
			return handler(srv, stream)
		}

		outgoing := metadata.ExtractOutgoing(stream.Context()).Set(api.OLMETagKey, etag)
		stream.SetHeader(grpcmetadata.MD(outgoing))
		wrappedStream := middleware.WrapServerStream(stream)
		wrappedStream.WrappedContext = outgoing.ToOutgoing(stream.Context())
		handler(srv, wrappedStream)
		return nil
	}
}

func UnarySetCacheDigest(etag string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if !strings.HasPrefix(info.FullMethod, "/api.Registry/") {
			return handler(ctx, req)
		}

		outgoing := metadata.ExtractOutgoing(ctx)
		outgoing.Set(api.OLMETagKey, etag)
		ctx = outgoing.ToOutgoing(ctx)
		grpc.SetHeader(ctx, grpcmetadata.MD(outgoing))
		return handler(ctx, req)
	}
}

func StreamCache(etag string) grpc.StreamServerInterceptor {
	noopServer := &server.NoopServer{}
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !strings.HasPrefix(info.FullMethod, "/api.Registry/") {
			return handler(srv, stream)
		}
		if etag == metadata.ExtractIncoming(stream.Context()).Get(api.OLMIfNoneMatchKey) {
			srv = noopServer
		}
		handler(srv, stream)
		return nil
	}
}

func UnaryCache(etag string) grpc.UnaryServerInterceptor {
	respLookup := buildEmptyGRPCResponseMap(&server.RegistryServer{})
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if !strings.HasPrefix(info.FullMethod, "/api.Registry/") {
			return handler(ctx, req)
		}
		if etag == metadata.ExtractIncoming(ctx).Get(api.OLMIfNoneMatchKey) {
			if v, ok := respLookup[info.FullMethod]; ok {
				return v, nil
			}
		}
		return handler(ctx, req)
	}
}

// buildEmptyResponseMap generates a map of fully-qualified gRPC method names
// to empty response messages by inspecting a generated gRPC server implementation.
// This is necessary for unary responses only (stream responses can just _not_ send
// any messages)
//
// A pointer to your service server implementation (e.g., &MyServiceServer{}) MUST BE
// passed in. Also, the struct name MUST BE <grpcServiceName>Server.
// Otherwise, buildEmptyGRPCResponseMap will panic.
func buildEmptyGRPCResponseMap(serverImpl any) map[string]proto.Message {
	typ := reflect.TypeOf(serverImpl)
	if typ.Kind() != reflect.Ptr {
		panic(fmt.Errorf("expected pointer to server, got %s", typ.Kind()))
	}

	serviceName := strings.TrimSuffix(typ.Elem().Name(), "Server")
	// This assumes your generated service follows the Go protobuf naming convention
	var foundDesc protoreflect.ServiceDescriptor
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		services := fd.Services()

		for i := 0; i < services.Len(); i++ {
			sd := services.Get(i)
			if string(sd.Name()) == serviceName {
				foundDesc = sd
				return false // stop iteration
			}
		}
		return true // continue iteration
	})
	if foundDesc == nil {
		panic(fmt.Errorf("could not find service descriptor for %q", serviceName))
	}

	methodMap := map[string]proto.Message{}

	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		if m.Type.NumIn() != 3 || m.Type.NumOut() != 2 {
			continue // not a unary RPC method
		}
		retType := m.Type.Out(0) // return type: *YourResponse

		if retType.Kind() != reflect.Ptr {
			continue
		}
		resp, ok := reflect.New(retType.Elem()).Interface().(proto.Message)
		if !ok {
			continue
		}

		methodDesc := foundDesc.Methods().ByName(protoreflect.Name(m.Name))
		if methodDesc == nil {
			continue
		}
		fullMethodName := fmt.Sprintf("/%s.%s/%s",
			foundDesc.ParentFile().Package(),
			foundDesc.Name(),
			methodDesc.Name())

		methodMap[fullMethodName] = resp
	}

	return methodMap
}
