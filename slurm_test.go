package main

import (
	"testing"
	"text/template"
)

func TestParseTemplates(t *testing.T) {
	var err error
	ptsSlurmGauss, err = template.ParseFS(templates, "templates/ptsGauss.slurm")
	if err != nil {
		t.Error(err)
	}
	pbsSlurmGauss, err = template.ParseFS(templates, "templates/pbsGauss.slurm")
	if err != nil {
		t.Error(err)
	}
}
