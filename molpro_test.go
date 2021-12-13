package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	symm "github.com/ntBre/chemutils/symmetry"
)

func TestLoadMolpro(t *testing.T) {
	got, _ := LoadMolpro("testfiles/load/molpro.in")
	want := &Molpro{
		Head: `memory,995,m   ! 30GB 12procs

gthresh,energy=1.d-12,zero=1.d-22,oneint=1.d-22,twoint=1.d-22;
gthresh,optgrad=1.d-8,optstep=1.d-8;
nocompress;

geometry={
`,
		Geom: "",
		Tail: `basis={
default,cc-pvdz-f12
}
set,charge=0
set,spin=0
hf,accuracy=16,energy=1.0d-10
{ccsd(t)-f12,thrden=1.0d-8,thrvar=1.0d-10;orbital,IGNORE_ERROR;}
`,
		Opt: `{optg,grms=1.d-8,srms=1.d-8}
`,
		Extra: `pbqff=energy
`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%#v, wanted\n%#v\n", got, want)
	}
}

func TestFormatZmat(t *testing.T) {
	m := new(Molpro)
	m.FormatZmat(
		`X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`,
	)
	got := m.Geom
	want := `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
}
AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
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
		prog, _ := LoadMolpro(test.load)
		_, zmat, _ := prog.HandleOutput(test.out)
		prog.FormatZmat(test.geom)
		prog.UpdateZmat(zmat)
		got := prog.Geom
		if got != test.want {
			t.Errorf("%s: got\n%q, wanted\n%q\n",
				test.msg, got, test.want)
		}
	}
}

// TODO test freq procedure
func TestWriteInput(t *testing.T) {
	tests := []struct {
		load  string
		write string
		right string
		geom  string
		proc  Procedure
	}{
		{
			load:  "testfiles/load/molpro.in",
			write: "testfiles/write/opt.inp",
			right: "testfiles/right/opt.inp",
			geom: `X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0
AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`,
			proc: opt,
		},
	}
	for _, test := range tests {
		mp, _ := LoadMolpro(test.load)
		mp.FormatZmat(test.geom)
		mp.WriteInput(test.write, test.proc)
		if !compareFile(test.write, test.right) {
			t.Errorf("mismatch between %s and %s\n"+
				"(diff %#[1]q %#[2]q)\n", test.write, test.right)
		}
	}
}

func TestReadOut(t *testing.T) {
	m := Molpro{}
	temp := Conf.RE(EnergyLine)
	defer func() {
		Conf.Set(EnergyLine, temp)
	}()
	tests := []struct {
		msg      string
		filename string
		eline    *regexp.Regexp
		energy   float64
		time     float64
		grad     []float64
		err      error
	}{
		{
			msg:      "Gradient success",
			filename: "testfiles/read/showgrad.out",
			energy:   -152.379641595220,
			time:     28.92,
			grad: []float64{
				0.038675130622946, 0.002051946183374, 0.015073821827216,
				0.115196189670610, 0.146068323479018, 0.149725120162171,
				0.017871578588965, 0.009170027453292, 0.010289031057109,
				-0.019700971149367, -0.092155534025566, -0.096689855587206,
				-0.141218912305456, 0.057539789396310, -0.098009841881230,
				-0.010823015427707, -0.122674552486430, 0.019611724421961,
			},
			err: nil,
		},
		{
			msg:      "Normal success",
			filename: "testfiles/read/good.out",
			energy:   -168.463747095015,
			time:     10372.08,
			err:      nil,
		},
		{
			msg:      "Error in output",
			filename: "testfiles/read/error.out",
			energy:   math.NaN(),
			time:     119.29,
			err:      ErrFileContainsError,
		},
		{
			msg:      "File not found",
			filename: "nonexistent/file",
			energy:   math.NaN(),
			time:     0.0,
			err:      ErrFileNotFound,
		},
		{
			msg:      "One-line error",
			filename: "testfiles/read/shortcircuit.out",
			energy:   math.NaN(),
			time:     0.0,
			err:      ErrFileContainsError,
		},
		{
			msg:      "Blank file",
			filename: "testfiles/read/blank.out",
			energy:   math.NaN(),
			time:     0.0,
			err:      ErrBlankOutput,
		},
		{
			msg:      "Parse error",
			filename: "testfiles/read/parse.out",
			energy:   math.NaN(),
			time:     10372.08,
			err:      ErrFinishedButNoEnergy,
		},
		{
			msg:      "Sequoia partial",
			filename: "testfiles/read/seq.part",
			energy:   math.NaN(),
			time:     67.94,
			err:      ErrEnergyNotFound,
		},
		{
			msg:      "Sequoia success",
			filename: "testfiles/read/seq.out",
			eline:    regexp.MustCompile(`PBQFF\(2\)`),
			energy:   -634.43134170,
			time:     1075.84,
			err:      nil,
		},
		{
			msg:      "cccr success",
			filename: "testfiles/read/cccr.out",
			eline:    regexp.MustCompile(`^\s*CCCRE\s+=`),
			energy:   -56.591603910177,
			time:     567.99,
			err:      nil,
		},
	}
	for _, test := range tests {
		if test.eline != nil {
			Conf.Set(EnergyLine, test.eline)
		} else {
			Conf.Set(EnergyLine, regexp.MustCompile(`energy=`))
		}
		energy, time, grad, err := m.ReadOut(test.filename)
		if math.IsNaN(test.energy) {
			if !math.IsNaN(energy) {
				t.Errorf("got not NaN, wanted NaN\n")
			}
		} else if energy != test.energy {
			t.Errorf("got %v, wanted %v\n", energy, test.energy)
		}
		if time != test.time {
			t.Errorf("got %v, wanted %v\n", time, test.time)
		}
		if !reflect.DeepEqual(grad, test.grad) {
			t.Errorf("got %#+v, wanted %#+v\n", grad, test.grad)
		}
		if err != test.err {
			t.Errorf("got %v, wanted %v\n", err, test.err)
		}
	}
}

func BenchmarkReadOut(b *testing.B) {
	m := Molpro{}
	// msg:      "Normal success",
	filename := "testfiles/read/good.out"
	for i := 0; i < b.N; i++ {
		m.ReadOut(filename)
	}
}

func TestHandleOutput(t *testing.T) {
	qsub = "qsub/qsub"
	defer func() {
		qsub = "qsub"
	}()
	mp := new(Molpro)
	mp.FormatZmat(`X
X 1 1.0
Al 1 AlX 2 90.0
Al 1 AlX 2 90.0 3 180.0
O  1 OX  2 XXO  3 90.0
O  1 OX  2 XXO  4 90.0

AlX = 0.85 Ang
OX = 1.1 Ang
XXO = 80.0 Deg`)
	tests := []struct {
		msg  string
		file string
		err  error
	}{
		{
			msg:  "warning in outfile",
			file: "testfiles/opt",
			err:  nil,
		},
		{
			msg:  "no warning, normal case",
			file: "testfiles/nowarn",
			err:  nil,
		},
		{
			msg:  "Error in output",
			file: "testfiles/read/error",
			err:  ErrFileContainsError,
		},
	}
	for _, test := range tests {
		_, _, err := mp.HandleOutput(test.file)
		if err != test.err {
			t.Errorf("%s: got %q, wanted %q\n", test.msg, err, test.err)
		}
	}
}

func TestReadLog(t *testing.T) {
	tests := []struct {
		name string
		log  string
		cart string
		zmat string
	}{
		{
			name: "maple",
			log:  "testfiles/coords.log",
			cart: `O 1.000000000 0.118481857 -2.183553663
H 0.000000000 -1.563325812 -2.884671935
C 0.000000000 -0.014536611 0.273763522
N 0.000000000 -0.010373662 2.467030139
`,
			zmat: `OH=                  0.96421314 ANG
OC=                  1.30226003 ANG
HOC=               109.53197453 DEG
CN=                  1.16062880 ANG
OCN=               176.79276221 DEG
`,
		},
		{
			name: "sequoia",
			log:  "testfiles/read/seq.log",
			cart: `AL 0.000000000 0.000000000 2.273186636
AL 0.000000000 0.000000000 -2.273186636
O 0.000000000 2.392519895 0.000000000
O 0.000000000 -2.392519895 0.000000000
`,
			zmat: `ALX=                 1.20291856 ANG
OX=                  1.26606700 ANG
`,
		},
	}
	for _, test := range tests {
		cart, zmat := ReadLog(test.log)
		if cart != test.cart {
			t.Errorf("%s: got %v, wanted %v\n",
				test.name, cart, test.cart)
		}
		if zmat != test.zmat {
			t.Errorf("%s: got %v, wanted %v\n",
				test.name, zmat, test.zmat)
		}
	}
}

func TestReadFreqs(t *testing.T) {
	mp := Molpro{}
	got := mp.ReadFreqs("testfiles/freq.out")
	want := []float64{805.31, 774.77, 679.79, 647.70, 524.26, 301.99}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func dumpCalc(calcs []Calc) string {
	byt, _ := json.MarshalIndent(calcs, "", "\t")
	return string(byt)
}

// TODO test write=false case
func TestBuildPoints(t *testing.T) {
	queue := TestQueue{
		SinglePt: pbsMaple,
		ChunkPts: ptsMaple,
	}
	qsub = "qsub/qsub"
	defer func() {
		qsub = "qsub"
		cenergies = *new([]CountFloat)
	}()
	prog, _ := LoadMolpro("testfiles/load/molpro.in")
	cart, _, _ := prog.HandleOutput("testfiles/opt")
	names, _ := XYZGeom(cart)
	os.Mkdir("testfiles/read/inp", 0755)
	defer os.RemoveAll("testfiles/read/inp")
	paraCount = make(map[string]int)
	gen := BuildPoints(prog, queue, "testfiles/read/file07", names, true)
	got, _ := gen()
	want := []Calc{
		{
			Name:    "testfiles/read/inp/NHHH.00000",
			Targets: []Target{{1, &cenergies, 0}},
			SubFile: "testfiles/read/inp/main0.pbs",
			JobID:   "1",
			Scale:   1},
		{
			Name:    "testfiles/read/inp/NHHH.00001",
			Targets: []Target{{1, &cenergies, 1}},
			SubFile: "testfiles/read/inp/main0.pbs",
			JobID:   "1",
			Scale:   1},
		{
			Name:    "testfiles/read/inp/NHHH.00002",
			Targets: []Target{{1, &cenergies, 2}},
			SubFile: "testfiles/read/inp/main0.pbs",
			JobID:   "1",
			Scale:   1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%s, wanted\n%s", dumpCalc(got), dumpCalc(want))
	}
}

func TestStep(t *testing.T) {
	tmp := Conf
	defer func() {
		Conf = tmp
	}()
	Conf = Config{}
	Conf.Set(Deltas, []float64{
		0.005, 0.005, 0.005,
		0.005, 0.005, 0.005,
		0.005, 0.005, 0.005,
	})
	got := Step([]float64{
		1, 1, 1,
		1, 1, 1,
		1, 1, 1,
	}, []int{1, 2, 3}...)
	want := []float64{
		1.005, 1.005, 1.005,
		1, 1, 1,
		1, 1, 1,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestSelectNode(t *testing.T) {
	conf := Conf
	nodes := Global.Nodes
	defer func() {
		Conf = conf
		Global.Nodes = nodes
	}()
	tests := []struct {
		queue string
		nodes []string
		q     string
		node  string
	}{
		{
			queue: "r410",
			nodes: []string{
				"r410:cn113",
				"r410:cn114",
				"r410:cn115",
			},
			q:    "r410",
			node: "cn113",
		},
		{
			queue: "",
			nodes: []string{
				"r410:cn113",
				"r410:cn114",
				"r410:cn115",
			},
			q:    "r410",
			node: "cn113",
		},
		{
			queue: "workq",
			nodes: []string{
				"r410:cn113",
				"r410:cn114",
				"r410:cn115",
			},
			q:    "workq",
			node: "",
		},
	}
	for _, test := range tests {
		Conf.Set(WorkQueue, test.queue)
		Global.Nodes = test.nodes
		gn, gq := SelectNode()
		if gn != test.node {
			t.Errorf("got %v, wanted %v\n", gn, test.node)
		}
		if gq != test.q {
			t.Errorf("got %v, wanted %v\n", gq, test.q)
		}
	}
}

// This only tests a 2nd derivative
func TestDerivative(t *testing.T) {
	prog := new(Molpro)
	target := &fc2
	dir := t.TempDir()
	tmp := Conf
	Global.JobNum = 0
	defer func() {
		Conf = tmp
	}()
	tests := []struct {
		names  []string
		coords []float64
		dims   []int
		calcs  []Calc
	}{
		{
			names: []string{"O", "H", "H"},
			coords: []float64{
				0.0000000000, 0.0000000000, -0.0657441568,
				0.0000000000, 0.7574590974, 0.5217905143,
				0.0000000000, -0.7574590974, 0.5217905143,
			},
			dims: []int{1, 1, 0, 0},
			calcs: []Calc{
				{
					Name: filepath.Join(dir, "job.0000000000"),
					Targets: []Target{
						{
							Coeff: 2,
							Slice: target,
							Index: 0,
						},
						{
							Coeff: 1,
							Slice: &e2d,
							Index: 0,
						},
					},
					Scale: angbohr * angbohr / 4,
					Coords: []float64{
						2, 0, -0.0657441568,
						0, 0.7574590974, 0.5217905143,
						0, -0.7574590974, 0.5217905143,
					},
				},
				{
					Name: filepath.Join(dir, "E0"),
					Targets: []Target{
						{
							Coeff: -2,
							Slice: target,
							Index: 0,
						},
					},
					noRun: true,
					Scale: angbohr * angbohr / 4,
					Coords: []float64{
						0, 0, -0.0657441568,
						0, 0.7574590974, 0.5217905143,
						0, -0.7574590974, 0.5217905143,
					},
				},
			},
		},
	}
	for _, test := range tests {
		Conf = Config{}
		deltas := make([]float64, len(test.coords))
		for i := range test.coords {
			deltas[i] = 1.0
		}
		Conf.Set(Deltas, deltas)
		mol := symm.ReadXYZ(strings.NewReader(ZipXYZ(test.names, test.coords)))
		calcs := Derivative(prog, dir, test.names, test.coords,
			test.dims[0], test.dims[1], test.dims[2], test.dims[3],
			mol)
		if !reflect.DeepEqual(calcs, test.calcs) {
			fmt.Println("mismatch got and want:")
			fmt.Println("got:")
			byts, _ := json.MarshalIndent(&calcs, "", "\t")
			fmt.Println(string(byts))
			fmt.Println("wanted:")
			byts, _ = json.MarshalIndent(&test.calcs, "", "\t")
			fmt.Println(string(byts))
			t.Error()
		}
	}
}

func TestPush(t *testing.T) {
	dir := t.TempDir()
	var pf, count int
	count = 1
	calcs := []Calc{
		{Name: "job1"},
		{Name: "job2"},
		{Name: "job3"},
	}
	tmp2 := Conf
	defer func() {
		Conf = tmp2
	}()
	paraCount = make(map[string]int)
	Conf.Set(ChunkSize, 2)
	queue := TestQueue{
		SinglePt: pbsMaple,
		ChunkPts: ptsMaple,
	}
	got := Push(queue, dir, pf, count, calcs[0:2])
	got = append(got, Push(queue, dir, pf+1, count, calcs[2:])...)
	want := []Calc{
		{
			Name:     "job1",
			SubFile:  filepath.Join(dir, "main0.pbs"),
			ChunkNum: 0,
			JobID:    "1",
		},
		{
			Name:     "job2",
			SubFile:  filepath.Join(dir, "main0.pbs"),
			ChunkNum: 0,
			JobID:    "1",
		},
		{
			Name:     "job3",
			SubFile:  filepath.Join(dir, "main1.pbs"),
			ChunkNum: 1,
			JobID:    "1",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s,\n\nwanted %s\n", dumpCalc(got), dumpCalc(want))
	}
	var buf bytes.Buffer
	right := filepath.Join(dir, "right")
	fmt.Fprintf(&buf,
		`#!/bin/sh
#PBS -N Al2O2pts
#PBS -S /bin/bash
#PBS -j oe
#PBS -o %s
#PBS -W umask=022
#PBS -l walltime=5000:00:00
#PBS -l ncpus=1
#PBS -l mem=8gb

module load pbspro molpro

export WORKDIR=$PBS_O_WORKDIR
export TMPDIR=/tmp/$USER/$PBS_JOBID
cd $WORKDIR
mkdir -p $TMPDIR

date
hostname

molpro -t 1 job1.inp --no-xml-output
molpro -t 1 job2.inp --no-xml-output
date

rm -rf $TMPDIR
`, want[0].SubFile+".out")
	os.WriteFile(right, buf.Bytes(), 0755)
	if !compareFile(want[0].SubFile, right) {
		byt, _ := os.ReadFile(right)
		fmt.Println("wanted: ")
		fmt.Printf("%q\n", string(byt))
		byt, _ = os.ReadFile(want[0].SubFile)
		fmt.Println("got: ")
		fmt.Printf("%q\n", string(byt))
		t.Errorf("commands file mismatch\n")
	}
}

func TestBuildCartPoints(t *testing.T) {
	qsub = "qsub/qsub"
	// test to make sure we get the right number of points
	tmp := Conf
	defer func() {
		qsub = "qsub"
		Conf = tmp
	}()
	Conf.Set(Deltas, []float64{
		0.005, 0.005, 0.005,
		0.005, 0.005, 0.005,
		0.005, 0.005, 0.005,
	})
	Conf.Set(Deriv, 4)
	names := []string{"O", "H", "H"}
	coords := []float64{
		0.0000000000, 0.0000000000, -0.0657441568,
		0.0000000000, 0.7574590974, 0.5217905143,
		0.0000000000, -0.7574590974, 0.5217905143,
	}
	t2, t3, t4 := fc2, fc3, fc4
	defer func() {
		fc2, fc3, fc4 = t2, t3, t4
	}()
	fc2, fc3, fc4 = *new([]CountFloat), *new([]CountFloat), *new([]CountFloat)
	want := 5145
	mp := new(Molpro)
	dir := t.TempDir()
	queue := TestQueue{
		SinglePt: pbsMaple,
		ChunkPts: ptsMaple,
	}
	cart := ZipXYZ(names, coords)
	mol := symm.ReadXYZ(strings.NewReader(cart))
	gen := BuildCartPoints(mp, queue, dir, names, coords, mol)
	paraCount = make(map[string]int)
	got := make([]Calc, 0)
	hold, ok := gen()
	got = append(got, hold...)
	for ok {
		hold, ok = gen()
		got = append(got, hold...)
	}
	if lgot := len(got); lgot != want {
		t.Errorf("got %d, wanted %d calcs\n", lgot, want)
	}
}

// This only tests a 3rd derivative
func TestGradDerivative(t *testing.T) {
	prog := new(Molpro)
	t3 := fc3
	target := &fc3
	dir := t.TempDir()
	tmp := Conf
	glob := Global
	defer func() {
		Conf = tmp
		Global = glob
		fc3 = t3
	}()
	Global.JobNum = 0 // HashName just increases, have to reset
	tests := []struct {
		names  []string
		coords []float64
		dims   []int
		calcs  []Calc
	}{
		{
			names: []string{"O", "H", "H"},
			coords: []float64{
				0.0000000000, 0.0000000000, -0.0657441568,
				0.0000000000, 0.7574590974, 0.5217905143,
				0.0000000000, -0.7574590974, 0.5217905143,
			},
			dims: []int{1, 1, 0},
			calcs: []Calc{
				{
					Name: filepath.Join(dir, "job.0000000000"),
					Targets: []Target{
						{
							Coeff: 2,
							Slice: target,
							Index: 0,
						},
					},
					Scale: angbohr * angbohr / 4,
				},
				{
					Name: filepath.Join(dir, "E0"),
					Targets: []Target{
						{
							Coeff: -2,
							Slice: target,
							Index: 0,
						},
					},
					noRun: true,
					Scale: angbohr * angbohr / 4,
				},
			},
		},
	}
	for _, test := range tests {
		Conf = Config{}
		deltas := make([]float64, len(test.coords))
		for i := range test.coords {
			deltas[i] = 1.0
		}
		Conf.Set(Deltas, deltas)
		mol := symm.ReadXYZ(strings.NewReader(ZipXYZ(test.names, test.coords)))
		calcs := GradDerivative(prog, dir, test.names, test.coords,
			test.dims[0], test.dims[1], test.dims[2], mol)
		if !reflect.DeepEqual(calcs, test.calcs) {
			t.Errorf("got\n%v, wanted\n%v\n", calcs, test.calcs)
		}
	}
}

func TestBuildGradPoints(t *testing.T) {
	queue := TestQueue{
		SinglePt: pbsMaple,
		ChunkPts: ptsMaple,
	}
	qsub = "qsub/qsub"
	// test to make sure we get the right number of points
	tmp := Conf
	defer func() {
		qsub = "qsub"
		Conf = tmp
	}()
	Conf.Set(Deltas, []float64{
		0.005, 0.005, 0.005,
		0.005, 0.005, 0.005,
		0.005, 0.005, 0.005,
	})
	dir := t.TempDir()
	names := []string{"O", "H", "H"}
	coords := []float64{
		0.0000000000, 0.0000000000, -0.0657441568,
		0.0000000000, 0.7574590974, 0.5217905143,
		0.0000000000, -0.7574590974, 0.5217905143,
	}
	t2, t3, t4 := fc2, fc3, fc4
	defer func() {
		fc2, fc3, fc4 = t2, t3, t4
	}()
	fc2, fc3, fc4 = *new([]CountFloat), *new([]CountFloat), *new([]CountFloat)
	want := 1048
	mp := new(Molpro)
	cart := ZipXYZ(names, coords)
	mol := symm.ReadXYZ(strings.NewReader(cart))
	gen := BuildGradPoints(mp, queue, dir, names, coords, mol)
	paraCount = make(map[string]int)
	got := make([]Calc, 0)
	hold, ok := gen()
	got = append(got, hold...)
	for ok {
		hold, ok = gen()
		got = append(got, hold...)
	}
	if lgot := len(got); lgot != want {
		t.Errorf("got %d, wanted %d calcs\n", lgot, want)
	}
}

func TestE2dIndex(t *testing.T) {
	tests := []struct {
		ncoords int
		ids     []int
		want    []int
	}{
		{9, []int{1, 1}, []int{0}},
		{9, []int{1, 2}, []int{1, 18}},
		{9, []int{1, 8}, []int{7, 126}},
		{9, []int{2, 2}, []int{19}},
		{9, []int{1, -9}, []int{17, 306}},
		{9, []int{-9, -9}, []int{323}},
		{6, []int{1, 1}, []int{0}},
		{6, []int{-1, -1}, []int{78}},
	}
	for _, test := range tests {
		got := E2dIndex(test.ncoords, test.ids...)
		want := test.want
		if !reflect.DeepEqual(got, want) {
			t.Errorf("E2dIndex(%d, %v): got %v, wanted %v\n",
				test.ncoords, test.ids, got, want)
		}
	}
}

// only tests one possibility
func TestIndex(t *testing.T) {
	tests := []struct {
		ncoords int
		nosort  bool
		id      []int
		want    []int
	}{
		{
			ncoords: 9,
			id:      []int{1, 1},
			want:    []int{0},
		},
	}
	for _, test := range tests {
		got := Index(test.ncoords, test.nosort, test.id...)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("got %v, wanted %v\n", got, test.want)
		}
	}
}
