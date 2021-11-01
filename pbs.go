package main

import (
	"bufio"
	"embed"
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
	Host     string
	Queue    string
	NumCPUs  int
	PBSMem   int
}

func (j Job) Basename(file string) string {
	return TrimExt(file)
}

//go:embed templates/*
var templates embed.FS
var (
	ptsMaple, _      = template.ParseFS(templates, "templates/ptsMaple.pbs")
	ptsMapleGauss, _ = template.ParseFS(templates, "templates/ptsMapleGauss.pbs")
	pbsMaple, _      = template.ParseFS(templates, "templates/pbsMaple.pbs")
	pbsMapleGauss, _ = template.ParseFS(templates, "templates/pbsMapleGauss.pbs")
	pbsSequoia, _    = template.ParseFS(templates, "templates/pbsSequoia.pbs")
)

type PBS struct {
	SinglePt *template.Template
	ChunkPts *template.Template
}

func (p PBS) SinglePBS() *template.Template {
	return p.SinglePt
}

func (p PBS) ChunkPBS() *template.Template {
	return p.ChunkPts
}

// WritePBS writes a pbs infile based on the queue type and
// the templates above, with job information from job
func (p PBS) WritePBS(infile string, job *Job, pbs *template.Template) {
	f, err := os.Create(infile)
	defer f.Close()
	if err != nil {
		panic(err)
	}
	pbs.Execute(f, job)
}

// Submit submits the pbs script defined by filename to the queue and
// returns the jobid
func (p PBS) Submit(filename string) (jobid string) {
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

// Resubmit copies the input file associated with name to
// name_redo.inp, writes a new PBS file, submits the new PBS job, and
// returns the associated jobid
func (p PBS) Resubmit(name string, err error) string {
	fmt.Fprintf(os.Stderr, "resubmitting %s for %s\n", name, err)
	src, _ := os.Open(name + ".inp")
	dst, _ := os.Create(name + "_redo.inp")
	io.Copy(dst, src)
	defer func() {
		src.Close()
		dst.Close()
	}()
	p.WritePBS(name+"_redo.pbs",
		&Job{
			Name:     "redo",
			Filename: name + "_redo.inp",
			Jobs:     []string{name + "_redo.inp"},
			Host:     "",
			Queue:    "",
			NumCPUs:  Conf.Int(NumCPUs),
			PBSMem:   Conf.Int(PBSMem),
		}, p.SinglePBS())
	return p.Submit(name + "_redo.pbs")
}

// Stat returns a map of job names to their queue status. The map
// value is true if the job is either queued (Q) or running (R) and
// false otherwise
func (p PBS) Stat(qstat *map[string]bool) {
	status, _ := exec.Command("qstat", "-u", os.Getenv("USER")).CombinedOutput()
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
		if strings.Contains(line, "------") {
			header = false
			continue
		} else if header {
			continue
		}
		fields = strings.Fields(line)
		if _, ok := (*qstat)[fields[0]]; ok {
			if strings.Contains("QR", fields[9]) {
				(*qstat)[fields[0]] = true
			}
		}
	}
}

// Clear the PBS queue of the pts jobs
func queueClear(jobs []string) error {
	for _, job := range jobs {
		var host string
		status, err := exec.Command("qstat", "-f", job).Output()
		if err != nil {
			return err
		}
		fields := strings.Fields(string(status))
		for f := range fields {
			if strings.Contains(fields[f], "exec_host") {
				host = strings.Split(fields[f+2], "/")[0]
				break
			}
		}
		if host != "" {
			// I think this doesn't work anymore and it's very slow
			// it's now $USER.jobid.maple
			// out, err := exec.Command("ssh", host, "-t",
			// 	"rm -rf /tmp/$USER/"+job+".maple").CombinedOutput()
			// if *debug {
			// 	fmt.Println("CombinedOutput and error from queueClear: ",
			// 		string(out), err)
			// }
		}
	}
	err := exec.Command("qdel", jobs...).Run()
	return err
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
