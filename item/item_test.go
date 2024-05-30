package item

import "testing"

func Test_shorten(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"empty", args{""}, ""},
		{"simple", args{"kg"}, "kg"},
		{"simples", args{" kg "}, "kg"},
		{"normal", args{"Packung"}, "Pckng"},
		{"normal2", args{"Paket"}, "Pkt"},
		{"space", args{" große Dose "}, "grßDs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shorten(tt.args.s); got != tt.want {
				t.Errorf("shorten() = %v, want %v", got, tt.want)
			}
		})
	}
}
