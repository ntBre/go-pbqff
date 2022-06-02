package main

import (
	"fmt"
	"strings"
)

type Queue interface {
	WritePBS(string, *Job)
	Submit(string) string
	Resubmit(string, error) string
	Stat(*map[string]bool)
	NewMolpro()
	NewGauss()
	NewMopac()
}

// SelectNode returns a node and queue from the Global node list
func SelectNode() (node, queue string) {
	queue = Conf.WorkQueue
	// regenerate empty node list if empty
	if len(Global.Nodes) == 0 {
		Global.Nodes = PBSnodes()
	}
	if len(Global.Nodes) > 0 {
		tmp := strings.Split(Global.Nodes[0], ":")
		if queue == "" || tmp[0] == queue {
			node = tmp[1]
			queue = tmp[0]
			Global.Nodes = Global.Nodes[1:]
		}
	}
	return
}

// Push sends calculations to the queue
func Push(q Queue, dir string, pf int, calcs []Calc) []Calc {
	subfile := fmt.Sprintf("%s/main%d.pbs", dir, pf)
	jobs := make([]string, 0)
	for c := range calcs {
		calcs[c].SubFile = subfile
		calcs[c].ChunkNum = pf
		if !calcs[c].noRun {
			Global.Submitted++
			jobs = append(jobs, calcs[c].Name+".inp")
		}
	}
	if len(jobs) > 0 {
		node, queue := SelectNode()
		// This should be using the PBS from Config
		q.WritePBS(subfile,
			&Job{
				Name:     MakeName(Conf.Geometry) + "pts",
				Filename: subfile,
				Jobs:     jobs,
				Host:     node,
				Queue:    queue,
				NumCPUs:  Conf.NumCPUs,
				PBSMem:   Conf.PBSMem,
			})
		jobid := q.Submit(subfile)
		if *debug {
			fmt.Printf("submitted %s from %s\n", jobid, subfile)
		}
		Global.WatchedJobs = append(Global.WatchedJobs, jobid)
		pf++
		// if end reached with no calcs, which can happen on continue
		// from checkpoints
		for c := range calcs {
			calcs[c].JobID = jobid
			calcs[c].SubFile = subfile
		}
	}
	return calcs
}
