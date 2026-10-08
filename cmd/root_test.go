// Copyright 2026 HEARTBEATS Corporation. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import "testing"

func TestThresholdFlagDefaults(t *testing.T) {
	for _, name := range []string{"warning", "critical"} {
		f := rootCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag %q not defined", name)
		}
		if f.Value.Type() != "float64" {
			t.Errorf("flag %q type = %q, want %q", name, f.Value.Type(), "float64")
		}
		if f.DefValue != "-1" {
			t.Errorf("flag %q default = %q, want %q", name, f.DefValue, "-1")
		}
	}
}

func TestThresholdFlagParse(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantWarning  float64
		wantCritical float64
		wantErr      bool
	}{
		{name: "decimal short flags", args: []string{"-w", "0.3", "-c", "0.8"}, wantWarning: 0.3, wantCritical: 0.8},
		{name: "decimal long flags", args: []string{"--warning=0.5", "--critical=1.5"}, wantWarning: 0.5, wantCritical: 1.5},
		{name: "integer values (backward compatible)", args: []string{"-w", "2", "-c", "5"}, wantWarning: 2, wantCritical: 5},
		{name: "invalid warning", args: []string{"-w", "abc"}, wantErr: true},
		{name: "invalid critical", args: []string{"-c", "1s"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts.WarningThreshold, opts.CriticalThreshold = -1, -1
			err := rootCmd.Flags().Parse(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse(%v) succeeded, want error", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%v) error = %v", tt.args, err)
			}
			if opts.WarningThreshold != tt.wantWarning {
				t.Errorf("WarningThreshold = %v, want %v", opts.WarningThreshold, tt.wantWarning)
			}
			if opts.CriticalThreshold != tt.wantCritical {
				t.Errorf("CriticalThreshold = %v, want %v", opts.CriticalThreshold, tt.wantCritical)
			}
		})
	}
}
