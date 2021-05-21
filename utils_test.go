package main

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCleanSplit(t *testing.T) {
	t.Run("trailing newline", func(t *testing.T) {
		got := CleanSplit("this is\nan\nexample\n", "\n")
		want := []string{"this is", "an", "example"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})

	t.Run("internal newline", func(t *testing.T) {
		got := CleanSplit("this is\nan\n\nexample\n", "\n")
		want := []string{"this is", "an", "example"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
}

func TestRunProgram(t *testing.T) {
	tmp := t.TempDir()
	intder, _ := filepath.Abs("bin/intder")
	infile := "testfiles/write/intder.in"
	base := filepath.Base(infile)
	use := filepath.Join(tmp, base)
	file, err := os.Open(infile)
	defer file.Close()
	if err != nil {
		panic(err)
	}
	usefile, err := os.Create(use)
	defer usefile.Close()
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(usefile, file)
	if err != nil {
		panic(err)
	}
	err = RunProgram(intder, TrimExt(use))
	if err != nil {
		panic(err)
	}
	if _, err := os.Stat(TrimExt(use) + ".out"); os.IsNotExist(err) {
		t.Error("output file not generated")
	}
	// outfile, _ := os.Open(TrimExt(use) + ".out")
	// io.Copy(os.Stdout, outfile)
}

func TestTrimExt(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"test.in", "test"},
	}
	for _, test := range tests {
		got := TrimExt(test.name)
		if got != test.want {
			t.Errorf("got %v, wanted %v\n", got, test.want)
		}
	}
}
