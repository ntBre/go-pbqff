package main

import (
	"reflect"
	"testing"
)

func TestParseDeltas(t *testing.T) {
	tests := []struct {
		msg   string
		in    string
		out   []float64
		nc    int
		delta float64
	}{
		{
			msg:   "normal input",
			in:    "1:0.005,2:0.010,3:0.015,4:0.0075",
			nc:    9,
			delta: -1,
			out: []float64{
				0.005, 0.010, 0.015,
				0.0075, -1, -1,
				-1, -1, -1,
			},
		},
		{
			msg:   "spaces in input",
			in:    "1:0.005, 2: 0.010, 3:   0.015, 4:0.0075",
			nc:    6,
			delta: 0.005,
			out: []float64{
				0.005, 0.010, 0.015,
				0.0075, 0.005, 0.005,
			},
		},
		{
			msg:   "nonconsecutive coords",
			in:    "1:0.005,4: 0.010,7:0.015",
			nc:    9,
			delta: 0.005,
			out: []float64{
				0.005, 0.005, 0.005,
				0.010, 0.005, 0.005,
				0.015, 0.005, 0.005,
			},
		},
	}
	for _, test := range tests {
		c := new(Config)
		c.Ncoords = test.nc
		c.Delta = test.delta
		got := c.ParseDeltas(test.in)
		if !reflect.DeepEqual(got, test.out) {
			t.Errorf("ParseDeltas(%q): got %v, wanted %v\n",
				test.msg, got, test.out)
		}
	}
}

func TestProcessGeom(t *testing.T) {
	tests := []struct {
		in      string
		gtype   string
		ncoords int
	}{
		{
			in: `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`,
			gtype:   "zmat",
			ncoords: 6,
		},
		{
			in: `6
DF-CCSD(T)-F12/CC-PVTZ-F12  ENERGY=-78.46753079
C          0.0000000000        0.0000000000       -0.6668427197
C          0.0000000000        0.0000000000        0.6668427197
H          0.0000000000        0.9238557835       -1.2312205732
H          0.0000000000       -0.9238557835       -1.2312205732
H          0.0000000000        0.9238557835        1.2312205732
H          0.0000000000       -0.9238557835        1.2312205732`,
			gtype:   "xyz",
			ncoords: 18,
		},
		{
			in: `C        0.0000000000        0.0000000000       -1.6794733900
C        0.0000000000        1.2524327590        0.6959098120
C        0.0000000000       -1.2524327590        0.6959098120
H        0.0000000000        3.0146272390        1.7138963510
H        0.0000000000       -3.0146272390        1.7138963510`,
			gtype:   "xyz",
			ncoords: 15,
		},
	}
	for _, test := range tests {
		c := new(Config)
		c.Geometry = test.in
		c.GeomType = test.gtype
		c.ProcessGeom()
		got := c.Ncoords
		if got != test.ncoords {
			t.Errorf("got %v, wanted %v\n", got, test.ncoords)
		}
	}
}
