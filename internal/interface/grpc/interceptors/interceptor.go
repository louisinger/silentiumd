package interceptors

import (
	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
)

// UnaryInterceptor returns the unary interceptor
func UnaryInterceptor() grpc.ServerOption {
	return grpc.UnaryInterceptor(middleware.ChainUnaryServer(unaryLogger))
}
