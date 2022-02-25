package main

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"
)

func TestLoadMopac(t *testing.T) {
	var got Mopac
	got.Load("testfiles/mopac.in")
	want := Mopac{
		Dir: "",
		Head: `XYZ A0 scfcrt=1.D-21 aux(precision=9) external=params.dat 1SCF charge=1 PM6
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
			want: `XYZ A0 scfcrt=1.D-21 aux(precision=9) external=params.dat 1SCF charge=1 PM6
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
			energy: -0.255423728052956e+02,
			time:   0.02,
			grad:   nil,
		},
	}
	for _, test := range tests {
		m := new(Mopac)
		energy, time, grad, _ := m.ReadOut(test.infile)
		if test.energy != energy {
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
