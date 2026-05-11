// Copyright 2026 HEARTBEATS Corporation. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cmd implements the CLI interface for check_grpc.

package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/heartbeatsjp/check_grpc/internal"
)

var opts check.Options

var rootCmd = &cobra.Command{
	Use:   "check_grpc",
	Short: "Nagios Plugin for gRPC",
	Long:  `This plugin tests the gRPC service on the specified host. It can test normal (no TLS) and secure (TLS) servers, search for strings, check connection times.`,
	Run: func(cmd *cobra.Command, args []string) {
		res := check.Check(opts)
		res.ExitCmd()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// requirement
	rootCmd.Flags().StringVarP(&opts.Host, "host", "H", "", "Target gRPC endpoint (host:port) (required)")
	rootCmd.Flags().StringVarP(&opts.Method, "method", "m", "", "Target gRPC method (e.g., package.Service/Method) (required)")
	rootCmd.Flags().StringVarP(&opts.DescriptorSetFile, "descriptor_set_file", "D", "", "Path to the descriptor_set file (*.pb) (required)")
	rootCmd.MarkFlagRequired("host")
	rootCmd.MarkFlagRequired("method")
	rootCmd.MarkFlagRequired("descriptor_set_file")

	// Option
	rootCmd.Flags().BoolVarP(&opts.UseTLS, "secure", "S", false, "Use TLS/SSL for the gRPC connection")
	rootCmd.Flags().IntVarP(&opts.Timeout, "timeout", "t", 10, "Timeout for the gRPC call [sec]")
	rootCmd.Flags().IntVarP(&opts.WarningThreshold, "warning", "w", -1, "Warning threshold for response time [sec]")
	rootCmd.Flags().IntVarP(&opts.CriticalThreshold, "critical", "c", -1, "CRITICAL threshold for response time [sec]")
	rootCmd.Flags().StringVarP(&opts.ExpectedResponse, "string", "s", "", "String to expect in the content")
	rootCmd.Flags().StringVarP(&opts.RequestData, "argument", "a", "", "Request data in JSON format")
	rootCmd.Flags().IntVarP(&opts.ExpectedStatusCode, "expect_status_code", "e", 0, "gRPC Status code to expect in the content")
	rootCmd.Flags().BoolVarP(&opts.VerboseFlag, "verbose", "v", false, "Show details for command-line debugging")
	rootCmd.Flags().StringArrayVarP(&opts.RawMetadata, "metadata", "M", nil, "Metadata to be sent with gRPC request. Use multiple times for additional metadata")
}
