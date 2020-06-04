package main

import (
	"bytes"
	"os/exec"
	"reflect"
	"testing"
	"time"
)

func TestMakeDirs(t *testing.T) {
	overwrite = true
	got := MakeDirs("testfiles")
	overwrite = false
	if got != nil {
		t.Errorf("got an error %q, didn't want one", got)
	}
}

func TestReadFile(t *testing.T) {
	got := ReadFile("testfiles/read.this")
	want := []string{
		"this is a sample file",
		"to test ReadFile",
		"does it skip leading space",
		"what about trailing",
		"and a blank line",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestLoadTemplate(t *testing.T) {
	temp := LoadTemplate("templates/molpro.in")
	mp := Molpro{"this is a geom", "cc-pvtz-f12",
		"charge", "spin", "ccsd(t)-f12", defaultOpt}
	var buf bytes.Buffer
	temp.Execute(&buf, mp)
	got := buf.String()
	want := `memory,312,m

gthresh,energy=1.d-12,zero=1.d-22,oneint=1.d-22,twoint=1.d-22;
gthresh,optgrad=1.d-8,optstep=1.d-8;
nocompress;

geometry={
this is a geom
basis=cc-pvtz-f12
set,charge=charge
set,spin=spin
hf,accuracy=16,energy=1.0d-10
ccsd(t)-f12,thrden=1.0d-8,thrvar=1.0d-10
optg,grms=1.d-8,srms=1.d-8`
	if !reflect.DeepEqual(got, want) {
		t.Errorf("\ngot %q\nwad %q\n", got, want)
	}
}

func TestMakeName(t *testing.T) {
	got := MakeName(Input[Geometry])
	want := "al2o2"
	want2 := "o2al2"
	if !(got == want || got == want2) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestHandleSignal(t *testing.T) {
	t.Run("received signal", func(t *testing.T) {
		c := make(chan error)
		go func(c chan error) {
			err := HandleSignal(35, 5*time.Second)
			c <- err
		}(c)
		exec.Command("pkill", "-35", "pbqff").Run()
		err := <-c
		if err != nil {
			t.Errorf("did not receive signal")
		}
	})
	t.Run("no signal", func(t *testing.T) {
		c := make(chan error)
		go func(c chan error) {
			err := HandleSignal(35, 50*time.Millisecond)
			c <- err
		}(c)
		exec.Command("pkill", "-34", "go-cart").Run()
		err := <-c
		if err == nil {
			t.Errorf("received signal and didn't want one")
		}
	})
}

func TestGetNames(t *testing.T) {
	prog := Molpro{
		Geometry: Input[Geometry],
		Basis:    Input[Basis],
		Charge:   Input[Charge],
		Spin:     Input[Spin],
		Method:   Input[Method],
	}
	cart, _, _ := prog.HandleOutput("testfiles/opt")
	got := GetNames(cart)
	want := []string{"N", "H", "H", "H"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestLoadAnpass(t *testing.T) {
	got := LoadAnpass("testfiles/anpass.small")
	want := Anpass{Fmt1: "%12.8f", Fmt2: "%20.12f"}
	if got.Fmt1 != want.Fmt1 {
		t.Errorf("got %#v, wanted %#v\n", got.Fmt1, want.Fmt1)
	}
	if got.Fmt2 != want.Fmt2 {
		t.Errorf("got %#v, wanted %#v\n", got.Fmt2, want.Fmt2)
	}
}

func TestWriteAnpass(t *testing.T) {
	a := LoadAnpass("testfiles/anpass.small")
	a.WriteAnpass("testfiles/anpass.test", []float64{0, 0, 0, 0, 0, 0})
	// if got != want {
	// 	t.Errorf("got %v, wanted %v\n", got, want)
	// }
}
