package main

import (
	"reflect"
	"testing"
)

func TestNewIntder(t *testing.T) {
	cart, _ := ReadLog("testfiles/coords.log")
	got := LoadIntder("testfiles/intder.in")
	got.ConvertCart(cart)
	want := &Intder{Geometry: `      1.000000000        0.118481857       -2.183553663
      0.000000000       -1.563325812       -2.884671935
      0.000000000       -0.014536611        0.273763522
      0.000000000       -0.010373662        2.467030139`}
	if got.Geometry != want.Geometry {
		t.Errorf("got %v, wanted %v\n", got.Geometry, want.Geometry)
	}
}

func TestWritePtsIntder(t *testing.T) {
	cart, _ := ReadLog("testfiles/coords.log")
	i := LoadIntder("testfiles/intder.in")
	i.ConvertCart(cart)
	i.WritePts("testfiles/pts/intder.in")
}

func TestRunIntder(t *testing.T) {
	RunIntder("testfiles/pts/intder")
}

func TestBuildPoints(t *testing.T) {
	prog := Molpro{
		Geometry: Input[Geometry],
		Basis:    Input[Basis],
		Charge:   Input[Charge],
		Spin:     Input[Spin],
		Method:   Input[Method],
	}
	cart, _, _ := prog.HandleOutput("testfiles/opt")
	names := GetNames(cart)
	got := BuildPoints("testfiles/pts/file07", names)
	want := []string{
		"testfiles/pts/inp/NHHH.00000",
		"testfiles/pts/inp/NHHH.00001",
		"testfiles/pts/inp/NHHH.00002",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestLoadIntder(t *testing.T) {
	LoadIntder("testfiles/intder.in")
}

func TestWriteIntderGeom(t *testing.T) {
	cart, _ := ReadLog("testfiles/coords.log")
	i := LoadIntder("testfiles/intder.in")
	i.ConvertCart(cart)
	longLine, _ := GetLongLine("testfiles/anpass1.out")
	i.WriteGeom("testfiles/freqs/intder_geom.in", longLine)
}

func TestReadGeom(t *testing.T) {
	cart, _ := ReadLog("testfiles/coords.log")
	i := LoadIntder("testfiles/intder.in")
	i.ConvertCart(cart)
	i.ReadGeom("testfiles/intder_geom.out")
	want := `        0.0000000000       -0.0115666469        2.4598228639
        0.0000000000       -0.0139207809        0.2726915161
        0.0000000000        0.1184234620       -2.1785371074
        0.0000000000       -1.5591967852       -2.8818447886`
	if i.Geometry != want {
		t.Errorf("got %v, wanted %v", i.Geometry, want)
	}
}

func TestReadIntderOut(t *testing.T) {
	cart, _ := ReadLog("testfiles/coords.log")
	i := LoadIntder("testfiles/intder.in")
	i.ConvertCart(cart)
	i.ReadGeom("testfiles/intder_geom.out")
	got := i.ReadOut("testfiles/fintder.out")
	want := []float64{437.8, 496.8, 1086.4,
		1267.6, 2337.7, 3811.4}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestRead9903(t *testing.T) {
	cart, _ := ReadLog("testfiles/coords.log")
	i := LoadIntder("testfiles/intder.in")
	i.ConvertCart(cart)
	i.ReadGeom("testfiles/intder_geom.out")
	i.Read9903("testfiles/fort.9903")
}

func TestWriteIntderFreqs(t *testing.T) {
	cart, _ := ReadLog("testfiles/coords.log")
	i := LoadIntder("testfiles/intder.in")
	i.ConvertCart(cart)
	i.ReadGeom("testfiles/intder_geom.out")
	i.Read9903("testfiles/fort.9903")
	i.WriteFreqs("testfiles/freqs/intder.in", GetNames(cart))
}

func TestLoadSpectro(t *testing.T) {
	LoadSpectro("testfiles/spectro.in")
}

func TestWriteSpectroInput(t *testing.T) {
	spec := LoadSpectro("testfiles/spectro.in")
	spec.WriteInput("testfiles/freqs/spectro.in")
}

func TestReadSpectroOutput(t *testing.T) {
	spec := LoadSpectro("testfiles/spectro.in")
	spec.ReadOutput("testfiles/spectro.out")
}
