package main

import (
	"bytes"
	"os"
	"reflect"
	"testing"
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
	q := PBS{
		SinglePt: pbsSequoia,
	}
	q.WritePBS(write, &p, true)
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
	want := `#!/bin/sh
#PBS -N test
#PBS -S /bin/bash
#PBS -j oe
#PBS -o .out
#PBS -W umask=022
#PBS -l walltime=5000:00:00
#PBS -l ncpus=1
#PBS -l mem=0gb

scrdir=/tmp/$USER.$PBS_JOBID

mkdir -p $scrdir
export GAUSS_SCRDIR=$scrdir
export OMP_NUM_THREADS=1

echo "exec_host = $HOSTNAME"

if [[ $HOSTNAME =~ cn([0-9]{3}) ]];
then
  nodenum=${BASH_REMATCH[1]};
  nodenum=$((10#$nodenum));
  echo $nodenum

  if (( $nodenum <= 29 ))
  then
    echo "Using AVX version";
    export g16root=/usr/local/apps/gaussian/g16-c01-avx/
  elif (( $nodenum > 29 ))
  then
    echo "Using AVX2 version";
    export g16root=/usr/local/apps/gaussian/g16-c01-avx2/
  else
    echo "Unexpected condition!"
    exit 1;
  fi
else
  echo "Not on a compute node!"
  exit 1;
fi

cd $PBS_O_WORKDIR
. $g16root/g16/bsd/g16.profile

date
hostname

g16 first.com
formchk first.chk first.fchk
g16 second.com
formchk second.chk second.fchk
g16 third.com
formchk third.chk third.fchk
date

rm -rf $TMPDIR
`
	var buf bytes.Buffer
	ptsMapleGauss.Execute(&buf, Job{
		Name: "test",
		Jobs: []string{"first.com", "second.com", "third.com"},
	})
	got := buf.String()
	if got != want {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}
