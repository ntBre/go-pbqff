package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestProcessInput(t *testing.T) {
	tmp := Config
	defer func() {
		Config = tmp
	}()
	Config[Program].Value = "test"
	ProcessInput("program=notatest")
	if Config[Program].Value != "notatest" {
		t.Errorf("got %q, wanted %q\n",
			Config[Program].Value, "notatest")
	}
}

func compConf(kws []interface{}, conf Conf) (bool, string) {
	for k := range kws {
		if !reflect.DeepEqual(kws[k], conf.At(k)) {
			return false,
				fmt.Sprintf("At %s, %v != %v\n",
					Key(k), kws[k], conf.At(k))
		}
	}
	return true, ""
}

func TestParseInfile(t *testing.T) {
	tmp := Config
	defer func() {
		Config = tmp
	}()
	ParseInfile("testfiles/test.in")
	want := []interface{}{
		Cluster: "maple",
		Program: "molpro",
		Geometry: `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`,
		Delta:      0.005,
		GeomType:   "zmat",
		ChunkSize:  8,
		Deriv:      4,
		JobLimit:   1024,
		NumJobs:    8,
		CheckInt:   100,
		Queue:      "",
		SleepInt:   60,
		IntderCmd:  "/home/brent/Packages/intder/intder",
		AnpassCmd:  "",
		SpectroCmd: "",
	}
	if ok, msg := compConf(want, Config); !ok {
		t.Errorf(msg)
	}
}

