package main

import _ "embed"
import "os/exec"

//go:embed embed/taylor.py
var taylor string

func Taylor() {
	flags := ""
	cmd := exec.Command("python2", "-c", taylor, flags)
	cmd.Run()
}
