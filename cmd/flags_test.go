package cmd

import "testing"

func TestValidatePathFlagValue(t *testing.T) {
	tests := []struct {
		name    string
		flag    string
		value   string
		wantErr bool
	}{
		{name: "empty ok", flag: "--output", value: "", wantErr: false},
		{name: "stdout ok", flag: "--output", value: "-", wantErr: false},
		{name: "file ok", flag: "--output", value: "duplikate.csv", wantErr: false},
		{name: "short flag rejected", flag: "--output", value: "-i", wantErr: true},
		{name: "long flag rejected", flag: "--csv", value: "--dry-run", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePathFlagValue(tt.flag, tt.value)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
