package main

import "testing"

func TestTaylor(t *testing.T) {
	tmp := Conf.At(IntderCmd)
	Conf.Set(IntderCmd, "/home/brent/gopath/pbqff/bin/intder")
	defer func() {
		Conf.Set(IntderCmd, tmp)
	}()
	intder, _ := LoadIntder("tests/sic/intder.in")
	Taylor([]string{"H", "O", "H"}, intder)
}
