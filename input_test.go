package main

import (
	"reflect"
	"testing"
)

func TestParseInfile(t *testing.T) {
	input := ParseInfile("testfiles/test.in")
	after := [NumKeys]string{
		QueueType: "maple",
		Program:   "molpro",
		Geometry: `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`,
		GeomType:   "zmat",
		IntderCmd:  "/home/brent/Packages/intder/intder",
		ChunkSize:  "8",
		AnpassCmd:  "",
		SpectroCmd: "",
	}
	if !reflect.DeepEqual(input, after) {
		t.Errorf("\ngot %q\nwad %q\n", input, after)
	}
}

func TestParseDeltas(t *testing.T) {
	tests := []struct {
		msg string
		c   Configuration
		in  string
		we  error
		out []float64
	}{
		{
			msg: "normal input",
			c: Configuration{
				Delta:   0.005,
				Ncoords: 9,
			},
			in: "1:0.005,2:0.010,3:0.015,4:0.0075",
			we: nil,
			out: []float64{
				0.005, 0.010, 0.015,
				0.0075, 0.005, 0.005,
				0.005, 0.005, 0.005,
			},
		},
		{
			msg: "spaces in input",
			c: Configuration{
				Delta:   0.005,
				Ncoords: 9,
			},
			in: "1:0.005, 2: 0.010, 3:   0.015, 4:0.0075",
			we: nil,
			out: []float64{
				0.005, 0.010, 0.015,
				0.0075, 0.005, 0.005,
				0.005, 0.005, 0.005,
			},
		},
	}
	for i, test := range tests {
		err := tests[i].c.ParseDeltas(test.in)
		if !reflect.DeepEqual(tests[i].c.Deltas, test.out) {
			t.Errorf("ParseDeltas(%q): got %v, wanted %v\n",
				test.msg, tests[i].c.Deltas, test.out)
		}
		if test.we != err {
			t.Errorf("ParseDeltas(%q): got %v, wanted %v\n",
				test.msg, err, test.we)
		}
	}
}

func TestNewConfig(t *testing.T) {
	want := Configuration{
		Program:   "molpro",
		QueueType: "maple",
		GeomType:  "zmat",
		Delta:     0.005,
		Deltas: []float64{
			0.005, 0.005, 0.005,
			0.005, 0.005, 0.005,
			0.005, 0.005, 0.005,
		},
		Geometry: `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`,
		ChunkSize: 8,
		Deriv:     4,
		JobLimit:  1024,
		CheckInt:  100,
		SleepInt:  1,
		NumJobs:   8,
		Ncoords:   9,
		IntderCmd: "/home/brent/Packages/intder/intder",
	}
	inp := ParseInfile("testfiles/test.in")
	got := NewConfig(inp)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, wanted %+v\n", got, want)
	}
}
