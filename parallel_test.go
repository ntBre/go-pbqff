package main

import "testing"

func TestCheckLog(t *testing.T) {
	tests := []struct {
		msg     string
		cmdfile string
		jobname string
		want    bool
	}{
		{
			msg:     "job not found",
			cmdfile: "testfiles/read/commands37.txt",
			jobname: "pts/inp/job.0000002862",
			want:    true,
		},
		{
			msg:     "job found",
			cmdfile: "testfiles/read/commands35.txt",
			jobname: "pts/inp/job.0000002694",
			want:    false,
		},
	}
	for _, test := range tests {
		got := CheckLog(test.cmdfile, test.jobname)
		if got != test.want {
			t.Errorf("CheckLog(%q): got %v, wanted %v\n",
				test.msg, got, test.want)
		}
	}
}

func BenchmarkCheckLog(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CheckLog(
			"testfiles/read/commands37.txt",
			"pts/inp/job.0000002694",
		)
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

func BenchmarkCheckProg(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CheckProg("testfiles/read/commands37.txt")
	}
}
