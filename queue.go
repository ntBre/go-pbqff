package main

import (
	"fmt"
	"strings"
)

// SelectNode returns a node and queue from the Global node list
func SelectNode() (node, queue string) {
	queue = Conf.Str(Queue)
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
func Push(dir string, pf, count int, calcs []Calc) []Calc {
	subfile := fmt.Sprintf("%s/main%d.pbs", dir, pf)
	jobs := make([]string, 0)
	for c := range calcs {
		calcs[c].SubFile = subfile
		calcs[c].ChunkNum = pf
		if !calcs[c].noRun {
			submitted++
			jobs = append(jobs, calcs[c].Name+".inp")
		} else {
			count++
		}
	}
	node, queue := SelectNode()
	// This should be using the PBS from Config
	WritePBS(subfile,
		&Job{
			Name:     MakeName(Conf.Str(Geometry)) + "pts",
			Filename: subfile,
			Jobs:     jobs,
			Host:     node,
			Queue:    queue,
			NumCPUs:  Conf.Int(NumCPUs),
			PBSMem:   Conf.Int(PBSMem),
		}, ptsMaple)
	jobid := Submit(subfile)
	if *debug {
		fmt.Printf("submitted %s from %s\n", jobid, subfile)
	}
	ptsJobs = append(ptsJobs, jobid)
	paraJobs = append(paraJobs, jobid)
	paraCount[jobid] = Conf.Int(ChunkSize)
	count = 1
	pf++
	// if end reached with no calcs, which can happen on continue
	// from checkpoints
	for c := range calcs {
		calcs[c].JobID = jobid
		calcs[c].SubFile = subfile
	}
	return calcs
}
