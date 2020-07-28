package main

import (
	"reflect"
	"sort"
	"testing"
)

func TestNewIntder(t *testing.T) {
	cart, _ := ReadLog("testfiles/al2o2.log")
	got, _ := LoadIntder("testfiles/intder.full")
	got.ConvertCart(cart)
	want := &Intder{Geometry: `      0.000000000        2.473478532        0.000000000
     -2.348339221        0.000000000        0.000000000
      2.348339221        0.000000000        0.000000000
      0.000000000       -2.473478532        0.000000000`}
	if got.Geometry != want.Geometry {
		t.Errorf("got\n%v\n, wanted\n%v\n", got.Geometry, want.Geometry)
	}
}

func TestWritePtsIntder(t *testing.T) {
	cart, _ := ReadLog("testfiles/al2o2.log")
	i, _ := LoadIntder("testfiles/intder.full")
	i.ConvertCart(cart)
	i.WritePts("testfiles/pts/intder.in")
}

func TestRunIntder(t *testing.T) {
	RunIntder("testfiles/pts/intder")
}

func TestBuildPoints(t *testing.T) {
	prog, _ := LoadMolpro("testfiles/molpro.in")
	prog.Geometry = Input[Geometry]
	cart, _, _ := prog.HandleOutput("testfiles/opt")
	names := GetNames(cart)
	got := prog.BuildPoints("testfiles/file07", names, true)
	want := []Calc{
		Calc{"testfiles/inp/NHHH.00000", 0},
		Calc{"testfiles/inp/NHHH.00001", 1},
		Calc{"testfiles/inp/NHHH.00002", 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestLoadIntder(t *testing.T) {
	LoadIntder("testfiles/intder.full")
}

func TestWriteIntderGeom(t *testing.T) {
	cart, _ := ReadLog("testfiles/al2o2.log")
	i, _ := LoadIntder("testfiles/intder.full")
	i.ConvertCart(cart)
	longLine, _ := GetLongLine("testfiles/anpass1.out")
	i.WriteGeom("testfiles/freqs/intder_geom.in", longLine)
}

func TestReadGeom(t *testing.T) {
	t.Run("no dummy atoms", func(t *testing.T) {
		cart, _ := ReadLog("testfiles/al2o2.log")
		i, _ := LoadIntder("testfiles/intder.full")
		i.ConvertCart(cart)
		i.ReadGeom("testfiles/intder_geom.out")
		want := `        0.0000000000       -0.0115666469        2.4598228639
        0.0000000000       -0.0139207809        0.2726915161
        0.0000000000        0.1184234620       -2.1785371074
        0.0000000000       -1.5591967852       -2.8818447886`
		if i.Geometry != want {
			t.Errorf("got %v, wanted %v", i.Geometry, want)
		}
	})
	t.Run("dummy atoms", func(t *testing.T) {
		cart, _ := ReadLog("testfiles/dummy.log")
		i, _ := LoadIntder("testfiles/dummy.intder.in")
		i.ConvertCart(cart)
		i.ReadGeom("testfiles/dummy_geom.out")
		want := `        0.0000000000        0.0000000000        1.0109039650
        0.0000000000        0.0000000000       -1.0824085329
        0.0000000000        0.0000000000       -3.1489094311
        0.0000000000        1.1111111110       -1.0824085329
        1.1111111110        0.0000000000       -1.0824085329`
		if i.Geometry != want {
			t.Errorf("got\n%v, wanted\n%v", i.Geometry, want)
		}
	})
}

func TestReadIntderOut(t *testing.T) {
	cart, _ := ReadLog("testfiles/al2o2.log")
	i, _:= LoadIntder("testfiles/intder.full")
	i.ConvertCart(cart)
	i.ReadGeom("testfiles/intder_geom.out")
	got := i.ReadOut("testfiles/fintder.out")
	f := []float64{437.8, 496.8, 1086.4, 1267.6, 2337.7, 3811.4}
	sort.Sort(sort.Reverse(sort.Float64Slice(f)))
	want := f
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestRead9903(t *testing.T) {
	cart, _ := ReadLog("testfiles/al2o2.log")
	i, _:= LoadIntder("testfiles/intder.full")
	i.ConvertCart(cart)
	i.ReadGeom("testfiles/intder_geom.out")
	i.Read9903("testfiles/fort.9903")
}

func TestWriteIntderFreqs(t *testing.T) {
	cart, _ := ReadLog("testfiles/al2o2.log")
	i , _:= LoadIntder("testfiles/intder.full")
	order := i.ConvertCart(cart)
	i.ReadGeom("testfiles/intder_geom.out")
	i.Read9903("testfiles/prob.9903")
	i.WriteFreqs("testfiles/freqs/intder.in", order)
}
