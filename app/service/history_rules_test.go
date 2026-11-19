package service

import (
	"errors"
	"testing"
)

func TestParseStatusFilter(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantErr   error
		expectErr bool
	}{
		{
			name:      "string kosong berhasil tanpa filter",
			input:     "",
			want:      "",
			expectErr: false,
		},
		{
			name:      "hanya spasi berhasil tanpa filter",
			input:     "   ",
			want:      "",
			expectErr: false,
		},
		{
			name:      "status approved huruf kecil",
			input:     "approved",
			want:      "approved",
			expectErr: false,
		},
		{
			name:      "status APPROVED huruf besar dinormalkan",
			input:     "APPROVED",
			want:      "approved",
			expectErr: false,
		},
		{
			name:      "status Pending dengan spasi dan campuran huruf",
			input:     "  Pending  ",
			want:      "pending",
			expectErr: false,
		},
		{
			name:      "status rejected valid",
			input:     "rejected",
			want:      "rejected",
			expectErr: false,
		},
		{
			name:      "status cancelled valid",
			input:     "cancelled",
			want:      "cancelled",
			expectErr: false,
		},
		{
			name:      "status done tidak dikenal menghasilkan error",
			input:     "done",
			want:      "",
			wantErr:   ErrInvalidStatus,
			expectErr: true,
		},
		{
			name:      "status acak menghasilkan error",
			input:     "invalid-status",
			want:      "",
			wantErr:   ErrInvalidStatus,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStatusFilter(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("ParseStatusFilter(%q) diharapkan error, tetapi tidak ada error", tt.input)
				}
				if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Errorf("ParseStatusFilter(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("ParseStatusFilter(%q) tidak diharapkan error, tetapi dapat: %v", tt.input, err)
				}
				if got != tt.want {
					t.Errorf("ParseStatusFilter(%q) = %q, want %q", tt.input, got, tt.want)
				}
			}
		})
	}
}