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

func compConf(t *testing.T, a, b Config) {
	if a.Cluster != b.Cluster {
		t.Errorf("got %v, wanted %v\n", a.Cluster, b.Cluster)
	}
	if a.Package != b.Package {
		t.Errorf("got %v, wanted %v\n", a.Package, b.Package)
	}
	if a.Program != b.Program {
		t.Errorf("got %v, wanted %v\n", a.Program, b.Program)
	}
	if a.WorkQueue != b.WorkQueue {
		t.Errorf("got %v, wanted %v\n", a.WorkQueue, b.WorkQueue)
	}
	if a.Delta != b.Delta {
		t.Errorf("got %v, wanted %v\n", a.Delta, b.Delta)
	}
	if !reflect.DeepEqual(a.Deltas, b.Deltas) {
		t.Errorf("got %v, wanted %v\n", a.Deltas, b.Deltas)
	}
	if a.Geometry != b.Geometry {
		t.Errorf("got %v, wanted %v\n", a.Geometry, b.Geometry)
	}
	if a.GeomType != b.GeomType {
		t.Errorf("got %v, wanted %v\n", a.GeomType, b.GeomType)
	}
	if a.Flags != b.Flags {
		t.Errorf("got %v, wanted %v\n", a.Flags, b.Flags)
	}
	if a.Deriv != b.Deriv {
		t.Errorf("got %v, wanted %v\n", a.Deriv, b.Deriv)
	}
	if a.JobLimit != b.JobLimit {
		t.Errorf("got %v, wanted %v\n", a.JobLimit, b.JobLimit)
	}
	if a.ChunkSize != b.ChunkSize {
		t.Errorf("got %v, wanted %v\n", a.ChunkSize, b.ChunkSize)
	}
	if a.CheckInt != b.CheckInt {
		t.Errorf("got %v, wanted %v\n", a.CheckInt, b.CheckInt)
	}
	if a.SleepInt != b.SleepInt {
		t.Errorf("got %v, wanted %v\n", a.SleepInt, b.SleepInt)
	}
	if a.NumCPUs != b.NumCPUs {
		t.Errorf("got %v, wanted %v\n", a.NumCPUs, b.NumCPUs)
	}
	if a.PBSMem != b.PBSMem {
		t.Errorf("got %v, wanted %v\n", a.PBSMem, b.PBSMem)
	}
	if a.Intder != b.Intder {
		t.Errorf("got %v, wanted %v\n", a.Intder, b.Intder)
	}
	if a.Spectro != b.Spectro {
		t.Errorf("got %v, wanted %v\n", a.Spectro, b.Spectro)
	}
	if a.Ncoords != b.Ncoords {
		t.Errorf("got %v, wanted %v\n", a.Ncoords, b.Ncoords)
	}
	if !reflect.DeepEqual(a.EnergyLine, b.EnergyLine) {
		t.Errorf("got %v, wanted %v\n", a.EnergyLine, b.EnergyLine)
	}
	if !reflect.DeepEqual(a.Queue, b.Queue) {
		t.Errorf("got %#+v, wanted %#+v\n", a.Queue, b.Queue)
	}
	if a.MolproTmpl != b.MolproTmpl {
		t.Errorf("got %v, wanted %v\n", a.MolproTmpl, b.MolproTmpl)
	}
	if a.AnpassTmpl != b.AnpassTmpl {
		t.Errorf("got %v, wanted %v\n", a.AnpassTmpl, b.AnpassTmpl)
	}
	if a.IntderTmpl != b.IntderTmpl {
		t.Errorf("got %v, wanted %v\n", a.IntderTmpl, b.IntderTmpl)
	}
}

func TestParseInfile(t *testing.T) {
	tmp := Conf
	defer func() {
		Conf = tmp
	}()
	tests := []struct {
		msg  string
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
				Delta:     0.005,
				GeomType:  "zmat",
				ChunkSize: 8,
				Deriv:     4,
				JobLimit:  1024,
				NumCPUs:   1,
				CheckInt:  100,
				WorkQueue: "",
				Queue: PBS{
					SinglePt: pbsMaple,
					ChunkPts: ptsMaple,
				},
				SleepInt:   60,
				Intder:     "/home/brent/Packages/intder/intder",
				Spectro:    "",
				PBSMem:     8,
				EnergyLine: regexp.MustCompile(`energy=`),
				Ncoords:    6,
				Package:    "molpro",
				MolproTmpl: "molpro.in",
				AnpassTmpl: "anpass.in",
				IntderTmpl: "intder.in",
				Deltas:     []float64{0.005, 0.005, 0.005, 0.005, 0.005, 0.005},
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
				Delta:     0.005,
				GeomType:  "zmat",
				ChunkSize: 8,
				Deriv:     4,
				JobLimit:  1024,
				NumCPUs:   1,
				CheckInt:  100,
				WorkQueue: "",
				Queue: PBS{
					SinglePt: pbsMaple,
					ChunkPts: ptsMaple,
				},
				SleepInt:   60,
				Intder:     "/home/brent/Packages/intder/intder",
				Spectro:    "",
				PBSMem:     8,
				EnergyLine: regexp.MustCompile(`^\s*CCCRE\s+=`),
				Ncoords:    6,
				Package:    "molpro",
				MolproTmpl: "molpro.in",
				AnpassTmpl: "anpass.in",
				IntderTmpl: "intder.in",
				Deltas:     []float64{0.005, 0.005, 0.005, 0.005, 0.005, 0.005},
			},
		},
		{
			msg: "eland gauss",
			in:  "testfiles/eland_gauss.in",
			want: Config{
				Cluster: "maple",
				Program: "sic",
				Geometry: `C        0.0000000000        0.0000000000       -1.6794733900
C        0.0000000000        1.2524327590        0.6959098120
C        0.0000000000       -1.2524327590        0.6959098120
H        0.0000000000        3.0146272390        1.7138963510
H        0.0000000000       -3.0146272390        1.7138963510`,
				Delta:     0.005,
				GeomType:  "xyz",
				ChunkSize: 8,
				Deriv:     4,
				JobLimit:  8000,
				NumCPUs:   1,
				CheckInt:  100,
				Flags:     "noopt",
				WorkQueue: "",
				Queue: &Slurm{
					SinglePt: pbsSlurm,
					ChunkPts: ptsSlurm,
				},
				SleepInt:   1,
				Intder:     "/home/r410/programs/intder/Intder2005.x",
				Spectro:    "/home/r410/programs/spec3jm.ifort-O0.static.x",
				PBSMem:     8,
				EnergyLine: regexp.MustCompile(`SCF Done:`),
				Ncoords:    15,
				Package:    "g16",
				MolproTmpl: "molpro.in",
				AnpassTmpl: "anpass.in",
				IntderTmpl: "intder.in",
				Deltas: []float64{
					0.005, 0.005, 0.005,
					0.005, 0.005, 0.005,
					0.005, 0.005, 0.005,
					0.005, 0.005, 0.005,
					0.005, 0.005, 0.005,
				},
			},
		},
	}
	for _, test := range tests[2:] {
		Conf = NewConfig()
		ParseInfile(test.in)
		compConf(t, Conf, test.want)
	}
}
