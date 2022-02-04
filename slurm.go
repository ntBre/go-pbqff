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

// TODO copy g16b01 script and remove requirement that working dir be
// where you submit from - probably just embed it in my template
var (
	ptsSlurm, _ = template.ParseFS(templates, "templates/pts.slurm")
	pbsSlurm, _ = template.ParseFS(templates, "templates/pbs.slurm")
	ptsSlurmGauss, _ = template.ParseFS(templates, "templates/ptsGauss.slurm")
	pbsSlurmGauss, _ = template.ParseFS(templates, "templates/pbsGauss.slurm")
)

type Slurm struct {
	SinglePt *template.Template
	ChunkPts *template.Template
}

func (s *Slurm) SinglePBS() *template.Template {
	return s.SinglePt
}

func (s *Slurm) ChunkPBS() *template.Template {
	return s.ChunkPts
}

func (s *Slurm) NewMolpro() {
	s.SinglePt = pbsSlurm
	s.ChunkPts = ptsSlurm
}

func (s *Slurm) NewGauss() {
	s.SinglePt = pbsSlurmGauss
	s.ChunkPts = ptsSlurmGauss
}

// WritePBS writes a pbs infile based on the queue type and
// the templates above, with job information from job
func (s *Slurm) WritePBS(infile string, job *Job, single bool) {
	f, err := os.Create(infile)
	defer f.Close()
	if err != nil {
		panic(err)
	}
	if single {
		s.SinglePt.Execute(f, job)
	} else {
		s.ChunkPts.Execute(f, job)
	}
}

// Submit submits the pbs script defined by filename to the queue and
// returns the jobid
func (s *Slurm) Submit(filename string) (jobid string) {
	var (
		maxRetries = 15
		maxTime    = 1 << maxRetries
	)
	// -f option to run qsub in foreground
	cmd := exec.Command("sbatch", filename)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	for i := maxRetries; i >= 0 && err != nil; i-- {
		fmt.Printf("Submit: having trouble submitting %s with %v\n", filename, err)
		time.Sleep(time.Second * time.Duration(maxTime>>i))
		cmd := exec.Command("sbatch", filename)
		cmd.Stderr = os.Stderr
		out, err = cmd.Output()
	}
	return strings.TrimSpace(
		strings.ReplaceAll(string(out), "Submitted batch job ", ""))
}

// Resubmit copies the input file associated with name to
// name_redo.inp, writes a new PBS file, submits the new PBS job, and
// returns the associated jobid
func (s *Slurm) Resubmit(name string, err error) string {
	fmt.Fprintf(os.Stderr, "resubmitting %s for %s\n", name, err)
	src, _ := os.Open(name + ".inp")
	dst, _ := os.Create(name + "_redo.inp")
	io.Copy(dst, src)
	defer func() {
		src.Close()
		dst.Close()
	}()
	s.WritePBS(name+"_redo.pbs",
		&Job{
			Name:     "redo",
			Filename: name + "_redo.inp",
			Jobs:     []string{name + "_redo.inp"},
			Host:     "",
			Queue:    "",
			NumCPUs:  Conf.NumCPUs,
			PBSMem:   Conf.PBSMem,
		}, true)
	return s.Submit(name + "_redo.pbs")
}

// Stat returns a map of job names to their queue status. The map
// value is true if the job is either queued (Q) or running (R) and
// false otherwise
func (s *Slurm) Stat(qstat *map[string]bool) {
	status, _ := exec.Command("squeue", "-u", os.Getenv("USER")).CombinedOutput()
	scanner := bufio.NewScanner(strings.NewReader(string(status)))
	var (
		line   string
		fields []string
		header = true
	)
	// initialize them all to false and set true if run
	for key := range *qstat {
		(*qstat)[key] = false
	}
	for scanner.Scan() {
		line = scanner.Text()
		if strings.Contains(line, "JOBID") {
			header = false
			continue
		} else if header {
			continue
		}
		fields = strings.Fields(line)
		if _, ok := (*qstat)[fields[0]]; ok {
			// jobs are initially put in PD = pending
			// state
			if strings.Contains("PDQR", fields[4]) {
				(*qstat)[fields[0]] = true
			}
		}
	}
}
