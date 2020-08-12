package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Job holds the information for a pbs job
type Job struct {
	Name     string
	Filename string
	Signal   int
	Host     string
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
#PBS -l host={{.Host}}

module load pbspro molpro

export WORKDIR=$PBS_O_WORKDIR
export TMPDIR=/tmp/$USER/$PBS_JOBID
cd $WORKDIR
mkdir -p $TMPDIR

date
echo $HOSTNAME
parallel -j 8 --joblog {{.Filename}}.pl.log --memfree 8g --progress < {{.Filename}} 2> {{.Filename}}.prog
date

rm -rf $TMPDIR
`
// trying --memfree, seemed to have no time cost on 2nd derivatives with plenty of nodes
// now check --progress
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
molpro -t 1 {{.Filename}} --no-xml-output
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
	fmt.Fprintf(f, "%s %s --no-xml-output\n", mapleCmd, infile)
}

// WritePBS writes a pbs infile based on the queue type and
// the templates above, with job information from job
func WritePBS(infile string, job *Job, pbs string) {
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

// Submit submits the pbs script defined by filename to the queue and
// returns the jobid
func Submit(filename string) string {
	// -f option to run qsub in foreground
	out, err := exec.Command("qsub", filename).Output()
	for err != nil {
		time.Sleep(time.Second)
		out, err = exec.Command("qsub", filename).Output()
	}
	jobid := string(out)
	return strings.TrimSuffix(jobid, filepath.Ext(jobid))
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
			if node != nil && node.queue == "workq" && !node.busy {
				nodes = append(nodes, node.name)
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
