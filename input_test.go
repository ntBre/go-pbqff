package main

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"
)

func TestProcessInput(t *testing.T) {
	tmp := Conf
	defer func() {
		Conf = tmp
	}()
	Conf[ChemProg].Value = "test"
	ProcessInput("program=notatest")
	if Conf[ChemProg].Value != "notatest" {
		t.Errorf("got %q, wanted %q\n",
			Conf[ChemProg].Value, "notatest")
	}
}

func compConf(kws []interface{}, conf Config) (bool, string) {
	for k := range kws {
		if !reflect.DeepEqual(kws[k], conf.At(Key(k))) {
			return false,
				fmt.Sprintf("At %s, %v != %v\n",
					Key(k), kws[k], conf.At(Key(k)))
		}
	}
	return true, ""
}

func TestParseInfile(t *testing.T) {
	tmp := Conf
	Conf = NewConfig()
	defer func() {
		Conf = tmp
	}()
	tests := []struct {
		in   string
		want []interface{}
	}{
		{
			in: "testfiles/test.in",
			want: []interface{}{
				Cluster:  "maple",
				ChemProg: "molpro",
				Geometry: `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`,
				Delta:      0.005,
				GeomType:   "zmat",
				ChunkSize:  8,
				Deriv:      4,
				JobLimit:   1024,
				NumCPUs:    1,
				CheckInt:   100,
				Queue:      "",
				SleepInt:   60,
				IntderCmd:  "/home/brent/Packages/intder/intder",
				AnpassCmd:  "",
				SpectroCmd: "",
				PBS:        pbsMaple,
				PBSMem:     8,
				EnergyLine: regexp.MustCompile(`energy=`),
				Ncoords:    6,
			},
		},
		{
			in: "testfiles/cccr.in",
			want: []interface{}{
				Cluster:  "maple",
				ChemProg: "cart",
				Geometry: `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`,
				Delta:      0.005,
				GeomType:   "zmat",
				ChunkSize:  8,
				Deriv:      4,
				JobLimit:   1024,
				NumCPUs:    1,
				CheckInt:   100,
				Queue:      "",
				SleepInt:   60,
				IntderCmd:  "/home/brent/Packages/intder/intder",
				AnpassCmd:  "",
				SpectroCmd: "",
				PBS:        pbsMaple,
				PBSMem:     8,
				EnergyLine: regexp.MustCompile(`^\s*CCCRE\s+=`),
				Ncoords:    6,
			},
		},
	}
	for _, test := range tests {
		ParseInfile(test.in)
		if ok, msg := compConf(test.want, Conf); !ok {
			t.Errorf(msg)
		}
	}
}
