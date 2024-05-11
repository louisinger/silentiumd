package grpcservice

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	silentiumv1 "github.com/louisinger/silentiumd/api/protobuf/gen/silentium/v1"
	"github.com/louisinger/silentiumd/internal/interface/grpc/handlers"
	"github.com/louisinger/silentiumd/internal/interface/grpc/interceptors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	grpchealth "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type Service struct {
	config Config
	server *http.Server
}

func NewService(
	svcConfig Config,
) (*Service, error) {
	if err := svcConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid service config: %s", err)
	}

	grpcConfig := []grpc.ServerOption{interceptors.UnaryInterceptor()}

	var tlsConfig *tls.Config

	if !svcConfig.insecure() {
		var err error
		tlsConfig, err = svcConfig.tlsConfig()
		if err != nil {
			return nil, err
		}
	}

	creds := insecure.NewCredentials()
	if !svcConfig.insecure() {
		creds = credentials.NewTLS(tlsConfig)
	}
	grpcConfig = append(grpcConfig, grpc.Creds(creds))

	// Server grpc.
	grpcServer := grpc.NewServer(grpcConfig...)
	appHandler := handlers.NewHandler(svcConfig.AppService)
	silentiumv1.RegisterSilentiumServiceServer(grpcServer, appHandler)
	healthHandler := handlers.NewHealthHandler()
	grpchealth.RegisterHealthServer(grpcServer, healthHandler)

	// Creds for grpc gateway reverse proxy.
	gatewayCreds := insecure.NewCredentials()
	if !svcConfig.insecure() {
		gatewayCreds = credentials.NewTLS(tlsConfig)
	}
	gatewayOpts := grpc.WithTransportCredentials(gatewayCreds)
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, svcConfig.gatewayAddress(), gatewayOpts,
	)
	if err != nil {
		return nil, err
	}
	// Reverse proxy grpc-gateway.
	gwmux := runtime.NewServeMux(
		runtime.WithHealthzEndpoint(grpchealth.NewHealthClient(conn)),
		runtime.WithMarshalerOption("application/json+pretty", &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				Indent:    "  ",
				Multiline: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	)
	if err := silentiumv1.RegisterSilentiumServiceHandler(
		ctx, gwmux, conn,
	); err != nil {
		return nil, err
	}
	grpcGateway := http.Handler(gwmux)

	handler := router(grpcServer, grpcGateway)
	mux := http.NewServeMux()
	mux.Handle("/", handler)

	httpServerHandler := http.Handler(mux)
	if svcConfig.insecure() {
		httpServerHandler = h2c.NewHandler(httpServerHandler, &http2.Server{})
	}

	server := &http.Server{
		Addr:      svcConfig.address(),
		Handler:   httpServerHandler,
		TLSConfig: tlsConfig,
	}

	return &Service{svcConfig, server}, nil
}

func (s *Service) Start() error {
	if s.config.insecure() {
		go s.server.ListenAndServe()
	} else {
		go s.server.ListenAndServeTLS("", "")
	}
	logrus.Infof("started listening at %s", s.config.address())

	return nil
}

func (s *Service) Stop() {
	if err := s.server.Shutdown(context.Background()); err != nil {
		logrus.Errorf("failed to stop grpc server: %s", err)
	}
	logrus.Info("stopped grpc server")
}

func router(
	grpcServer *grpc.Server, grpcGateway http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isOptionRequest(r) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			return
		}

		if isHttpRequest(r) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS")

			grpcGateway.ServeHTTP(w, r)
			return
		}
		grpcServer.ServeHTTP(w, r)
	})
}

func isOptionRequest(req *http.Request) bool {
	return req.Method == http.MethodOptions
}

func isHttpRequest(req *http.Request) bool {
	return req.Method == http.MethodGet ||
		strings.Contains(req.Header.Get("Content-Type"), "application/json")
}
