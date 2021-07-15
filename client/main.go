/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a client for Greeter service.
package main

import (
	helloworld "awesomeProject_grpc/pb"
	"context"
	"github.com/opentracing-contrib/go-grpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentracer"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"log"
	"os"
	"time"
)

const (
	address     = "localhost:9999"
	defaultName = "world"
)

func main() {
	// Step 1 - Setup tracing
	t := opentracer.New(tracer.WithAgentAddr("localhost:8126"), tracer.WithServiceName("test_client"), tracer.WithEnv("dev"))
	opentracing.SetGlobalTracer(t)

	// Set up a connection to the server.
	conn, err := grpc.Dial(
		address,
		grpc.WithInsecure(),
		grpc.WithBlock(),

		// Step 2 - setup tracing
		grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(t)),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := helloworld.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}

	for i := 0; i < 1000; i++ {

		// Step 3 - Create new context and span
		// This will capture client call >> server call >> "server sub_job span"
		span, ctx := opentracing.StartSpanFromContext(context.Background(), "client_call")
		ctx, cancel := context.WithTimeout(ctx, time.Second)

		r, err := c.SayHello(ctx, &helloworld.HelloRequest{Name: name})
		if err != nil {
			log.Fatalf("could not greet: %v", err)
		}
		log.Printf("Greeting: %s", r.GetMessage())
		time.Sleep(10 * time.Millisecond)

		cancel()
		span.Finish()
	}
}
