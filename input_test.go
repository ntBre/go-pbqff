package main

import (
	"reflect"
	"testing"
)

func TestParseInfile(t *testing.T) {
	var before [NumKeys]string
	if !reflect.DeepEqual(Input, before) {
		t.Errorf("nonzero initial input")
	}
	ParseInfile("testfiles/test.in")
	after := [NumKeys]string{
		"PBS",
		"MOLPRO",
		`X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`,
		"ZMAT",
		"CCSD(T)-F12",
		"CC-PVTZ-F12",
		"0",
		"0",
	}
	if !reflect.DeepEqual(Input, after) {
		t.Errorf("\ngot %q\nwad %q\n", Input, after)
	}
}
