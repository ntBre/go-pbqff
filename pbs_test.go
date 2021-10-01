package main

import (
	"os"
	"reflect"
	"testing"
	"text/template"
)

func TestWritePBS(t *testing.T) {
	p := Job{
		Name:     "Al2O2",
		Filename: "opt.inp",
		Host:     "",
		Queue:    "",
		NumCPUs:  8,
		PBSMem:   8,
	}
	write := "testfiles/write/mp.pbs"
	right := "testfiles/right/mp.pbs"
	WritePBS(write, &p, pbsSequoia)
	if !compareFile(write, right) {
		t.Errorf("mismatch between %s and %s\n(diff %[1]q %[2]q)", right, write)
	}
}

func TestReadPBSNodes(t *testing.T) {
	// cn074 has 6 jobs
	f, _ := os.Open("testfiles/read/pbsnodes")
	defer f.Close()
	got := readPBSnodes(f)
	want := []string{"workq:cn064", "workq:cn065", "workq:cn066", "workq:cn067"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, wanted %q\n", got, want)
	}
}

func TestTemplate(t *testing.T) {
	tmpl, err := template.New("pbs").Parse(ptsMapleGauss)
	if err != nil {
		t.Errorf("template failed:  %v\n", err)
	}
	tmpl.Execute(os.Stdout, Job{
		Jobs: []string{"first.com", "second.com", "third.com"},
	})
}
