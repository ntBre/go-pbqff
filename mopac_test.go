package main

import (
	"reflect"
	"testing"
)

func TestLoadMopac(t *testing.T) {
	var got Mopac
	got.Load("testfiles/mopac.in")
	want := Mopac{
		Dir:  "",
		Head: "",
		Opt:  "",
		Body: "",
		Geom: "",
		Tail: "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}
