package main

import (
	"io"
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
	OPT = true
	defer func() {
		OPT = false
	}()
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
	tests := []struct {
		msg string
		zpt float64
		mph []float64
		idh []float64
		sph []float64
		spf []float64
		spc []float64
		err bool
	}{
		{
			msg: "dimension mismatch",
			zpt: 100.0,
			mph: []float64{1, 2, 3},
			idh: []float64{4, 5, 6},
			sph: []float64{7, 8, 9},
			spf: []float64{10, 11, 12},
			spc: []float64{13, 14, 15, 16},
			err: true,
		},
		{
			msg: "success",
			zpt: 100.0,
			mph: []float64{1, 2, 3},
			idh: []float64{4, 5, 6},
			sph: []float64{7, 8, 9},
			spf: []float64{10, 11, 12},
			spc: []float64{13, 14, 15},
		},
	}
	for _, test := range tests {
		err := Summarize(io.Discard, test.zpt, test.mph, test.idh, test.sph,
			test.spf, test.spc)
		if err == nil && test.err || err != nil && !test.err {
			t.Errorf("%s: wanted an error, didn't get one", test.msg)
		}
	}
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

type TestQueue struct {
	SinglePt *template.Template
	ChunkPts *template.Template
}

func (tq TestQueue) WritePBS(infile string, job *Job, single bool) {
	f, err := os.Create(infile)
	defer f.Close()
	if err != nil {
		panic(err)
	}
	if single {
		tq.SinglePt.Execute(f, job)
	} else {
		tq.ChunkPts.Execute(f, job)
	}
}

func (tq TestQueue) SinglePBS() *template.Template { return tq.SinglePt }
func (tq TestQueue) ChunkPBS() *template.Template  { return tq.ChunkPts }
func (tq TestQueue) Submit(string) string          { return "1" }
func (tq TestQueue) Resubmit(string, error) string { return "" }
func (tq TestQueue) Stat(*map[string]bool)         {}
func (tq TestQueue) NewGauss()                     {}
func (tq TestQueue) NewMolpro()                    {}

func TestDrain(t *testing.T) {
	Global.Submitted = 0
	conf := Conf
	qsub = "qsub/qsub"
	defer func() {
		Conf = conf
		qsub = "qsub"
	}()
	Conf.JobLimit = 128
	Conf.Deltas = []float64{
		0.005, 0.005, 0.005,
		0.005, 0.005, 0.005,
	}
	Conf.SleepInt = 0
	Conf.ChunkSize = 64
	prog := new(Molpro)
	ncoords := 6
	E0 := 0.0
	// TODO reinstate with below
	// cf := []CountFloat{
	// 	{1.0, 1, false},
	// }
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
		// TODO reinstate this
		// {
		// 	// .Src set
		// 	Name: "some/job",
		// 	Src: &Source{
		// 		Index: 0,
		// 		Slice: &cf,
		// 	},
		// 	ChunkNum: 0,
		// },
	}
	dir := t.TempDir()
	queue := TestQueue{
		SinglePt: pbsMaple,
		ChunkPts: ptsMaple,
	}
	gen := func() ([]Calc, bool) {
		return Push(queue, dir, 0, 0, calcs), false
	}
	Global.ErrMap = make(map[error]int)
	min, time := Drain(prog, queue, ncoords, E0, gen)
	wmin, wtime := -56.499802779375, 867.46
	if min != wmin {
		t.Errorf("got %v, wanted %v\n", min, wmin)
	}
	if time != wtime {
		t.Errorf("got %v, wanted %v\n", time, wtime)
	}
}
