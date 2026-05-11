// Copyright 2026 HEARTBEATS Corporation. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nagios

import (
	"bytes"
	"io"
	"os"
	"sync"
	"testing"
)

func TestNewNagiosResult(t *testing.T) {
	tests := []struct {
		name    string
		status  NagiosStatus
		message string
	}{
		{"OK status", OK, "Everything is fine"},
		{"WARNING status", WARNING, "Something is a bit off"},
		{"CRITICAL status", CRITICAL, "Major problem!"},
		{"UNKNOWN status", UNKNOWN, "Cannot determine status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewNagiosResult(tt.status, tt.message)
			if result.Status != tt.status {
				t.Errorf("NewNagiosResult() Status = %v, want %v", result.Status, tt.status)
			}
			if result.Message != tt.message {
				t.Errorf("NewNagiosResult() Message = %q, want %q", result.Message, tt.message)
			}
		})
	}
}

// Variable to hold the original os.Exit function for monkey patching
var (
	exitCode    int
	exitCodeSet bool
	exitMutex   sync.Mutex
)

// Function to reset os.Exit to its original state after testing
func resetOsExit() {
	osExit = os.Exit
	exitMutex.Lock()
	exitCodeSet = false
	exitCode = 0
	exitMutex.Unlock()
}

// Mock function to replace os.Exit during testing
func fakeOsExit(code int) {
	exitMutex.Lock()
	exitCode = code
	exitCodeSet = true
	exitMutex.Unlock()
}

func TestNagiosResult_ExitCmd(t *testing.T) {
	tests := []struct {
		name             string
		result           NagiosResult
		expectedOutput   string
		expectedExitCode int
	}{
		{
			name: "OK status",
			result: NagiosResult{
				Status:  OK,
				Message: "All good",
			},
			expectedOutput:   "OK: All good\n",
			expectedExitCode: 0,
		},
		{
			name: "WARNING status",
			result: NagiosResult{
				Status:  WARNING,
				Message: "Something is a bit off",
			},
			expectedOutput:   "WARNING: Something is a bit off\n",
			expectedExitCode: 1,
		},
		{
			name: "CRITICAL status",
			result: NagiosResult{
				Status:  CRITICAL,
				Message: "Major problem!",
			},
			expectedOutput:   "CRITICAL: Major problem!\n",
			expectedExitCode: 2,
		},
		{
			name: "UNKNOWN status",
			result: NagiosResult{
				Status:  UNKNOWN,
				Message: "Cannot determine status",
			},
			expectedOutput:   "UNKNOWN: Cannot determine status\n",
			expectedExitCode: 3,
		},
		{
			name: "Unexpected status",
			result: NagiosResult{
				Status:  NagiosStatus(99), // Non-existent status
				Message: "Unexpected",
			},
			expectedOutput:   "UNKNOWN: unexpected status code 99: Unexpected\n",
			expectedExitCode: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replace os.Exit with fakeOsExit before each test
			osExit = fakeOsExit
			defer resetOsExit() // Ensure os.Exit is reset after each test

			// Capture standard output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			tt.result.ExitCmd()

			// Verify the captured output
			_ = w.Close()
			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			out := buf.String()
			if string(out) != tt.expectedOutput {
				t.Errorf("ExitCmd() printed incorrect output: got %q, want %q", string(out), tt.expectedOutput)
			}

			// Verify that osExit was called and the exit code is as expected
			exitMutex.Lock()
			exited := exitCodeSet
			actualExitCode := exitCode
			exitMutex.Unlock()

			if !exited {
				t.Errorf("ExitCmd() did not call osExit")
			}
			if actualExitCode != tt.expectedExitCode {
				t.Errorf("ExitCmd() called osExit with code %d, want %d", actualExitCode, tt.expectedExitCode)
			}
			os.Stdout = oldStdout
		})
	}
}
