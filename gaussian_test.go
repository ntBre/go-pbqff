package main

import (
	"reflect"
	"testing"
)

func TestLoadGaussian(t *testing.T) {
	got, _ := LoadGaussian("testfiles/gaussian/opt.com")
	want := &Gaussian{
		Head: "%nprocs=4\n",
		Opt:  "#P PM6=(print,zero,input) opt\n",
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
