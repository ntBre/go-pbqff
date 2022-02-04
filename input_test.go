package main

import (
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

func compConf(a, b Config) (ok bool, got interface{}, want interface{}) {
	if a.Cluster != b.Cluster {
		return false, a.Cluster, b.Cluster
	}
	if a.Package != b.Package {
		return false, a.Package, b.Package
	}
	if a.Program != b.Program {
		return false, a.Program, b.Program
	}
	if a.WorkQueue != b.WorkQueue {
		return false, a.WorkQueue, b.WorkQueue
	}
	if a.Delta != b.Delta {
		return false, a.Delta, b.Delta
	}
	if !reflect.DeepEqual(a.Deltas, b.Deltas) {
		return false, a.Deltas, b.Deltas
	}
	if a.Geometry != b.Geometry {
		return false, a.Geometry, b.Geometry
	}
	if a.GeomType != b.GeomType {
		return false, a.GeomType, b.GeomType
	}
	if a.Flags != b.Flags {
		return false, a.Flags, b.Flags
	}
	if a.Deriv != b.Deriv {
		return false, a.Deriv, b.Deriv
	}
	if a.JobLimit != b.JobLimit {
		return false, a.JobLimit, b.JobLimit
	}
	if a.ChunkSize != b.ChunkSize {
		return false, a.ChunkSize, b.ChunkSize
	}
	if a.CheckInt != b.CheckInt {
		return false, a.CheckInt, b.CheckInt
	}
	if a.SleepInt != b.SleepInt {
		return false, a.SleepInt, b.SleepInt
	}
	if a.NumCPUs != b.NumCPUs {
		return false, a.NumCPUs, b.NumCPUs
	}
	if a.PBSMem != b.PBSMem {
		return false, a.PBSMem, b.PBSMem
	}
	if a.Intder != b.Intder {
		return false, a.Intder, b.Intder
	}
	if a.Spectro != b.Spectro {
		return false, a.Spectro, b.Spectro
	}
	if a.Ncoords != b.Ncoords {
		return false, a.Ncoords, b.Ncoords
	}
	if !reflect.DeepEqual(a.EnergyLine, b.EnergyLine) {
		return false, a.EnergyLine, b.EnergyLine
	}
	if !reflect.DeepEqual(a.PBSTmpl, b.PBSTmpl) {
		return false, a.PBSTmpl, b.PBSTmpl
	}
	if a.QueueSystem != b.QueueSystem {
		return false, a.QueueSystem, b.QueueSystem
	}
	if a.MolproTmpl != b.MolproTmpl {
		return false, a.MolproTmpl, b.MolproTmpl
	}
	if a.AnpassTmpl != b.AnpassTmpl {
		return false, a.AnpassTmpl, b.AnpassTmpl
	}
	if a.IntderTmpl != b.IntderTmpl {
		return false, a.IntderTmpl, b.IntderTmpl
	}
	return true, nil, nil
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
				Deltas:      []float64{0.005, 0.005, 0.005, 0.005, 0.005, 0.005},
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
				Deltas:      []float64{0.005, 0.005, 0.005, 0.005, 0.005, 0.005},
			},
		},
	}
	for _, test := range tests {
		ParseInfile(test.in)
		if ok, got, want := compConf(Conf, test.want); !ok {
			t.Errorf("got %v wanted %v\n", got, want)
		}
	}
}
