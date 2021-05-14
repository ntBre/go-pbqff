package main

import (
	"io/ioutil"
	"strconv"
	"strings"
)

// Resubmit if the returned bools are true

// CheckLog checks a GNU parallel log file, assuming an extension of
// .log, for a jobname
func CheckLog(cmdfile, jobname string) bool {
	ext := ".log"
	logfile := cmdfile + ext
	logbytes, _ := ioutil.ReadFile(logfile)
	return !strings.Contains(string(logbytes), jobname)
}

// CheckProg checks a GNU parallel progress file, assuming an
// extension of .prog, to see if the number of running jobs is less
// than the maximum but greater than zero
func CheckProg(cmdfile string) bool {
	ext := ".prog"
	progfile := cmdfile + ext
	logbytes, _ := ioutil.ReadFile(progfile)
	runes := strings.Map(func(r rune) rune {
		switch r {
		case 0x0D: // ^M dos line ending
			return '\n'
		default:
			return r
		}
	}, string(logbytes))
	lines := strings.Split(runes, "\n")
	var curjobs, maxjobs int
	for _, line := range lines {
		if strings.Contains(line, "1:local") {
			fields := strings.Fields(line)
			maxjobs, _ = strconv.Atoi(fields[len(fields)-1])
		} else if strings.Contains(line, "local:") {
			// TODO becomes more complicated if maxjobs > 9
			// need regexp probably to match whole number
			curjobs, _ = strconv.Atoi(string(line[6]))
		}
	}
	return curjobs < maxjobs && curjobs > 0
}
