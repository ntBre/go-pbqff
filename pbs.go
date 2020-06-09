package main

import (
	"os"
	"os/exec"
	"text/template"
	"time"
)

type Job struct {
	Name     string
	Filename string
	Signal   int
}

const pbs = `#!/bin/sh
#PBS -N {{.Name}}
#PBS -S /bin/bash
#PBS -j oe
#PBS -o /dev/null
#PBS -W umask=022
#PBS -l walltime=5000:00:00
#PBS -l ncpus=1
#PBS -l mem=32gb

module load intel
module load mvapich2
module load pbspro
export PATH=/usr/local/apps/molpro/2015.1.35/bin:$PATH

export WORKDIR=$PBS_O_WORKDIR
export TMPDIR=/tmp/$USER/$PBS_JOBID
cd $WORKDIR
mkdir -p $TMPDIR

date
molpro -t 1 {{.Filename}}
date

rm -rf $TMPDIR
ssh -t maple pkill -{{.Signal}} pbqff`

// Write infile based on template
// with job information from job
func WritePBS(infile string, job *Job) {
	f, err := os.Create(infile)
	if err != nil {
		panic(err)
	}
	t, err := template.New("pbs").Parse(pbs)
	if err != nil {
		panic(err)
	}
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
