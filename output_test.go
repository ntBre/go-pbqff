package main

import (
	"reflect"
	"testing"
)

func TestParseOutput(t *testing.T) {
	tests := []struct {
		file   string
		coords []float64
		energy float64
	}{
		{
			file: "testfiles/read/output/ref.out",
			coords: []float64{
				0.0000000000, 0.7574590974, 0.5217905143,
				0.0000000000, 0.0000000000, -0.0657441568,
				0.0000000000, -0.7574590974, 0.5217905143,
			},
			energy: -76.369839607972,
		},
	}
	for _, test := range tests {
		coords, energy := ParseOutput(test.file, true)
		if !reflect.DeepEqual(coords, test.coords) {
			t.Errorf("got %v, wanted %v\n", coords, test.coords)
		}
		if energy != test.energy {
			t.Errorf("got %v, wanted %v\n", energy, test.energy)
		}
	}
}

func TestFormatOutput(t *testing.T) {
	FormatOutput("testfiles/read/output/")
	// t.Errorf("")
}
