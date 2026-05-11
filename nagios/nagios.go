// Copyright 2026 HEARTBEATS Corporation. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package nagios defines result types and exit behavior for Nagios-compatible monitoring plugins.
package nagios

import (
	"fmt"
	"os"
)

type NagiosStatus int

const (
	OK       NagiosStatus = 0
	WARNING  NagiosStatus = 1
	CRITICAL NagiosStatus = 2
	UNKNOWN  NagiosStatus = 3
)

type NagiosResult struct {
	Status  NagiosStatus
	Message string
}

func NewNagiosResult(status NagiosStatus, message string) NagiosResult {
	return NagiosResult{
		Status:  status,
		Message: message,
	}
}

// for test
var osExit = os.Exit

func (res NagiosResult) ExitCmd() {
	switch res.Status {
	case OK:
		fmt.Printf("OK: %s\n", res.Message)
		osExit(int(OK))

	case WARNING:
		fmt.Printf("WARNING: %s\n", res.Message)
		osExit(int(WARNING))

	case CRITICAL:
		fmt.Printf("CRITICAL: %s\n", res.Message)
		osExit(int(CRITICAL))

	case UNKNOWN:
		fmt.Printf("UNKNOWN: %s\n", res.Message)
		osExit(int(UNKNOWN))
	default:
		fmt.Printf("UNKNOWN: unexpected status code %d: %s\n", res.Status, res.Message)
		osExit(int(UNKNOWN))
	}
}
