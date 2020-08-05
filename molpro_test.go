package main

import (
	"fmt"
	"math"
	"os"
	"path"
	"reflect"
	"regexp"
	"testing"
)

func TestLoadMolpro(t *testing.T) {
	got, _ := LoadMolpro("testfiles/load/molpro.in")
	want := &Molpro{
		Head: `memory,995,m   ! 30GB 12procs

gthresh,energy=1.d-12,zero=1.d-22,oneint=1.d-22,twoint=1.d-22;
gthresh,optgrad=1.d-8,optstep=1.d-8;
nocompress;

geometry={
`,
		Geometry: "",
		Tail: `basis={
default,cc-pvdz-f12
}
set,charge=0
set,spin=0
hf,accuracy=16,energy=1.0d-10
{ccsd(t)-f12,thrden=1.0d-8,thrvar=1.0d-10;orbital,IGNORE_ERROR;}
`,
		Opt: `{optg,grms=1.d-8,srms=1.d-8}
`,
		Extra: `pbqff=energy
`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%#v, wanted\n%#v\n", got, want)
	}
}

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
	load := "testfiles/load/molpro.in"
	write := "testfiles/write/opt.inp"
	right := "testfiles/right/opt.inp"
	mp, _ := LoadMolpro(load)
	mp.Geometry = FormatZmat(Input[Geometry])
	mp.WriteInput(write, opt)
	if !compareFile(write, right) {
		t.Errorf("mismatch between %s and %s\n", write, right)
	}
}

func TestReadOut(t *testing.T) {
	mp := Molpro{Geometry: FormatZmat(Input[Geometry])}
	temp := energyLine
	energyLine = regexp.MustCompile(`energy=`)
	defer func() {
		energyLine = temp
	}()

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

	t.Run("sequoia, partial", func(t *testing.T) {
		got, err := mp.ReadOut("testfiles/seq.part")
		if !math.IsNaN(got) {
			t.Errorf("got %v, wanted %v\n", got, math.NaN())
		} else if err != ErrEnergyNotFound {
			t.Errorf("got %q, wanted %q", err, ErrFinishedButNoEnergy)
		}
	})

	t.Run("sequoia success", func(t *testing.T) {
		e := energyLine
		energyLine = regexp.MustCompile(`PBQFF\(2\)`)
		defer func() {
			energyLine = e
		}()
		got, err := mp.ReadOut("testfiles/seq.out")
		want := -634.43134170
		if got != want {
			t.Errorf("got %v and %v, wanted %v\n", got, err, want)
		} else if err != nil {
			t.Error("got an error, didn't want one")
		}
	})

	t.Run("cccr success", func(t *testing.T) {
		e := energyLine
		energyLine = regexp.MustCompile(`^\s*CCCRE\s+=`)
		defer func() {
			energyLine = e
		}()
		got, err := mp.ReadOut("testfiles/cccr.out")
		want := -56.591603910177
		if got != want {
			t.Errorf("got %v and %v, wanted %v\n", got, err, want)
		} else if err != nil {
			t.Error("got an error, didn't want one")
		}
	})
}

func TestHandleOutput(t *testing.T) {
	mp := Molpro{Geometry: FormatZmat(Input[Geometry])}
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
	// There was a problem on Sequoia where the new zmat params
	// were inexplicably not in the frequency calculation
	t.Run("Sequoia", func(t *testing.T) {
		p, _ := LoadMolpro("testfiles/load/molpro.in")
		p.Geometry = FormatZmat(Input[Geometry])
		_, zmat, _ := p.HandleOutput("testfiles/seq")
		want := `ALX=                 1.20291856 ANG
OX=                  1.26606700 ANG
`
		p.Geometry = UpdateZmat(p.Geometry, zmat)
		p.WriteInput("testfiles/seq.freq", freq)
		if !reflect.DeepEqual(zmat, want) {
			t.Errorf("got %q, wanted %q\n", zmat, want)
		}
	})
}

func TestReadLog(t *testing.T) {
	t.Run("maple", func(t *testing.T) {
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
	})

	t.Run("sequoia", func(t *testing.T) {
		cart, zmat := ReadLog("testfiles/seq.log")
		wantCart := `AL 0.000000000 0.000000000 2.273186636
AL 0.000000000 0.000000000 -2.273186636
O 0.000000000 2.392519895 0.000000000
O 0.000000000 -2.392519895 0.000000000
`
		wantZmat := `ALX=                 1.20291856 ANG
OX=                  1.26606700 ANG
`
		if cart != wantCart {
			t.Errorf("\ngot %q, \nwad %q\n", cart, wantCart)
		}
		if zmat != wantZmat {
			t.Errorf("got %v, wanted %v\n", zmat, wantZmat)
		}
	})
}

func TestReadFreqs(t *testing.T) {
	mp := Molpro{Geometry: FormatZmat(Input[Geometry])}
	got := mp.ReadFreqs("testfiles/freq.out")
	want := []float64{805.31, 774.77, 679.79, 647.70, 524.26, 301.99}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestBuildPoints(t *testing.T) {
	prog, _ := LoadMolpro("testfiles/load/molpro.in")
	prog.Geometry = Input[Geometry]
	cart, _, _ := prog.HandleOutput("testfiles/opt")
	names := GetNames(cart)
	os.Mkdir("testfiles/read/inp", 0755)
	defer os.RemoveAll("testfiles/read/inp")
	fmt.Println("dir: ", path.Dir("testfiles/read/file07"))
	ch := make(chan Calc, 3)
	prog.BuildPoints("testfiles/read/file07", names, nil, ch, true)
	var got []Calc
	for calc := range ch {
		got = append(got, calc)
	}
	want := []Calc{
		Calc{Name: "testfiles/read/inp/NHHH.00000", Targets: []Target{{1, nil, 0}}},
		Calc{Name: "testfiles/read/inp/NHHH.00001", Targets: []Target{{1, nil, 1}}},
		Calc{Name: "testfiles/read/inp/NHHH.00002", Targets: []Target{{1, nil, 2}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestZipXYZ(t *testing.T) {
	fcoords := []float64{
		0.000000000, 2.391678166, 0.000000000,
		-2.274263181, 0.000000000, 0.000000000,
		2.274263181, 0.000000000, 0.000000000,
		0.000000000, -2.391678166, 0.000000000,
	}
	got := ZipXYZ(names, fcoords)
	want := `Al 0.0000000000 2.3916781660 0.0000000000
O -2.2742631810 0.0000000000 0.0000000000
O 2.2742631810 0.0000000000 0.0000000000
Al 0.0000000000 -2.3916781660 0.0000000000
`
	if got != want {
		t.Errorf("got\n%q, wanted\n%q\n", got, want)
	}
}
