// Copyright 2026 HEARTBEATS Corporation. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package check implements the core monitoring logic for the check_grpc Nagios plugin.
package check

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/dynamicpb"

	check_grpc "github.com/heartbeatsjp/check_grpc/grpc"
	"github.com/heartbeatsjp/check_grpc/nagios"
)

type Options struct {
	Host              string
	Method            string
	DescriptorSetFile string
	UseTLS            bool
	Timeout           int
	RequestData       string
	WarningThreshold  int
	CriticalThreshold int
	ExpectedResponse  string
	ExpectedStatusCode int
	RawMetadata       []string
	VerboseFlag       bool
}

func Check(opts Options) nagios.NagiosResult {
	timeout := time.Duration(opts.Timeout) * time.Second
	warningThresholdTime := time.Duration(opts.WarningThreshold) * time.Second
	criticalThresholdTime := time.Duration(opts.CriticalThreshold) * time.Second

	var credential credentials.TransportCredentials
	var err error
	if opts.UseTLS {
		credential, err = check_grpc.GetSystemCredentials()
		if err != nil {
			return nagios.NewNagiosResult(nagios.UNKNOWN, fmt.Sprintf("Failed to load system certificate pool: %v", err))
		}
	} else {
		credential = insecure.NewCredentials()
	}

	packageName, serviceName, methodName, err := check_grpc.ParseMethod(opts.Method)
	if err != nil {
		return nagios.NewNagiosResult(nagios.UNKNOWN, fmt.Sprintf("Invalid method format: %v", err))
	}
	methodFullName := fmt.Sprintf("/%s.%s/%s", packageName, serviceName, methodName)

	regs, err := check_grpc.GetProtoRegistryFromFile(opts.DescriptorSetFile)
	if err != nil {
		return nagios.NewNagiosResult(nagios.UNKNOWN, fmt.Sprintf("Failed to open %s: %v", opts.DescriptorSetFile, err))
	}

	serviceDesc, err := check_grpc.FindServiceDescriptor(regs, packageName, serviceName)
	if err != nil {
		return nagios.NewNagiosResult(nagios.CRITICAL, fmt.Sprintf("Failed to find service %s.%s: %v", packageName, serviceName, err))
	}

	methodDesc, err := check_grpc.FindMethodDescriptor(serviceDesc, methodName)
	if err != nil {
		return nagios.NewNagiosResult(nagios.CRITICAL, fmt.Sprintf("Method %s not found in service %s.%s", methodName, packageName, serviceName))
	}

	inputType := methodDesc.Input()
	outputType := methodDesc.Output()
	input, err := check_grpc.NewRequestMessage(inputType, opts.RequestData)
	if err != nil {
		return nagios.NewNagiosResult(nagios.UNKNOWN, fmt.Sprintf("Failed to create request input: %v", err))
	}
	output := dynamicpb.NewMessage(outputType)

	client, err := grpc.NewClient(opts.Host, grpc.WithTransportCredentials(credential))
	if err != nil {
		return nagios.NewNagiosResult(nagios.UNKNOWN, fmt.Sprintf("Failed to create gRPC client: %v", err))
	}
	defer func() { _ = client.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if len(opts.RawMetadata) > 0 {
		md, err := check_grpc.ConvertMetadata(opts.RawMetadata)
		if err != nil {
			return nagios.NewNagiosResult(nagios.UNKNOWN, fmt.Sprintf("Invalid metadata format, %v", err))
		}
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	if opts.VerboseFlag {
		fmt.Printf("Request: %v\n", opts.RequestData)
	}

	startTime := time.Now()
	invokeErr := client.Invoke(ctx, methodFullName, input, output)
	responseTime := time.Since(startTime)

	statusCode := 0
	if invokeErr != nil {
		if st, ok := status.FromError(invokeErr); ok {
			statusCode = int(st.Code())
		} else {
			// For errors not expressed by gRPC status code
			statusCode = -1
		}
	}

	if statusCode == -1 {
		return nagios.NewNagiosResult(nagios.CRITICAL, fmt.Sprintf("gRPC call failed: %v", invokeErr))
	}

	outputContent := output.String()
	if opts.VerboseFlag {
		fmt.Printf("%v\n", outputContent)
	}
	if opts.CriticalThreshold != -1 && responseTime > criticalThresholdTime {
		return nagios.NewNagiosResult(nagios.CRITICAL, fmt.Sprintf("Response Time: %s > %s", responseTime, criticalThresholdTime))
	}
	if opts.WarningThreshold != -1 && responseTime > warningThresholdTime {
		return nagios.NewNagiosResult(nagios.WARNING, fmt.Sprintf("Response Time: %s > %s", responseTime, warningThresholdTime))
	}
	if !strings.Contains(outputContent, opts.ExpectedResponse) {
		return nagios.NewNagiosResult(nagios.CRITICAL, fmt.Sprintf("does not contain %s", opts.ExpectedResponse))
	}

	if statusCode != opts.ExpectedStatusCode {
		return nagios.NewNagiosResult(nagios.CRITICAL, fmt.Sprintf("gRPC call failed or returned unexpected status code. Expected: %d, Actual: %d (Response Time: %s)", opts.ExpectedStatusCode, statusCode, responseTime))
	}

	return nagios.NewNagiosResult(nagios.OK, fmt.Sprintf("gRPC %v OK - invoke %s, %s response time", statusCode, opts.Method, responseTime))

}
