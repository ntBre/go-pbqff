package main

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCleanSplit(t *testing.T) {
	tests := []struct {
		msg  string
		inp  string
		sep  string
		want []string
	}{
		{
			msg:  "trailing newline",
			inp:  "this is\nan\nexample\n",
			sep:  "\n",
			want: []string{"this is", "an", "example"},
		},
		{
			msg:  "internal newline",
			inp:  "this is\nan\n\nexample\n",
			sep:  "\n",
			want: []string{"this is", "an", "example"},
		},
	}

	for _, test := range tests {
		got := CleanSplit(test.inp, test.sep)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("%s: got %v, wanted %v\n",
				test.msg, got, test.want)
		}
	}
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

func TestMakeName(t *testing.T) {
	tmp := Conf
	defer func() {
		Conf = tmp
	}()
	tests := []struct {
		gtype string
		geom  string
		want  string
	}{
		{
			gtype: "zmat",
			geom: ` X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0

AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg
`,
			want: "Al2O2",
		},
		{
			gtype: "zmat",
			geom: ` X
C 1 1.0
O 2 co 1 90.0
H 2 ch 1 90.0 3 180.0

co=                  1.10797263 ANG
ch=                  1.09346324 ANG
`,
			want: "CHO",
		},
		{
			gtype: "xyz",
			geom: ` H          0.0000000000        0.7574590974        0.5217905143
 O          0.0000000000        0.0000000000       -0.0657441568
 H          0.0000000000       -0.7574590974        0.5217905143
`,
			want: "H2O",
		},
		{
			gtype: "xyz",
			geom: ` 3
 Comment
 H          0.0000000000        0.7574590974        0.5217905143
 O          0.0000000000        0.0000000000       -0.0657441568
 H          0.0000000000       -0.7574590974        0.5217905143
`,
			want: "H2O",
		},
	}
	for _, test := range tests {
		got := MakeName(test.geom)
		Conf.GeomType =  test.gtype
		want := test.want
		if got != want {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	}
}
