package main

import (
	"path/filepath"
	"testing"
)

func TestTaylor(t *testing.T) {
	tmp := Conf.Intder
	path, _ := filepath.Abs("bin/intder")
	Conf.Intder =  path
	defer func() {
		Conf.Intder =  tmp
	}()
	intder, _ := LoadIntder("tests/sic/intder.in")
	Taylor([]string{"H", "O", "H"}, intder)
}
