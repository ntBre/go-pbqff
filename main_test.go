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

func TestMakeDirs(t *testing.T) {
	*overwrite = true
	root := "testfiles"
	got := MakeDirs(root)
	*overwrite = false
	if got != nil {
		t.Errorf("got an error %q, didn't want one", got)
	}
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
	got := ReadFile("testfiles/read.this")
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
	t.Run("original", func(t *testing.T) {
		got := MakeName(Input[Geometry])
		want := "Al2O2"
		if got != want {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
	t.Run("no dummy atoms", func(t *testing.T) {
		keep := Input
		defer func() { Input = keep }()
		ParseInfile("testfiles/prob.in")
		got := MakeName(Input[Geometry])
		want := "CHO"
		if got != want {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
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
	prog := Molpro{
		Geometry: Input[Geometry],
	}
	cart, _, _ := prog.HandleOutput("testfiles/opt")
	got := GetNames(cart)
	want := []string{"N", "H", "H", "H"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestLoadAnpass(t *testing.T) {
	tests := []struct {
		file string
		fmt1 string
		fmt2 string
	}{
		{
			file: "testfiles/load/anpass.small",
			fmt1: "%12.8f",
			fmt2: "%20.12f",
		},
		{
			file: "testfiles/anpass.prob",
			fmt1: "%12.8f",
			fmt2: "%20.12f",
		},
	}
	for _, test := range tests {
		got, _ := LoadAnpass(test.file)
		if got.Fmt1 != test.fmt1 {
			t.Errorf("got %#v, wanted %#v\n", got.Fmt1, test.fmt1)
		}
		if got.Fmt2 != test.fmt2 {
			t.Errorf("got %#v, wanted %#v\n", got.Fmt2, test.fmt2)
		}
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

func TestWriteAnpass(t *testing.T) {
	tests := []struct {
		load  string
		write string
		right string
	}{
		{
			load:  "testfiles/load/anpass.small",
			write: "testfiles/write/anpass.test",
			right: "testfiles/right/anpass.test",
		},
	}
	for _, test := range tests {
		a, _ := LoadAnpass(test.load)
		a.WriteAnpass(test.write, []float64{0, 0, 0, 0, 0, 0})
		if !compareFile(test.write, test.right) {
			t.Errorf("mismatch between %s and %s\n", test.write, test.right)
		}
	}
}

func TestWriteAnpass2(t *testing.T) {
	tests := []struct {
		load  string
		lline string
		write string
		right string
	}{
		{
			load:  "testfiles/load/anpass.small",
			lline: "testfiles/read/anpass1.out",
			write: "testfiles/write/anpass2.test",
			right: "testfiles/right/anpass2.test",
		},
	}
	for _, test := range tests {
		a, _ := LoadAnpass(test.load)
		ll, _ := GetLongLine(test.lline)
		a.WriteAnpass2(test.write, ll, []float64{0, 0, 0, 0, 0, 0})
		if !compareFile(test.write, test.right) {
			t.Errorf("mismatch between %s and %s\n", test.write, test.right)
		}
	}
}

func TestGetLongLine(t *testing.T) {
	got, _ := GetLongLine("testfiles/read/anpass1.out")
	want := `     -0.000879072913     -0.000974769219     -0.000489284859     -0.000744291296      0.000772915057      0.000000000000     -0.000002937018`
	if got != want {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
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
	t.Run("maple", func(t *testing.T) {
		prog, _ := LoadMolpro("testfiles/opt.inp")
		_, zmat, _ := prog.HandleOutput("testfiles/nowarn")
		got := UpdateZmat(FormatZmat(Input[Geometry]), zmat)
		want := `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
}
NH=                  1.91310288 BOHR
XNH=               112.21209367 DEGREE
D1=                119.99647304 DEGREE
`
		if got != want {
			t.Errorf("got\n%q, wanted\n%q\n", got, want)
		}
	})
	t.Run("sequoia", func(t *testing.T) {
		prog, _ := LoadMolpro("testfiles/opt.inp")
		_, zmat, _ := prog.HandleOutput("testfiles/seq")
		start := `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 90.0  3 90.0
O  1 OX  2 90.0  4 90.0
}
ALX=                 1.20291855 ANG
OX=                  1.26606704 ANG
`
		got := UpdateZmat(start, zmat)
		want := `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 90.0  3 90.0
O  1 OX  2 90.0  4 90.0
}
ALX=                 1.20291856 ANG
OX=                  1.26606700 ANG
`
		if got != want {
			t.Errorf("got\n%q, wanted\n%q\n", got, want)
		}
	})
}
