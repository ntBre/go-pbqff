package main

import (
	"os"
	"os/exec"
	"time"
)

type Job struct {
	Name     string
	Filename string
	Signal   int
}

// Write infile based on template tfile
// with job information from job
func WritePBS(infile, tfile string, job *Job) {
	f, err := os.Create(infile)
	if err != nil {
		panic(err)
	}
	t := LoadTemplate(tfile)
	t.Execute(f, job)
}

func Submit(filename string) error {
	// -f option to run qsub in foreground
	_, err := exec.Command("qsub", "-f", filename).Output()
	for err != nil {
		time.Sleep(time.Second)
		_, err = exec.Command("qsub", "-f", filename).Output()
	}
	return nil
}
