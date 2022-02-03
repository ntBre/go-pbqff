package main

import (
	"path/filepath"
	"testing"
)

func TestTaylor(t *testing.T) {
	tmp := Conf.IntderCmd
	path, _ := filepath.Abs("bin/intder")
	Conf.Set(IntderCmd, path)
	defer func() {
		Conf.Set(IntderCmd, tmp)
	}()
	intder, _ := LoadIntder("tests/sic/intder.in")
	Taylor([]string{"H", "O", "H"}, intder)
}
