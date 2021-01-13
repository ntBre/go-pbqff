package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	*overwrite = true
	MakeDirs("testfiles")
	ParseInfile("testfiles/test.in")
	code := m.Run()
	os.Exit(code)
}

func TestGHAdd(t *testing.T) {
	heap := new(GarbageHeap)
	heap.Add("testname")
	got := heap.heap
	want := []string{"testname.inp", "testname.out"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestMakeDirs(t *testing.T) {
	*overwrite = true
	root := "testfiles"
	got := MakeDirs(root)
	*overwrite = false
	if got != nil {
		t.Errorf("got an error %q, didn't want one", got)
	}
	dirs := []string{"opt", "freq", "pts", "freqs", "pts/inp"}
	for _, dir := range dirs {
		if _, err := os.Stat(root + "/" + dir); os.IsNotExist(err) {
			t.Errorf("failed to create %s in %s\n", dir, root)
		}
	}
	for _, dir := range dirs {
		os.RemoveAll(root + "/" + dir)
	}
}

func TestReadFile(t *testing.T) {
	got, _ := ReadFile("testfiles/read.this")
	want := []string{
		"this is a sample file",
		"to test ReadFile",
		"does it skip leading space",
		"what about trailing",
		"and a blank line",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestMakeName(t *testing.T) {
	tests := []struct {
		geom string
		want string
	}{
		{
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
			geom: ` H          0.0000000000        0.7574590974        0.5217905143
 O          0.0000000000        0.0000000000       -0.0657441568
 H          0.0000000000       -0.7574590974        0.5217905143
`,
			want: "H2O",
		},
	}
	for _, test := range tests {
		got := MakeName(test.geom)
		want := test.want
		if got != want {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	}
}

func TestHandleSignal(t *testing.T) {
	t.Run("received signal", func(t *testing.T) {
		c := make(chan error)
		go func(c chan error) {
			err := HandleSignal(35, 5*time.Second)
			c <- err
		}(c)
		exec.Command("pkill", "-35", "pbqff").Run()
		err := <-c
		if err != nil {
			t.Errorf("did not receive signal")
		}
	})
	t.Run("no signal", func(t *testing.T) {
		c := make(chan error)
		go func(c chan error) {
			err := HandleSignal(35, 50*time.Millisecond)
			c <- err
		}(c)
		exec.Command("pkill", "-34", "go-cart").Run()
		err := <-c
		if err == nil {
			t.Errorf("received signal and didn't want one")
		}
	})
}

func TestGetNames(t *testing.T) {
	prog := Molpro{}
	cart, _, _ := prog.HandleOutput("testfiles/opt")
	got := GetNames(cart)
	want := []string{"N", "H", "H", "H"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func compareFile(file1, file2 string) bool {
	str1, err := ioutil.ReadFile(file1)
	if err != nil {
		panic(err)
	}
	str2, err := ioutil.ReadFile(file2)
	if err != nil {
		panic(err)
	}
	lines1 := strings.Split(string(str1), "\n")
	lines2 := strings.Split(string(str2), "\n")
	if len(lines1) != len(lines2) {
		return false
	}
	for l := range lines1 {
		if lines1[l] != lines2[l] {
			return false
		}
	}
	return true
}

func TestSummarize(t *testing.T) {
	zpt := 100.0
	mph := []float64{1, 2, 3}
	idh := []float64{4, 5, 6}
	sph := []float64{7, 8, 9}
	spf := []float64{10, 11, 12}
	spc := []float64{13, 14, 15}
	t.Run("dimension mismatch", func(t *testing.T) {
		spc := []float64{13, 14, 15, 16}
		err := Summarize(zpt, mph, idh, sph, spf, spc)
		if err == nil {
			t.Errorf("wanted an error, didn't get one")
		}
	})
	t.Run("success", func(t *testing.T) {
		err := Summarize(zpt, mph, idh, sph, spf, spc)
		if err != nil {
			t.Errorf("didn't want an error, got one")
		}
	})
}

func TestUpdateZmat(t *testing.T) {
	tests := []struct {
		msg  string
		load string
		out  string
		geom string
		want string
	}{
		{
			msg:  "maple",
			load: "testfiles/opt.inp",
			out:  "testfiles/nowarn",
			geom: `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`,
			want: `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
}
NH=                  1.91310288 BOHR
XNH=               112.21209367 DEGREE
D1=                119.99647304 DEGREE
`,
		},
		{
			msg:  "sequoia",
			load: "testfiles/opt.inp",
			out:  "testfiles/read/seq",
			geom: `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 90.0  3 90.0
O  1 OX  2 90.0  4 90.0
}
ALX=                 1.20291855 ANG
OX=                  1.26606704 ANG
`,
			want: `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 90.0  3 90.0
O  1 OX  2 90.0  4 90.0
}
ALX=                 1.20291856 ANG
OX=                  1.26606700 ANG
`,
		},
	}
	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			prog, _ := LoadMolpro(test.load)
			_, zmat, _ := prog.HandleOutput(test.out)
			prog.FormatZmat(test.geom)
			prog.UpdateZmat(zmat)
			got := prog.Geometry
			if got != test.want {
				t.Errorf("got\n%q, wanted\n%q\n", got, test.want)
			}
		})
	}
}

func TestXYZGeom(t *testing.T) {
	tests := []struct {
		geom   string
		coords []float64
		names  []string
	}{
		{
			geom: ` 3
 Comment
 H          0.0000000000        0.7574590974        0.5217905143
 O          0.0000000000        0.0000000000       -0.0657441568
 H          0.0000000000       -0.7574590974        0.5217905143
`,
			coords: []float64{
				0.0000000000, 0.7574590974, 0.5217905143,
				0.0000000000, 0.0000000000, -0.0657441568,
				0.0000000000, -0.7574590974, 0.5217905143,
			},
			names: []string{"H", "O", "H"},
		},
		{
			geom: ` H          0.0000000000        0.7574590974        0.5217905143
 O          0.0000000000        0.0000000000       -0.0657441568
 H          0.0000000000       -0.7574590974        0.5217905143
`,
			coords: []float64{
				0.0000000000, 0.7574590974, 0.5217905143,
				0.0000000000, 0.0000000000, -0.0657441568,
				0.0000000000, -0.7574590974, 0.5217905143,
			},
			names: []string{"H", "O", "H"},
		},
	}
	for _, want := range tests {
		names, coords := XYZGeom(want.geom)
		if !reflect.DeepEqual(coords, want.coords) {
			t.Errorf("got %v, wanted %v\n", coords, want.coords)
		}
		if !reflect.DeepEqual(names, want.names) {
			t.Errorf("got %v, wanted %v\n", names, want.names)
		}
	}
}
