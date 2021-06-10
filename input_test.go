package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseInfile(t *testing.T) {
	abs, _ := filepath.Abs("bin/intder")
	ParseInfile("testfiles/test.in")
	after := [NumKeys]string{
		QueueType: "pbs",
		Program:   "molpro",
		Geometry: `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`,
		GeomType:   "zmat",
		IntderCmd:  abs,
		ChunkSize:  "8",
		AnpassCmd:  "",
		SpectroCmd: "",
	}
	if !reflect.DeepEqual(Input, after) {
		t.Errorf("\ngot %q\nwad %q\n", Input, after)
	}
}
