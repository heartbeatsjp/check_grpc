// Copyright 2026 HEARTBEATS Corporation. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grpc

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestParseMethod(t *testing.T) {
	tests := []struct {
		name        string
		argMethod   string
		wantPackage string
		wantService string
		wantMethod  string
		wantErr     bool
	}{
		{
			name:        "valid format",
			argMethod:   "package.Service/Method",
			wantPackage: "package",
			wantService: "Service",
			wantMethod:  "Method",
			wantErr:     false,
		},
		{
			name:        "invalid format - no slash",
			argMethod:   "package.Service.Method",
			wantPackage: "",
			wantService: "",
			wantMethod:  "",
			wantErr:     true,
		},
		{
			name:        "invalid format - multiple slashes",
			argMethod:   "package.Service/Method/Extra",
			wantPackage: "",
			wantService: "",
			wantMethod:  "",
			wantErr:     true,
		},
		{
			name:        "invalid service format - no dot",
			argMethod:   "Service/Method",
			wantPackage: "",
			wantService: "",
			wantMethod:  "",
			wantErr:     true,
		},
		{
			name:        "valid format - multiple package parts",
			argMethod:   "com.example.package.Service/Method",
			wantPackage: "com.example.package",
			wantService: "Service",
			wantMethod:  "Method",
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPackage, gotService, gotMethod, err := ParseMethod(tt.argMethod)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMethod() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotPackage != tt.wantPackage {
				t.Errorf("ParseMethod() gotPackage = %v, want %v", gotPackage, tt.wantPackage)
			}
			if gotService != tt.wantService {
				t.Errorf("ParseMethod() gotService = %v, want %v", gotService, tt.wantService)
			}
			if gotMethod != tt.wantMethod {
				t.Errorf("ParseMethod() gotMethod = %v, want %v", gotMethod, tt.wantMethod)
			}
		})
	}
}

func TestFindServiceDescriptor(t *testing.T) {
	regs, err := GetProtoRegistryFromFile("../testdata/test.pb")
	if err != nil {
		t.Fatalf("GetProtoRegistryFromFile() error = %v", err)
	}

	tests := []struct {
		name        string
		regs        ProtoRegistryFiles
		packageName string
		serviceName string
		wantFound   bool
		wantErr     bool
	}{
		{
			name:        "service found",
			regs:        regs,
			packageName: "com.example",
			serviceName: "MyService",
			wantFound:   true,
			wantErr:     false,
		},
		{
			name:        "service not found - wrong package",
			regs:        regs,
			packageName: "com.wrong",
			serviceName: "MyService",
			wantFound:   false,
			wantErr:     true,
		},
		{
			name:        "service not found - wrong service name",
			regs:        regs,
			packageName: "com.example",
			serviceName: "WrongService",
			wantFound:   false,
			wantErr:     true,
		},
		{
			name:        "empty registry",
			regs:        ProtoRegistryFiles{},
			packageName: "com.example",
			serviceName: "MyService",
			wantFound:   false,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindServiceDescriptor(tt.regs, tt.packageName, tt.serviceName)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindServiceDescriptor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantFound && got == nil {
				t.Errorf("FindServiceDescriptor() expected a service descriptor but got nil")
			}
			if !tt.wantFound && got != nil {
				t.Errorf("FindServiceDescriptor() expected nil but got a service descriptor: %v", got)
			}
		})
	}
}

func TestFindMethodDescriptor(t *testing.T) {
	regs, _ := GetProtoRegistryFromFile("../testdata/test.pb")
	serviceDesc, _ := FindServiceDescriptor(regs, "com.example", "MyService")

	tests := []struct {
		name        string
		serviceDesc protoreflect.ServiceDescriptor
		methodName  string
		wantFound   bool
		wantErr     bool
	}{
		{
			name:        "method found",
			serviceDesc: serviceDesc,
			methodName:  "MyMethod",
			wantFound:   true,
			wantErr:     false,
		},
		{
			name:        "method not found",
			serviceDesc: serviceDesc,
			methodName:  "WrongMethod",
			wantFound:   false,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindMethodDescriptor(tt.serviceDesc, tt.methodName)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindMethodDescriptor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantFound && got == nil {
				t.Errorf("FindMethodDescriptor() expected a method descriptor but got nil")
			}
			if !tt.wantFound && got != nil {
				t.Errorf("FindMethodDescriptor() expected nil but got a method descriptor: %v", got)
			}
		})
	}
}

func TestGetProtoRegistry(t *testing.T) {
	tests := []struct {
		name            string
		descriptorFile  string
		wantRegistryLen int
		wantErr         bool
	}{
		{
			name:            "valid descriptor file",
			descriptorFile:  "../testdata/test.pb",
			wantRegistryLen: 1,
			wantErr:         false,
		},
		{
			name:            "descriptor with cross-file imports",
			descriptorFile:  "../testdata/imports.pb",
			wantRegistryLen: 1,
			wantErr:         false,
		},
		{
			name:            "empty descriptor file",
			descriptorFile:  "../testdata/empty.pb",
			wantRegistryLen: 0,
			wantErr:         true,
		},
		{
			name:            "non-existent descriptor file",
			descriptorFile:  "../testdata/non_existent.pb",
			wantRegistryLen: 0,
			wantErr:         true,
		},
		{
			name:            "invalid proto content",
			descriptorFile:  "../testdata/invalid.pb",
			wantRegistryLen: 0,
			wantErr:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetProtoRegistryFromFile(tt.descriptorFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProtoRegistryFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantRegistryLen {
				t.Errorf("GetProtoRegistryFromFile() got length = %v, want %v", len(got), tt.wantRegistryLen)
			}
		})
	}
}

// TestGetProtoRegistry_CrossFileImports verifies that GetProtoRegistry can
// load a FileDescriptorSet whose files import each other (the shape produced
// by `protoc --include_imports`). The fixture testdata/imports.pb bundles
// service.proto (which imports common.proto) and uses com.example.common.User
// as the RPC input/output type, so resolving the method descriptor requires
// cross-file dependency resolution.
func TestGetProtoRegistry_CrossFileImports(t *testing.T) {
	regs, err := GetProtoRegistryFromFile("../testdata/imports.pb")
	if err != nil {
		t.Fatalf("GetProtoRegistryFromFile() error = %v", err)
	}

	svc, err := FindServiceDescriptor(regs, "com.example.imports", "UserService")
	if err != nil {
		t.Fatalf("FindServiceDescriptor() error = %v", err)
	}

	methodDesc, err := FindMethodDescriptor(svc, "GetUser")
	if err != nil {
		t.Fatalf("FindMethodDescriptor() error = %v", err)
	}

	if got, want := string(methodDesc.Input().FullName()), "com.example.common.User"; got != want {
		t.Errorf("methodDesc.Input().FullName() = %q, want %q", got, want)
	}
}

func TestNewRequestMessage(t *testing.T) {
	regs, _ := GetProtoRegistryFromFile("../testdata/test.pb")
	serviceDesc, _ := FindServiceDescriptor(regs, "com.example", "MyService")
	messageDesc, _ := FindMethodDescriptor(serviceDesc, "MyMethod")

	tests := []struct {
		name        string
		inputType   protoreflect.MessageDescriptor
		requestData string
		want        string
		wantErr     bool
	}{
		{
			name:        "empty request data",
			inputType:   messageDesc.Input(),
			requestData: "",
			want:        `{}`,
			wantErr:     false,
		},
		{
			name:        "valid request data",
			inputType:   messageDesc.Input(),
			requestData: `{"name": "test", "age": 30}`,
			want:        `{"name":"test","age":30}`,
			wantErr:     false,
		},
		{
			name:        "invalid request data",
			inputType:   messageDesc.Input(),
			requestData: `{"name": "test", "age": "invalid"}`,
			wantErr:     true,
		},
		{
			name:        "malformed json",
			inputType:   messageDesc.Input(),
			requestData: `{"name": "test"`,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRequestMessage(tt.inputType, tt.requestData)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRequestMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				gotJSON, err := protojson.Marshal(got)
				if err != nil {
					t.Fatalf("failed to marshal got message: %v", err)
				}
				var buf bytes.Buffer
				if err := json.Compact(&buf, gotJSON); err != nil {
					t.Fatalf("failed to compact JSON: %v", err)
				}
				if buf.String() != tt.want {
					t.Errorf("NewRequestMessage() got = %v, want %v", buf.String(), tt.want)
				}
			}
		})
	}
}

func TestConvertMetadata(t *testing.T) {
	tests := []struct {
		name        string
		rawMetadata []string
		want        metadata.MD
		wantErr     bool
	}{
		{
			name:        "empty metadata",
			rawMetadata: []string{},
			want:        metadata.MD{},
			wantErr:     false,
		},
		{
			name:        "single valid metadata",
			rawMetadata: []string{"key1:value1"},
			want:        metadata.Pairs("key1", "value1"),
			wantErr:     false,
		},
		{
			name:        "multiple valid metadata",
			rawMetadata: []string{"key1:value1", "key2:value2"},
			want:        metadata.Pairs("key1", "value1", "key2", "value2"),
			wantErr:     false,
		},
		{
			name:        "metadata with spaces",
			rawMetadata: []string{" key1 : value 1 "},
			want:        metadata.Pairs("key1", "value 1"),
			wantErr:     false,
		},
		{
			name:        "invalid metadata format - no colon",
			rawMetadata: []string{"key1value1"},
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "valid metadata with colon in value",
			rawMetadata: []string{"key1:value1:extra"},
			want:        metadata.Pairs("key1", "value1:extra"),
			wantErr:     false,
		},
		{
			name:        "authorization bearer token with colons",
			rawMetadata: []string{"authorization:Bearer eyJhbGciOi:JIUzI1NiJ9.abc"},
			want:        metadata.Pairs("authorization", "Bearer eyJhbGciOi:JIUzI1NiJ9.abc"),
			wantErr:     false,
		},
		{
			name:        "mixed valid and invalid metadata",
			rawMetadata: []string{"key1:value1", "invalid", "key2:value2"},
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "value containing commas (x-forwarded-for style)",
			rawMetadata: []string{"x-forwarded-for: 1.1.1.1, 2.2.2.2"},
			want:        metadata.Pairs("x-forwarded-for", "1.1.1.1, 2.2.2.2"),
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertMetadata(tt.rawMetadata)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertMetadata() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// The GetSystemCredentials function relies on the system's certificate pool,
// which can be difficult to mock or test reliably in a unit test.
// A simple test to ensure it doesn't panic or return an error might look like this:
func TestGetSystemCredentials(t *testing.T) {
	_, err := GetSystemCredentials()
	if err != nil {
		t.Errorf("GetSystemCredentials() error = %v", err)
	}
	// We can't easily verify the returned credentials,
	// as they depend on the system's configuration.
}
