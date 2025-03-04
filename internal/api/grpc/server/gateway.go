package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"

	client_middleware "github.com/zitadel/zitadel/internal/api/grpc/client/middleware"
	http_mw "github.com/zitadel/zitadel/internal/api/http/middleware"
)

const (
	mimeWildcard = "*/*"
)

var (
	customHeaders = []string{
		"x-zitadel-",
	}
	jsonMarshaler = &runtime.JSONPb{
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}

	serveMuxOptions = []runtime.ServeMuxOption{
		runtime.WithMarshalerOption(jsonMarshaler.ContentType(nil), jsonMarshaler),
		runtime.WithMarshalerOption(mimeWildcard, jsonMarshaler),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, jsonMarshaler),
		runtime.WithIncomingHeaderMatcher(headerMatcher),
		runtime.WithOutgoingHeaderMatcher(runtime.DefaultHeaderMatcher),
	}

	headerMatcher = runtime.HeaderMatcherFunc(
		func(header string) (string, bool) {
			for _, customHeader := range customHeaders {
				if strings.HasPrefix(strings.ToLower(header), customHeader) {
					return header, true
				}
			}
			return runtime.DefaultHeaderMatcher(header)
		},
	)
)

type Gateway interface {
	RegisterGateway() GatewayFunc
	GatewayPathPrefix() string
}

type GatewayFunc func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

func CreateGateway(ctx context.Context, g Gateway, port uint16) (http.Handler, string, error) {
	runtimeMux := runtime.NewServeMux(serveMuxOptions...)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(client_middleware.DefaultTracingClient()),
	}
	err := g.RegisterGateway()(ctx, runtimeMux, fmt.Sprintf("localhost:%d", port), opts)
	if err != nil {
		return nil, "", fmt.Errorf("failed to register grpc gateway: %w", err)
	}
	return addInterceptors(runtimeMux), g.GatewayPathPrefix(), nil
}

func addInterceptors(handler http.Handler) http.Handler {
	handler = http_mw.DefaultMetricsHandler(handler)
	handler = http_mw.DefaultTelemetryHandler(handler)
	return http_mw.CORSInterceptor(handler)
}
