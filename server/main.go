package main

import (
	helloworld "awesomeProject_grpc/pb"
	"context"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/opentracing-contrib/go-grpc"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentracer"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"log"
	"net"
)
import "google.golang.org/grpc"

type server struct {
	helloworld.GreeterServer
}

func (s *server) SayHello(ctx context.Context, in *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	log.Printf("Received: %v", in.GetName())

	// Step 3 - add new spans (this span will come under the Grpc Call)
	span, _ := opentracing.StartSpanFromContext(ctx, "sub_job")
	defer span.Finish()

	return &helloworld.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func main() {
	// Step 1 - Setup a tracer
	t := opentracer.New(tracer.WithAgentAddr("localhost:8126"), tracer.WithServiceName("test"), tracer.WithEnv("dev"))
	opentracing.SetGlobalTracer(t)

	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	zapLogger, _ := loggerConfig.Build()

	lis, err := net.Listen("tcp", "localhost:9999")
	if err != nil {
		panic(err)
	}

	myServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_recovery.UnaryServerInterceptor(),
			grpc_ctxtags.UnaryServerInterceptor(),
			// Step 2 - register open tracing to handle tracing
			otgrpc.OpenTracingServerInterceptor(t),
			grpc_zap.UnaryServerInterceptor(zapLogger),
		)),
	)

	helloworld.RegisterGreeterServer(myServer, &server{})

	if err := myServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
