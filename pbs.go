package main

import (
	"fmt"
	"os"
	"os/exec"
	"text/template"
	"time"
)

// Job holds the information for a pbs job
type Job struct {
	Name     string
	Filename string
	Signal   int
}

const mapleCmd = `molpro -t 1 `

const ptsMaple = `#!/bin/sh
#PBS -N {{.Name}}
#PBS -S /bin/bash
#PBS -j oe
#PBS -o {{.Filename}}.out
#PBS -W umask=022
#PBS -l walltime=5000:00:00
#PBS -l ncpus=8
#PBS -l mem=64gb

module load pbspro molpro

export WORKDIR=$PBS_O_WORKDIR
export TMPDIR=/tmp/$USER/$PBS_JOBID
cd $WORKDIR
mkdir -p $TMPDIR

date
parallel -j 8 < {{.Filename}}
date

rm -rf $TMPDIR
`

const pbsMaple = `#!/bin/sh
#PBS -N {{.Name}}
#PBS -S /bin/bash
#PBS -j oe
#PBS -o /dev/null
#PBS -W umask=022
#PBS -l walltime=5000:00:00
#PBS -l ncpus=1
#PBS -l mem=9gb

module load pbspro molpro

export WORKDIR=$PBS_O_WORKDIR
export TMPDIR=/tmp/$USER/$PBS_JOBID
cd $WORKDIR
mkdir -p $TMPDIR

date
molpro -t 1 {{.Filename}}
date

rm -rf $TMPDIR
ssh -t maple pkill -{{.Signal}} pbqff
`

const pbsSequoia = `#!/bin/sh
#PBS -N {{.Name}}
#PBS -S /bin/bash
#PBS -j oe
#PBS -o /dev/null
#PBS -W umask=022
#PBS -l walltime=5000:00:00
#PBS -l ncpus=1
#PBS -l mem=9gb

module load intel
module load mpt
export PATH=/ptmp/bwhopkin/molpro_mpt/2012/molprop_2012_1_Linux_x86_64_i8/bin:$PATH

export WORKDIR=$PBS_O_WORKDIR
export TMPDIR=/tmp/$USER/$PBS_JOBID
cd $WORKDIR
mkdir -p $TMPDIR

date
mpiexec molpro.exe {{.Filename}}
date

rm -rf $TMPDIR
ssh -t sequoia pkill -{{.Signal}} pbqff
`

func AddCommand(cmdfile, infile string) {
	f, err := os.OpenFile(cmdfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		panic("Cannot open commands file")
	}
	fmt.Fprintf(f, "%s %s\n", mapleCmd, infile)
}

// WritePBS writes a pbs infile based on the queue type and
// the templates above, with job information from job
func WritePBS(infile string, job *Job) {
	var t *template.Template
	f, err := os.Create(infile)
	if err != nil {
		panic(err)
	}
	t, err = template.New("pbs").Parse(pbs)
	if err != nil {
		panic(err)
	}
	t.Execute(f, job)
}

// Submit submits the pbs script defined by filename to the queue
func Submit(filename string) error {
	// -f option to run qsub in foreground
	_, err := exec.Command("qsub", "-f", filename).Output()
	for err != nil {
		time.Sleep(time.Second)
		_, err = exec.Command("qsub", "-f", filename).Output()
	}
	return nil
}
