// Copyright 2026 HEARTBEATS Corporation. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package check

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	"github.com/heartbeatsjp/check_grpc/nagios"
)

// testHealthServer implements grpc_health_v1.HealthServer for in-process testing.
type testHealthServer struct {
	grpc_health_v1.UnimplementedHealthServer
}

func (s *testHealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	if req.GetService() == "a" {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		}, nil
	}
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

func (s *testHealthServer) Watch(req *grpc_health_v1.HealthCheckRequest, srv grpc_health_v1.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "watch is not implemented.")
}

var testServerAddr string

func TestMain(m *testing.M) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	port := lis.Addr().(*net.TCPAddr).Port
	testServerAddr = fmt.Sprintf("localhost:%d", port)

	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, &testHealthServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	code := m.Run()
	s.GracefulStop()
	os.Exit(code)
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want nagios.NagiosResult
	}{
		{
			name: "Successful OK response",
			opts: Options{
				Host:                   testServerAddr,
				Method:              "grpc.health.v1.Health/Check",
				DescriptorSetFile:      "../testdata/sample.pb",
				RequestData:            "",
				ExpectedResponse:       `SERVING`,
				Timeout:           10,
				WarningThreshold:  -1,
				CriticalThreshold: -1,
			},
			want: nagios.NagiosResult{
				Status:  nagios.OK,
				Message: "gRPC 0 OK - invoke grpc.health.v1.Health/Check",
			},
		},
		{
			name: "CRITICAL on unexpected response content",
			opts: Options{
				Host:                   testServerAddr,
				Method:              "grpc.health.v1.Health/Check",
				DescriptorSetFile:      "../testdata/sample.pb",
				RequestData:            "",
				ExpectedResponse:       `BAD RESPONSE`,
				Timeout:           10,
				WarningThreshold:  -1,
				CriticalThreshold: -1,
			},
			want: nagios.NagiosResult{
				Status:  nagios.CRITICAL,
				Message: `does not contain BAD RESPONSE`,
			},
		},
		{
			name: "UNKNOWN on invalid method format",
			opts: Options{
				Host:                   testServerAddr,
				Method:              "invalid",
				DescriptorSetFile:      "../testdata/sample.pb",
				Timeout:           10,
				WarningThreshold:  -1,
				CriticalThreshold: -1,
			},
			want: nagios.NagiosResult{
				Status:  nagios.UNKNOWN,
				Message: "Invalid method format",
			},
		},
		{
			name: "UNKNOWN on non-existent descriptor file",
			opts: Options{
				Host:                   testServerAddr,
				Method:              "grpc.health.v1.Health/Check",
				DescriptorSetFile:      "../testdata/nonexistent.pb",
				Timeout:           10,
				WarningThreshold:  -1,
				CriticalThreshold: -1,
			},
			want: nagios.NagiosResult{
				Status:  nagios.UNKNOWN,
				Message: "Failed to open",
			},
		},
		{
			name: "CRITICAL on service not found",
			opts: Options{
				Host:                   testServerAddr,
				Method:              "wrong.package.Wrong/Check",
				DescriptorSetFile:      "../testdata/sample.pb",
				Timeout:           10,
				WarningThreshold:  -1,
				CriticalThreshold: -1,
			},
			want: nagios.NagiosResult{
				Status:  nagios.CRITICAL,
				Message: "Failed to find service",
			},
		},
		{
			name: "CRITICAL on method not found",
			opts: Options{
				Host:                   testServerAddr,
				Method:              "grpc.health.v1.Health/NonExistent",
				DescriptorSetFile:      "../testdata/sample.pb",
				Timeout:           10,
				WarningThreshold:  -1,
				CriticalThreshold: -1,
			},
			want: nagios.NagiosResult{
				Status:  nagios.CRITICAL,
				Message: "Method NonExistent not found",
			},
		},
		{
			name: "UNKNOWN on invalid metadata format",
			opts: Options{
				Host:                   testServerAddr,
				Method:              "grpc.health.v1.Health/Check",
				DescriptorSetFile:      "../testdata/sample.pb",
				Timeout:           10,
				WarningThreshold:  -1,
				CriticalThreshold: -1,
				RawMetadata:            []string{"invalid-no-colon"},
			},
			want: nagios.NagiosResult{
				Status:  nagios.UNKNOWN,
				Message: "Invalid metadata format",
			},
		},
		{
			name: "CRITICAL on critical threshold exceeded",
			opts: Options{
				Host:                   testServerAddr,
				Method:              "grpc.health.v1.Health/Check",
				DescriptorSetFile:      "../testdata/sample.pb",
				RequestData:            "",
				ExpectedResponse:       `SERVING`,
				Timeout:           10,
				WarningThreshold:  -1,
				CriticalThreshold: 0,
			},
			want: nagios.NagiosResult{
				Status:  nagios.CRITICAL,
				Message: "Response Time",
			},
		},
		{
			name: "WARNING on warning threshold exceeded",
			opts: Options{
				Host:                   testServerAddr,
				Method:              "grpc.health.v1.Health/Check",
				DescriptorSetFile:      "../testdata/sample.pb",
				RequestData:            "",
				ExpectedResponse:       `SERVING`,
				Timeout:           10,
				WarningThreshold:  0,
				CriticalThreshold: -1,
			},
			want: nagios.NagiosResult{
				Status:  nagios.WARNING,
				Message: "Response Time",
			},
		},
		{
			name: "CRITICAL on unexpected status code",
			opts: Options{
				Host:                   testServerAddr,
				Method:              "grpc.health.v1.Health/Check",
				DescriptorSetFile:      "../testdata/sample.pb",
				RequestData:            "",
				ExpectedResponse:       `SERVING`,
				ExpectedStatusCode:      99,
				Timeout:           10,
				WarningThreshold:  -1,
				CriticalThreshold: -1,
			},
			want: nagios.NagiosResult{
				Status:  nagios.CRITICAL,
				Message: "unexpected status code",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Check(tt.opts)

			if got.Status != tt.want.Status {
				t.Errorf("Check() status = %v, want %v", got.Status, tt.want.Status)
			}
			if !strings.Contains(got.Message, tt.want.Message) {
				t.Errorf("Check() message = %q, want substring %q", got.Message, tt.want.Message)
			}
		})
	}
}
