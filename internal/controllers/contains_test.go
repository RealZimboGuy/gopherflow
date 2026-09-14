package controllers

import "testing"

func TestContains(t *testing.T) {
	tests := []struct {
		name string
		arr  []string
		val  string
		want bool
	}{
		{"present", []string{"FINISHED", "FAILED"}, "FAILED", true},
		{"first entry", []string{"FINISHED", "FAILED"}, "FINISHED", true},
		{"absent", []string{"FINISHED", "FAILED"}, "IN_PROGRESS", false},
		{"empty slice", nil, "FINISHED", false},
		{"case sensitive", []string{"FINISHED"}, "finished", false},
		{"empty value", []string{"FINISHED"}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.arr, tt.val); got != tt.want {
				t.Errorf("contains(%v, %q) = %v, want %v", tt.arr, tt.val, got, tt.want)
			}
		})
	}
}
