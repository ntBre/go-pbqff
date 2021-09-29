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
#P PM6=(print,zero,input) opt

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
	tmp := Conf.At(EnergyLine)
	defer func() {
		Conf.Set(EnergyLine, tmp)
	}()
	Conf.Set(EnergyLine, regexp.MustCompile(`SCF Done:`))
	g := new(Gaussian)
	energy, time, grad, err := g.ReadOut("testfiles/gaussian/opt.out")
	want := struct {
		energy float64
		time   float64
		grad   []float64
		err    error
	}{-0.195847985171e-01, 0.7, nil, nil}
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
