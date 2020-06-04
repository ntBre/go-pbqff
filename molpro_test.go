package main

import (
	"math"
	"reflect"
	"testing"
)

func TestFormatZmat(t *testing.T) {
	got := FormatZmat(Input[Geometry])
	want := `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
}
AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestWriteInputMolpro(t *testing.T) {
	mp := Molpro{FormatZmat(Input[Geometry]), Input[Basis],
		Input[Charge], Input[Spin], Input[Method], defaultOpt}
	mp.WriteInput("testfiles/opt/opt.inp", "templates/molpro.in")
}

func TestReadOut(t *testing.T) {
	mp := Molpro{FormatZmat(Input[Geometry]), Input[Basis],
		Input[Charge], Input[Spin], Input[Method], defaultOpt}

	t.Run("Successful reading", func(t *testing.T) {
		got, err := mp.ReadOut("testfiles/good.out")
		want := -168.463747095015
		if got != want {
			t.Errorf("got %v, wanted %v\n", got, want)
		} else if err != nil {
			t.Error("got an error, didn't want one")
		}
	})

	t.Run("Error in output", func(t *testing.T) {
		got, err := mp.ReadOut("testfiles/error.out")
		if !math.IsNaN(got) {
			t.Errorf("got %v, wanted %v\n", got, math.NaN())
		} else if err != ErrFileContainsError {
			t.Error("didn't get an error, wanted one")
		}
	})

	t.Run("File not found", func(t *testing.T) {
		got, err := mp.ReadOut("nonexistent/file")
		if !math.IsNaN(got) {
			t.Errorf("got %v, wanted %v\n", got, math.NaN())
		} else if err != ErrFileNotFound {
			t.Error("didn't get an error, wanted one")
		}
	})

	t.Run("One-line error", func(t *testing.T) {
		got, err := mp.ReadOut("testfiles/shortcircuit.out")
		if !math.IsNaN(got) {
			t.Errorf("got %v, wanted %v\n", got, math.NaN())
		} else if err != ErrFileContainsError {
			t.Errorf("got %q, wanted %q", err, ErrFileContainsError)
		}
	})

	t.Run("blank", func(t *testing.T) {
		got, err := mp.ReadOut("testfiles/blank.out")
		if !math.IsNaN(got) {
			t.Errorf("got %v, wanted %v\n", got, math.NaN())
		} else if err != ErrBlankOutput {
			t.Errorf("got %q, wanted %q", err, ErrBlankOutput)
		}
	})

	t.Run("parse error", func(t *testing.T) {
		got, err := mp.ReadOut("testfiles/parse.out")
		if !math.IsNaN(got) {
			t.Errorf("got %v, wanted %v\n", got, math.NaN())
		} else if err != ErrFinishedButNoEnergy {
			t.Errorf("got %q, wanted %q", err, ErrFinishedButNoEnergy)
		}
	})
}

func TestHandleOutput(t *testing.T) {
	mp := Molpro{FormatZmat(Input[Geometry]), Input[Basis],
		Input[Charge], Input[Spin], Input[Method], defaultOpt}
	t.Run("warning in outfile", func(t *testing.T) {
		_, _, err := mp.HandleOutput("testfiles/opt")
		if err != nil {
			t.Error("got an error, didn't want one")
		}
	})
	t.Run("no warning, normal case", func(t *testing.T) {
		_, _, err := mp.HandleOutput("testfiles/nowarn")
		if err != nil {
			t.Error("got an error, didn't want one")
		}
	})
	t.Run("Error in output", func(t *testing.T) {
		_, _, err := mp.HandleOutput("testfiles/error")
		if err != ErrFileContainsError {
			t.Errorf("got %q, wanted %q", err, ErrFileContainsError)
		}
	})
}

func TestReadLog(t *testing.T) {
	cart, zmat := ReadLog("testfiles/coords.log")
	wantCart := `O 1.000000000 0.118481857 -2.183553663
H 0.000000000 -1.563325812 -2.884671935
C 0.000000000 -0.014536611 0.273763522
N 0.000000000 -0.010373662 2.467030139
`
	wantZmat := `OH=                  0.96421314 ANG
OC=                  1.30226003 ANG
HOC=               109.53197453 DEG
CN=                  1.16062880 ANG
OCN=               176.79276221 DEG
`
	if cart != wantCart {
		t.Errorf("got %v, wanted %v\n", cart, wantCart)
	}
	if zmat != wantZmat {
		t.Errorf("got %v, wanted %v\n", zmat, wantZmat)
	}
}
