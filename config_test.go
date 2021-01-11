package main

import (
	"reflect"
	"testing"
)

func TestParseDeltas(t *testing.T) {
	tests := []struct {
		msg string
		in  string
		we  error
		out []float64
	}{
		{
			msg: "normal input",
			in:  "1:0.005,2:0.010,3:0.015,4:0.0075",
			we:  nil,
			out: []float64{
				0.005, 0.010,
				0.015, 0.0075,
			},
		},
		{
			msg: "spaces in input",
			in:  "1:0.005, 2: 0.010, 3:   0.015, 4:0.0075",
			we:  nil,
			out: []float64{
				0.005, 0.010,
				0.015, 0.0075,
			},
		},
	}
	for _, test := range tests {
		got := ParseDeltas(test.in)
		if !reflect.DeepEqual(got, test.out) {
			t.Errorf("ParseDeltas(%q): got %v, wanted %v\n",
				test.msg, got, test.out)
		}
	}
}
