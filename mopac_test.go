package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"testing"
)

func TestLoadMopac(t *testing.T) {
	var got Mopac
	got.Load("testfiles/mopac.in")
	want := Mopac{
		Dir: "",
		Head: `scfcrt=1.D-21 aux(precision=9) external=params.dat charge=1 PM6
Comment line 1
Comment line 2
`,
		Geom: "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, wanted %#v\n", got, want)
	}
}
func TestWriteMopacInput(t *testing.T) {
	tests := []struct {
		load string
		geom string
		want string
		proc Procedure
	}{
		{
			load: "testfiles/mopac.in",
			geom: `H        0.0000000000        1.4186597974        0.9822041584
O        0.0000000000        0.0094006500       -0.1238566934
H        0.0000000000       -1.4280604475        0.9894964540`,
			want: `1SCF XYZ scfcrt=1.D-21 aux(precision=9) external=params.dat charge=1 PM6
Comment line 1
Comment line 2
H        0.0000000000        1.4186597974        0.9822041584
O        0.0000000000        0.0094006500       -0.1238566934
H        0.0000000000       -1.4280604475        0.9894964540

`,
			proc: none,
		},
	}
	for _, test := range tests {
		m := new(Mopac)
		m.Load(test.load)
		m.FormatCart(test.geom)
		f, _ := os.CreateTemp("", "")
		m.WriteInput(f.Name(), test.proc)
		byts, err := io.ReadAll(f)
		if err != nil {
			panic(err)
		}
		got := string(byts)
		if got != test.want {
			t.Errorf("got\n%#v, wanted\n%#v\n", got, test.want)
		}
	}
}

func TestMopacFormatZmat(t *testing.T) {
	tests := []struct {
		geom string
		want string
	}{
		{
			geom: `C
C 1 cc
C 1 cc 2 ccc
H 2 ch 1 hcc 3 180.0
H 3 ch 1 hcc 2 180.0
CC=                  1.42101898 ANG
CCC=                55.60133141 DEG
CH=                  1.07692776 ANG
HCC=               147.81488230 DEG
`,
			want: `C
C 1 cc
C 1 cc 2 ccc
H 2 ch 1 hcc 3 180.0
H 3 ch 1 hcc 2 180.0

CC=                  1.42101898
CCC=                55.60133141
CH=                  1.07692776
HCC=               147.81488230
`,
		},
	}
	for _, test := range tests {
		m := new(Mopac)
		m.FormatZmat(test.geom)
		got := m.Geom
		if got != test.want {
			fmt.Println(got)
			fmt.Println(test.want)
			t.Errorf("got\n%#v, wanted\n%#v\n", got, test.want)
		}
	}
}

func TestMopacReadOut(t *testing.T) {
	tests := []struct {
		infile string
		grad   []float64
		energy float64
		time   float64
	}{
		{
			infile: "testfiles/mopac.out",
			energy: -0.255423728052956e+02 / KCALHT,
			time:   0.02,
			grad:   nil,
		},
	}
	for _, test := range tests {
		m := new(Mopac)
		energy, time, grad, err := m.ReadOut(test.infile)
		if err != nil {
			t.Errorf("got an error %v, didn't want one", err)
		}
		if math.Abs(energy-test.energy) > 1e-17 {
			t.Errorf("got %v, wanted %v\n", energy, test.energy)
		}
		if test.time != time {
			t.Errorf("got %v, wanted %v\n", time, test.time)
		}
		if !reflect.DeepEqual(test.grad, grad) {
			t.Errorf("got %v, wanted %v\n", grad, test.grad)
		}
	}
}

func TestMopacHandleOutput(t *testing.T) {
	sic := SIC
	defer func() { SIC = sic }()
	SIC = false
	tests := []struct {
		infile string
		want   string
	}{
		{
			infile: "testfiles/mopac.opt",
			want: `C 0.0000000000 0.0000000000 0.0000000000
C 1.4361996439 0.0000000000 0.0000000000
C 0.7993316223 1.1932050849 0.0000000000
H 2.3607104536 -0.5060383603 0.0000000000
H 0.8934572415 2.2429362063 0.0000000000
`,
		},
	}
	for _, test := range tests {
		m := new(Mopac)
		got, _, err := m.HandleOutput(test.infile)
		if err != nil {
			t.Error("got an error, didn't want one")
		}
		if got != test.want {
			t.Errorf("got\n%#v, wanted\n%#v\n", got, test.want)
		}
	}
}

func TestMopacReadFreqs(t *testing.T) {
	m := new(Mopac)
	got := m.ReadFreqs("testfiles/mopac.freq.out")
	want := []float64{
		3225.14, 3206.55, 3156.25,
		3153.76, 1641.27, 1438.93,
		1393.34, 1228.47, 1024.32,
		945.26, 908.58, 885.60,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}
