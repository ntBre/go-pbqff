package main

import (
	"os"
	"strconv"
	"strings"
)

// Resubmit if the returned bools are true:

// file not in logfile => job is not finished => job is running or not
// started

// numjobs < maxjobs in progfile => all jobs have been started => job
// is running

// ErrFileNotFound => job is not actually running so we need to
// resubmit

// CheckLog checks a GNU parallel log file, assuming an extension of
// .log, for a jobname
func CheckLog(cmdfile, jobname string) bool {
	ext := ".log"
	logfile := cmdfile + ext
	logbytes, _ := os.ReadFile(logfile)
	return !strings.Contains(string(logbytes), jobname)
}

// CheckProg checks a GNU parallel progress file, assuming an
// extension of .prog, to see if the number of running jobs is less
// than the maximum but greater than zero
func CheckProg(cmdfile string) bool {
	ext := ".prog"
	progfile := cmdfile + ext
	logbytes, _ := os.ReadFile(progfile)
	lines := strings.Split(string(logbytes), "\x0D")
	var curjobs, maxjobs int
	for _, line := range lines {
		// after splitting on ^M, all of the header
		// information is on the same "line" so take the 14th
		// item
		if strings.Contains(line, "1:local") {
			fields := strings.Fields(line)
			maxjobs, _ = strconv.Atoi(fields[13])
		} else if strings.Contains(line, "local:") {
			// TODO becomes more complicated if maxjobs > 9
			// need regexp probably to match whole number
			curjobs, _ = strconv.Atoi(string(line[6]))
		}
	}
	return curjobs < maxjobs && curjobs > 0
}
