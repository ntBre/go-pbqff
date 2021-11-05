package main

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"text/template"
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

func compareFile(file1, file2 string) bool {
	str1, err := os.ReadFile(file1)
	if err != nil {
		panic(err)
	}
	str2, err := os.ReadFile(file2)
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

func TestXYZGeom(t *testing.T) {
	prog := Molpro{}
	cart, _, _ := prog.HandleOutput("testfiles/opt")
	tests := []struct {
		geom   string
		coords []float64
		names  []string
	}{
		{
			geom:  cart,
			names: []string{"N", "H", "H", "H"},
			coords: []float64{
				-0.000015401, 0.000000000, -0.128410266,
				1.771141454, 0.000000000, 0.594753622,
				-0.885463720, 1.533900737, 0.594841015,
				-0.885463720, -1.533900737, 0.594841015,
			},
		},
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

func TestCartPoints(t *testing.T) {
	tests := []struct {
		nc   int
		want int
	}{
		{9, 5784},
		{18, 79500},
	}
	for _, test := range tests {
		got := CartPoints(test.nc)
		if got != test.want {
			t.Errorf("got %v, wanted %v\n", got, test.want)
		}
	}
}

func TestGradPoints(t *testing.T) {
	tests := []struct {
		nc   int
		want int
	}{
		{9, 1320},
		{18, 9120},
	}
	for _, test := range tests {
		got := GradPoints(test.nc)
		if got != test.want {
			t.Errorf("got %v, wanted %v\n", got, test.want)
		}
	}
}

func TestDoSIC(t *testing.T) {
	f := flags
	defer func() {
		flags = f
	}()
	tests := []struct {
		msg   string
		flags int
		want  bool
	}{
		{"grad", GRAD, false},
		{"opt grad", OPT | GRAD, false},
		{"cart", CART, false},
		{"opt", OPT, true},
		{"opt cart", OPT | CART, false},
		{"pts", PTS, true},
		{"freqs", FREQS, true},
		{"full SIC", OPT | PTS | FREQS, true},
		{"zero", 0, true},
	}
	for _, test := range tests {
		flags = 0
		flags |= test.flags
		if got := DoSIC(); got != test.want {
			t.Errorf("DoSIC(%s): got %v, wanted %v\n",
				test.msg, got, test.want)
		}
	}
}

type TestQueue struct {
	SinglePt *template.Template
	ChunkPts *template.Template
}

func (tq TestQueue) WritePBS(infile string, job *Job, pbs *template.Template) {
	f, err := os.Create(infile)
	defer f.Close()
	if err != nil {
		panic(err)
	}
	pbs.Execute(f, job)
}

func (tq TestQueue) SinglePBS() *template.Template { return tq.SinglePt }
func (tq TestQueue) ChunkPBS() *template.Template  { return tq.ChunkPts }
func (tq TestQueue) Submit(string) string          { return "1" }
func (tq TestQueue) Resubmit(string, error) string { return "" }
func (tq TestQueue) Stat(*map[string]bool)         {}

func TestDrain(t *testing.T) {
	submitted = 0
	conf := Conf
	qsub = "qsub/qsub"
	defer func() {
		Conf = conf
		qsub = "qsub"
	}()
	Conf.Set(JobLimit, 128)
	Conf.Set(Deltas, []float64{
		0.005, 0.005, 0.005,
		0.005, 0.005, 0.005,
	})
	Conf.Set(SleepInt, 0)
	Conf.Set(ChunkSize, 64)
	paraCount = make(map[string]int)
	prog := new(Molpro)
	ncoords := 6
	E0 := 0.0
	cf := []CountFloat{
		{1.0, 1, false},
	}
	calcs := []Calc{
		{
			// find energy in ReadOut
			Name:     "testfiles/opt",
			ChunkNum: 0,
		},
		{
			// name contains E0
			Name:     "some/job/E0",
			ChunkNum: 0,
		},
		{
			// .Result set
			Name:     "some/job",
			Result:   3.14,
			ChunkNum: 0,
		},
		{
			// .Src set
			Name: "some/job",
			Src: &Source{
				Index: 0,
				Slice: &cf,
			},
			ChunkNum: 0,
		},
	}
	dir := t.TempDir()
	queue := TestQueue{
		SinglePt: pbsMaple,
		ChunkPts: ptsMaple,
	}
	gen := func() ([]Calc, bool) {
		return Push(queue, dir, 0, 0, calcs), false
	}
	errMap = make(map[error]int)
	min, time := Drain(prog, queue, ncoords, E0, gen)
	wmin, wtime := -56.499802779375, 867.46
	if min != wmin {
		t.Errorf("got %v, wanted %v\n", min, wmin)
	}
	if time != wtime {
		t.Errorf("got %v, wanted %v\n", time, wtime)
	}
}
