// Copyright 2026 HEARTBEATS Corporation. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package grpc provides utilities for parsing protocol buffer descriptors and invoking gRPC methods dynamically.
package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"os"
	"strings"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func ParseMethod(argMethod string) (packageName, serviceName, methodName string, err error) {
	parts := strings.Split(argMethod, "/")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid method format: expected 'package.Service/Method', got '%v'", argMethod)
	}
	serviceParts := strings.Split(parts[0], ".")
	if len(serviceParts) < 2 {
		return "", "", "", fmt.Errorf("invalid service format: expected 'package.Service'")
	}
	packageName = strings.Join(serviceParts[:len(serviceParts)-1], ".")
	serviceName = serviceParts[len(serviceParts)-1]
	methodName = parts[1]
	return packageName, serviceName, methodName, nil
}

type ProtoRegistryFiles []protoregistry.Files

func FindServiceDescriptor(regs ProtoRegistryFiles, packageName, serviceName string) (protoreflect.ServiceDescriptor, error) {
	fullName := protoreflect.FullName(packageName + "." + serviceName)

	var desc protoreflect.Descriptor
	for _, r := range regs {
		d, err := r.FindDescriptorByName(fullName)
		if err == nil {
			desc = d
			break
		}
	}
	if desc == nil {
		return nil, fmt.Errorf("can't find descriptor")
	}
	serviceDesc, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("found descriptor is not a service: %v", desc)
	}
	return serviceDesc, nil
}

func FindMethodDescriptor(serviceDesc protoreflect.ServiceDescriptor, methodName string) (protoreflect.MethodDescriptor, error) {
	methodDesc := serviceDesc.Methods().ByName(protoreflect.Name(methodName))
	if methodDesc == nil {
		return nil, fmt.Errorf("method %s not found", methodName)
	}
	return methodDesc, nil
}

func GetProtoRegistryFromFile(filePath string) (ProtoRegistryFiles, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	protoReg, err := GetProtoRegistry(file)
	return protoReg, err
}

func GetProtoRegistry(descriptorReader io.Reader) (ProtoRegistryFiles, error) {
	descriptorContent, err := io.ReadAll(descriptorReader)
	if err != nil {
		return nil, err
	}

	var fileDescriptorSet descriptorpb.FileDescriptorSet

	if err := proto.Unmarshal(descriptorContent, &fileDescriptorSet); err != nil {
		return nil, err
	}

	if len(fileDescriptorSet.File) == 0 {
		return nil, fmt.Errorf("descriptor set contains no files")
	}

	// protodesc.NewFiles resolves cross-file imports inside the set and
	// returns a single registry containing every file. This is required for
	// FileDescriptorSets produced by `protoc --include_imports`, where one
	// file may import another file from the same set.
	reg, err := protodesc.NewFiles(&fileDescriptorSet)
	if err != nil {
		return nil, err
	}
	return ProtoRegistryFiles{*reg}, nil
}

func NewRequestMessage(inputType protoreflect.MessageDescriptor, requestData string) (*dynamicpb.Message, error) {
	input := dynamicpb.NewMessage(inputType)
	if requestData != "" {
		if err := protojson.Unmarshal([]byte(requestData), input); err != nil {
			return nil, err
		}
	}
	return input, nil
}

func ConvertMetadata(rawMetadata []string) (metadata.MD, error) {
	metadataMap := make(map[string]string)
	for _, pair := range rawMetadata {
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) == 2 {
			metadataMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		} else {
			return nil, fmt.Errorf("invalid metadata format: '%s', expected 'key:value'", pair)
		}
	}
	return metadata.New(metadataMap), nil
}

func GetSystemCredentials() (credentials.TransportCredentials, error) {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{RootCAs: rootCAs}
	return credentials.NewTLS(tlsConfig), nil
}
