package main

import (
	"reflect"
	"testing"
)

func TestNormalizeArgs(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"glued watch", []string{"-w1"}, []string{"-w", "1"}},
		{"glued section", []string{"-sjapan"}, []string{"-s", "japan"}},
		{"separated watch unchanged", []string{"-w", "30"}, []string{"-w", "30"}},
		{"equals form unchanged", []string{"-w=1"}, []string{"-w=1"}},
		{"long flag unchanged", []string{"--watch", "30"}, []string{"--watch", "30"}},
		{"bare short unchanged", []string{"-w"}, []string{"-w"}},
		{"bool flags unchanged", []string{"-j", "-v"}, []string{"-j", "-v"}},
		{"multiple glued", []string{"-w5", "-sus"}, []string{"-w", "5", "-s", "us"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orig := append([]string(nil), tc.in...)
			got := normalizeArgs(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("normalizeArgs(%v) = %v, want %v", tc.in, got, tc.want)
			}
			if !reflect.DeepEqual(tc.in, orig) {
				t.Errorf("normalizeArgs mutated input: got %v, was %v", tc.in, orig)
			}
		})
	}
}
