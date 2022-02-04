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
	Conf.Program = "test"
	ProcessInput("program=notatest")
	if Conf.Program != "notatest" {
		t.Errorf("got %q, wanted %q\n",
			Conf.Program, "notatest")
	}
}

func compConf(a, b Config) bool {
	if a.Cluster != b.Cluster {
		return false
	}
	if a.Package != b.Package {
		return false
	}
	if a.Program != b.Program {
		return false
	}
	if a.WorkQueue != b.WorkQueue {
		return false
	}
	if a.Delta != b.Delta {
		return false
	}
	if !reflect.DeepEqual(a.Deltas, b.Deltas) {
		return false
	}
	if a.Geometry != b.Geometry {
		fmt.Printf("got\n%+#v, wanted\n%+#v\n", a.Geometry, b.Geometry)
		return false
	}
	if a.GeomType != b.GeomType {
		return false
	}
	if a.Flags != b.Flags {
		return false
	}
	if a.Deriv != b.Deriv {
		return false
	}
	if a.JobLimit != b.JobLimit {
		return false
	}
	if a.ChunkSize != b.ChunkSize {
		return false
	}
	if a.CheckInt != b.CheckInt {
		return false
	}
	if a.SleepInt != b.SleepInt {
		return false
	}
	if a.NumCPUs != b.NumCPUs {
		return false
	}
	if a.PBSMem != b.PBSMem {
		return false
	}
	if a.Intder != b.Intder {
		return false
	}
	if a.Spectro != b.Spectro {
		return false
	}
	if a.Ncoords != b.Ncoords {
		fmt.Println(a.Ncoords, b.Ncoords)
		return false
	}
	if !reflect.DeepEqual(a.EnergyLine, b.EnergyLine) {
		fmt.Print("it's energyline")
		return false
	}
	if !reflect.DeepEqual(a.PBSTmpl, b.PBSTmpl) {
		fmt.Print("it's pbstempl")
		return false
	}
	if a.QueueSystem != b.QueueSystem {
		fmt.Printf("%q %q\n", a.QueueSystem, b.QueueSystem)
		return false
	}
	if a.MolproTmpl != b.MolproTmpl {
		fmt.Printf("%q %q\n", a.MolproTmpl, b.MolproTmpl)
		return false
	}
	if a.AnpassTmpl != b.AnpassTmpl {
		fmt.Printf("%q %q\n", a.AnpassTmpl, b.AnpassTmpl)
		return false
	}
	if a.IntderTmpl != b.IntderTmpl {
		fmt.Printf("%q %q\n", a.IntderTmpl, b.IntderTmpl)
		return false
	}
	return true
}

func TestParseInfile(t *testing.T) {
	tmp := Conf
	Conf = NewConfig()
	defer func() {
		Conf = tmp
	}()
	tests := []struct {
		in   string
		want Config
	}{
		{
			in: "testfiles/test.in",
			want: Config{
				Cluster: "maple",
				Program: "molpro",
				Geometry: `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`,
				Delta:       0.005,
				GeomType:    "zmat",
				ChunkSize:   8,
				Deriv:       4,
				JobLimit:    1024,
				NumCPUs:     1,
				CheckInt:    100,
				WorkQueue:   "",
				QueueSystem: "pbs",
				SleepInt:    60,
				Intder:      "/home/brent/Packages/intder/intder",
				Spectro:     "",
				PBSTmpl:     pbsMaple,
				PBSMem:      8,
				EnergyLine:  regexp.MustCompile(`energy=`),
				Ncoords:     6,
				Package:     "molpro",
				MolproTmpl:  "molpro.in",
				AnpassTmpl:  "anpass.in",
				IntderTmpl:  "intder.in",
			},
		},
		{
			in: "testfiles/cccr.in",
			want: Config{
				Cluster: "maple",
				Program: "cart",
				Geometry: `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`,
				Delta:       0.005,
				GeomType:    "zmat",
				ChunkSize:   8,
				Deriv:       4,
				JobLimit:    1024,
				NumCPUs:     1,
				CheckInt:    100,
				WorkQueue:   "",
				QueueSystem: "pbs",
				SleepInt:    60,
				Intder:      "/home/brent/Packages/intder/intder",
				Spectro:     "",
				PBSTmpl:     pbsMaple,
				PBSMem:      8,
				EnergyLine:  regexp.MustCompile(`^\s*CCCRE\s+=`),
				Ncoords:     6,
				Package:     "molpro",
				MolproTmpl:  "molpro.in",
				AnpassTmpl:  "anpass.in",
				IntderTmpl:  "intder.in",
			},
		},
	}
	for _, test := range tests {
		ParseInfile(test.in)
		if !compConf(Conf, test.want) {
			t.Error()
		}
	}
}
