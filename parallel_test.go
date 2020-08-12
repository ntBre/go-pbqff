package main

import "testing"

func TestCheckLog(t *testing.T) {
	tests := []struct {
		msg     string
		cmdfile string
		jobname string
		want    bool
	}{
		{"job not found", "testfiles/read/commands37.txt",
			"pts/inp/job.0000002862", true},
		{"job found", "testfiles/read/commands35.txt",
			"pts/inp/job.0000002694", false},
	}
	for _, test := range tests {
		got := CheckLog(test.cmdfile, test.jobname)
		if got != test.want {
			t.Errorf("CheckLog(%q): got %v, wanted %v\n",
				test.msg, got, test.want)
		}
	}
}

func TestCheckProg(t *testing.T) {
	tests := []struct {
		msg     string
		cmdfile string
		want    bool
	}{
		{"stuck", "testfiles/read/commands37.txt", true},
		{"still running", "testfiles/read/intermediate.txt", false},
		{"finished", "testfiles/read/commands35.txt", false},
	}
	for _, test := range tests {
		got := CheckProg(test.cmdfile)
		if got != test.want {
			t.Errorf("CheckProg(%q): got %v, wanted %v\n",
				test.msg, got, test.want)
		}
	}
}
