package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"
)

var (
	qsub = "qsub"
)

// Job holds the information for a pbs job
type Job struct {
	Name     string
	Filename string
	Jobs     []string
	Signal   int
	Host     string
	Queue    string
	NumCPUs  int
	PBSMem   int
}

const mapleCmd = `molpro -t 1`

const ptsMaple = `#!/bin/sh
#PBS -N {{.Name}}
#PBS -S /bin/bash
#PBS -j oe
#PBS -o {{.Filename}}.out
#PBS -W umask=022
#PBS -l walltime=5000:00:00
#PBS -l ncpus=1
#PBS -l mem={{.PBSMem}}gb
{{- if .Queue}}
#PBS -q {{.Queue}}
{{- end}}
{{- if .Host}}
#PBS -l host={{.Host}}
{{- end}}

module load pbspro molpro

export WORKDIR=$PBS_O_WORKDIR
export TMPDIR=/tmp/$USER/$PBS_JOBID
cd $WORKDIR
mkdir -p $TMPDIR

date
hostname
{{range $j := .Jobs}}
molpro -t 1 {{ $j }} --no-xml-output
{{- end }}
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
molpro -t 1 --no-xml-output {{.Filename}}
date

rm -rf $TMPDIR
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
`

// WritePBS writes a pbs infile based on the queue type and
// the templates above, with job information from job
func WritePBS(infile string, job *Job, pbs string) {
	var t *template.Template
	f, err := os.Create(infile)
	defer f.Close()
	if err != nil {
		panic(err)
	}
	t, err = template.New("pbs").Parse(pbs)
	if err != nil {
		panic(err)
	}
	t.Execute(f, job)
}

// Submit submits the pbs script defined by filename to the queue and
// returns the jobid
var Submit = func(filename string) string {
	var (
		maxRetries = 15
		maxTime    = 1 << maxRetries
	)
	// -f option to run qsub in foreground
	cmd := exec.Command(qsub, "-f", filename)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	for i := maxRetries; i >= 0 && err != nil; i-- {
		fmt.Printf("Submit: having trouble submitting %s with %v\n", filename, err)
		time.Sleep(time.Second * time.Duration(maxTime>>i))
		cmd := exec.Command(qsub, "-f", filename)
		cmd.Stderr = os.Stderr
		out, err = cmd.Output()
	}
	return strings.TrimSpace(string(out))
}

// PBSnodes runs the pbsnodes -a command and returns a list of free
// nodes
func PBSnodes() []string {
	out, _ := exec.Command("pbsnodes", "-a").Output()
	return readPBSnodes(strings.NewReader(string(out)))
}

type cnode struct {
	name  string
	queue string
	busy  bool
}

func readPBSnodes(r io.Reader) (nodes []string) {
	scanner := bufio.NewScanner(r)
	var (
		line string
		init bool = true
		node *cnode
	)
	for scanner.Scan() {
		line = scanner.Text()
		switch {
		case line == "" || init:
			if node != nil &&
				(node.queue == "workq" || node.queue == "r410") &&
				!node.busy {
				nodes = append(nodes, node.queue+":"+node.name)
			}
			node = new(cnode)
			init = false
		case strings.Contains(line, "resources_available.host"):
			f := strings.Fields(line)
			node.name = f[len(f)-1]
		case strings.Contains(line, "resources_available.Qlist"):
			f := strings.Fields(line)
			node.queue = f[len(f)-1]
		case strings.Contains(line, "jobs = "):
			node.busy = true
		case strings.Contains(line, "state = "):
			f := strings.Fields(line)
			if f[len(f)-1] != "free" {
				node.busy = true
			}
		}
	}
	// process last file at the end
	if node != nil && node.queue == "workq" && !node.busy {
		nodes = append(nodes, node.name)
	}
	return
}
