package main

import (
	"os"
	"os/exec"
	"regexp"
	"text/template"
	"time"
)

// Job holds the information for a pbs job
type Job struct {
	Name     string
	Filename string
	Signal   int
}

const pbsMaple = `#!/bin/sh
#PBS -N {{.Name}}
#PBS -S /bin/bash
#PBS -j oe
#PBS -o /dev/null
#PBS -W umask=022
#PBS -l walltime=5000:00:00
#PBS -l ncpus=1
#PBS -l mem=9gb

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

// WritePBS writes a pbs infile based on the queue type and
// the templates above, with job information from job
func WritePBS(infile string, job *Job) {
	var t *template.Template
	f, err := os.Create(infile)
	if err != nil {
		panic(err)
	}
	sequoia := regexp.MustCompile(`(?i)sequoia`)
	maple := regexp.MustCompile(`(?i)maple`)
	q := Input[QueueType]
	switch {
	case q == "", maple.MatchString(q):
		t, err = template.New("pbs").Parse(pbsMaple)
	case sequoia.MatchString(q):
		energyLine = "PBQFF(2)"
		energySpace = 2
		t, err = template.New("pbs").Parse(pbsSequoia)
	}
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
