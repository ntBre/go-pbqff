package main

import (
	"bytes"
	"reflect"
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
