package main

import (
	"testing"
	"text/template"
)

func TestParseTemplates(t *testing.T) {
	var err error
	GaussSlurmTmpl, err = template.ParseFS(templates, "templates/gauss/slurm")
	if err != nil {
		t.Error(err)
	}
}
