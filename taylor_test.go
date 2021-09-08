package main

import "testing"

func TestTaylor(t *testing.T) {
	intder, _ := LoadIntder("tests/sic/intder.in")
	Taylor([]string{"H", "O", "H"}, intder)
}
