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

var names = []string{"Al", "O", "O", "Al"}

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
	m := Molpro{}
	temp := energyLine
	defer func() {
		energyLine = temp
	}()
	tests := []struct {
		msg      string
		filename string
		eline    *regexp.Regexp
		energy   float64
		time     float64
		grad     []float64
		err      error
	}{
		{
			msg:      "Gradient success",
			filename: "testfiles/read/showgrad.out",
			energy:   -152.379641595220,
			time:     28.92,
			grad: []float64{
				0.038675130622946, 0.002051946183374, 0.015073821827216,
				0.115196189670610, 0.146068323479018, 0.149725120162171,
				0.017871578588965, 0.009170027453292, 0.010289031057109,
				-0.019700971149367, -0.092155534025566, -0.096689855587206,
				-0.141218912305456, 0.057539789396310, -0.098009841881230,
				-0.010823015427707, -0.122674552486430, 0.019611724421961,
			},
			err: nil,
		},
		{
			msg:      "Normal success",
			filename: "testfiles/read/good.out",
			energy:   -168.463747095015,
			time:     10372.08,
			err:      nil,
		},
		{
			msg:      "Error in output",
			filename: "testfiles/read/error.out",
			energy:   math.NaN(),
			time:     119.29,
			err:      ErrFileContainsError,
		},
		{
			msg:      "File not found",
			filename: "nonexistent/file",
			energy:   math.NaN(),
			time:     0.0,
			err:      ErrFileNotFound,
		},
		{
			msg:      "One-line error",
			filename: "testfiles/read/shortcircuit.out",
			energy:   math.NaN(),
			time:     0.0,
			err:      ErrFileContainsError,
		},
		{
			msg:      "Blank file",
			filename: "testfiles/read/blank.out",
			energy:   math.NaN(),
			time:     0.0,
			err:      ErrBlankOutput,
		},
		{
			msg:      "Parse error",
			filename: "testfiles/read/parse.out",
			energy:   math.NaN(),
			time:     10372.08,
			err:      ErrFinishedButNoEnergy,
		},
		{
			msg:      "Sequoia partial",
			filename: "testfiles/read/seq.part",
			energy:   math.NaN(),
			time:     67.94,
			err:      ErrEnergyNotFound,
		},
		{
			msg:      "Sequoia success",
			filename: "testfiles/read/seq.out",
			eline:    regexp.MustCompile(`PBQFF\(2\)`),
			energy:   -634.43134170,
			time:     1075.84,
			err:      nil,
		},
		{
			msg:      "cccr success",
			filename: "testfiles/read/cccr.out",
			eline:    regexp.MustCompile(`^\s*CCCRE\s+=`),
			energy:   -56.591603910177,
			time:     567.99,
			err:      nil,
		},
	}

	for _, test := range tests {
		if test.eline != nil {
			energyLine = test.eline
		} else {
			energyLine = regexp.MustCompile(`energy=`)
		}
		energy, time, grad, err := m.ReadOut(test.filename)
		if math.IsNaN(test.energy) {
			if !math.IsNaN(energy) {
				t.Errorf("got not NaN, wanted NaN\n")
			}
		} else if energy != test.energy {
			t.Errorf("got %v, wanted %v\n", energy, test.energy)
		}
		if time != test.time {
			t.Errorf("got %v, wanted %v\n", time, test.time)
		}
		if !reflect.DeepEqual(grad, test.grad) {
			t.Errorf("got %#+v, wanted %#+v\n", grad, test.grad)
		}
		if err != test.err {
			t.Errorf("got %v, wanted %v\n", err, test.err)
		}
	}
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
		_, _, err := mp.HandleOutput("testfiles/read/error")
		if err != ErrFileContainsError {
			t.Errorf("got %q, wanted %q", err, ErrFileContainsError)
		}
	})
	// There was a problem on Sequoia where the new zmat params
	// were inexplicably not in the frequency calculation
	t.Run("Sequoia", func(t *testing.T) {
		p, _ := LoadMolpro("testfiles/load/molpro.in")
		p.Geometry = FormatZmat(Input[Geometry])
		_, zmat, _ := p.HandleOutput("testfiles/read/seq")
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
		cart, zmat := ReadLog("testfiles/read/seq.log")
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
	paraCount = make(map[string]int)
	cf := new([]CountFloat)
	prog.BuildPoints("testfiles/read/file07", names, cf, ch, true)
	var got []Calc
	for calc := range ch {
		got = append(got, calc)
	}
	want := []Calc{
		Calc{Name: "testfiles/read/inp/NHHH.00000", Targets: []Target{{1, cf, 0}}, cmdfile: "testfiles/read/inp/commands0.txt", Scale: 1},
		Calc{Name: "testfiles/read/inp/NHHH.00001", Targets: []Target{{1, cf, 1}}, cmdfile: "testfiles/read/inp/commands0.txt", Scale: 1},
		Calc{Name: "testfiles/read/inp/NHHH.00002", Targets: []Target{{1, cf, 2}}, cmdfile: "testfiles/read/inp/commands0.txt", Scale: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%v, wanted\n%v", got, want)
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

func TestIndex(t *testing.T) {
	tests := []struct {
		ncoords int
		ids     []int
		want    []int
	}{
		{
			ncoords: 9,
			ids:     []int{1, 1},
			want:    []int{0},
		},
		{9, []int{1, 2}, []int{1, 9}},
		{9, []int{2, 2}, []int{10}},
		{9, []int{1, 1, 1}, []int{0}},
	}
	for _, test := range tests {
		got := Index(test.ncoords, false, test.ids...)
		want := test.want
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	}
}

// water example:
// options for steps:
// 1 2 3 4 5 6 7 8 9 -1 -2 -3 -4 -5 -6 -7 -8 -9
// indices:
// 0 1 2 3 4 5 6 7 8  9 10 11 12 13 14 15 16 17
// grid is then 17x17 = 2ncoords-1 x 2ncoords-1
func TestE2dIndex(t *testing.T) {
	tests := []struct {
		ncoords int
		ids     []int
		want    []int
	}{
		{9, []int{1, 1}, []int{0}},
		{9, []int{1, 2}, []int{1, 18}},
		{9, []int{1, 8}, []int{7, 126}},
		{9, []int{2, 2}, []int{19}},
		{9, []int{1, -9}, []int{17, 306}},
		{9, []int{-9, -9}, []int{323}},
	}
	for _, test := range tests {
		got := E2dIndex(test.ncoords, test.ids...)
		want := test.want
		if !reflect.DeepEqual(got, want) {
			t.Errorf("E2dIndex(%d, %v): got %v, wanted %v\n",
				test.ncoords, test.ids, got, want)
		}
	}
}
