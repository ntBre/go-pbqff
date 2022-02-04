package main

import (
	"bytes"
	"reflect"
	"regexp"
	"testing"
)

func TestLoadGaussian(t *testing.T) {
	got, _ := LoadGaussian("testfiles/gaussian/opt.com")
	want := &Gaussian{
		Head: "%nprocs=4\n",
		Opt:  "#P PM6=(print,zero,input) \n",
		Body: `
the title

0 1
`,
		Tail: "@params.dat\n\n",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%#+v, wanted\n%#+v\n", got, want)
	}
}

func TestMakeInput(t *testing.T) {
	var got bytes.Buffer
	g, _ := LoadGaussian("testfiles/gaussian/opt.com")
	g.Geom = `
C  0.0000000000  0.0000000000 -0.0000000318
H  0.0000000000  0.0000000000  1.0840336982
H  1.0220367932  0.0000000000 -0.3613445025
H -0.5110183966 -0.8851098265 -0.3613445025
H -0.5110183966  0.8851098265 -0.3613445025
`
	g.makeInput(&got, opt)
	want := `%nprocs=4
#P PM6=(print,zero,input) opt=VeryTight

the title

0 1
C  0.0000000000  0.0000000000 -0.0000000318
H  0.0000000000  0.0000000000  1.0840336982
H  1.0220367932  0.0000000000 -0.3613445025
H -0.5110183966 -0.8851098265 -0.3613445025
H -0.5110183966  0.8851098265 -0.3613445025

@params.dat

`
	if !reflect.DeepEqual(got.String(), want) {
		t.Errorf("got\n%#+v, wanted\n%#+v\n", got.String(), want)
	}
}

func TestGaussFormatZmat(t *testing.T) {
	g := new(Gaussian)
	g.FormatZmat(
		`X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`,
	)
	got := g.Geom
	want := `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0

AlX = 0.85
OX = 1.1
XXO = 80.0
`
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestGaussFormatCart(t *testing.T) {
	g := new(Gaussian)
	g.FormatCart(`H 0.0000000000  0.7574590974  0.5217905143
O 0.0000000000  0.0000000000 -0.0657441568
H 0.0000000000 -0.7574590974  0.5217905143
`)
	got := g.Geom
	want := `H 0.0000000000  0.7574590974  0.5217905143
O 0.0000000000  0.0000000000 -0.0657441568
H 0.0000000000 -0.7574590974  0.5217905143
`
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestGaussReadOut(t *testing.T) {
	tmp := Conf.EnergyLine
	defer func() {
		Conf.EnergyLine = tmp
	}()
	Conf.EnergyLine = regexp.MustCompile(`SCF Done:`)
	g := new(Gaussian)
	energy, time, grad, err := g.ReadOut("testfiles/gaussian/opt.out")
	want := struct {
		err    error
		grad   []float64
		energy float64
		time   float64
	}{energy: 1.597082773539640e-01, time: 0.7, grad: nil, err: nil}
	if energy != want.energy {
		t.Errorf("got %v, wanted %v\n", energy, want.energy)
	}
	if time != want.time {
		t.Errorf("got %v, wanted %v\n", time, want.time)
	}
	if !reflect.DeepEqual(grad, want.grad) {
		t.Errorf("got %v, wanted %v\n", grad, want.grad)
	}
	if err != want.err {
		t.Errorf("got %v, wanted %v\n", err, want.err)
	}
}

func TestGaussHandleOutput(t *testing.T) {
	g := new(Gaussian)
	cart, zmat, err := g.HandleOutput("testfiles/gaussian/zmat_opt")
	want := struct {
		err  error
		cart string
		zmat string
	}{
		cart: `C  0.000000  0.000000  1.764568
C  0.000000  1.252433 -0.610816
C  0.000000 -1.252433 -0.610816
H  0.000000  3.014627 -1.628802
H  0.000000 -3.014627 -1.628802
`,
		zmat: `GCC=1.42101898
CCC=55.60133141
CH=1.07692776
HCC=147.8148823
`,
		err: nil,
	}
	if cart != want.cart {
		t.Errorf("got\n%#+v, wanted\n%#+v\n", cart, want.cart)
	}
	if zmat != want.zmat {
		t.Errorf("got\n%#+v, wanted\n%#+v\n", zmat, want.zmat)
	}
	if err != want.err {
		t.Errorf("got %v, wanted %v\n", err, want.err)
	}
}

func TestGaussReadFreqs(t *testing.T) {
	got := new(Gaussian).ReadFreqs("testfiles/gaussian/freq.out")
	want := []float64{
		3226.9641, 3212.4866, 3212.4866,
		3210.8590, 3210.8590, 1268.5148,
		1268.5148, 1141.1836, 1075.6897,
		1032.6991, 1032.6991, 904.5829,
		904.5828, 893.4540, 893.4540,
		743.1793, 743.1793, 665.5238,
		662.7349, 662.7349, 112.7008,
		112.7008, -564.2774, -564.2774,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}
