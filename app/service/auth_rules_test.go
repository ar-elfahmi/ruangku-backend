package service

import "testing"

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"spasi dan huruf besar", " Andi@Kampus.AC.id ", "andi@kampus.ac.id"},
		{"sudah normal", "andi@kampus.ac.id", "andi@kampus.ac.id"},
		{"huruf besar tanpa spasi", "ANDI@KAMPUS.AC.ID", "andi@kampus.ac.id"},
		{"string kosong", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeEmail(tt.input); got != tt.want {
				t.Errorf("NormalizeEmail(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
