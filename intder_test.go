package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

var (
	text = `0.000000000        2.391678166        0.000000000
     -2.274263181        0.000000000        0.000000000
      2.274263181        0.000000000        0.000000000
      0.000000000       -2.391678166        0.000000000
`
	text1 = `-1.2426875991        0.0000000000        0.0000000000
          1.2426875991        0.0000000000        0.0000000000
          0.0000000000        1.3089084707        0.0000000000
          0.0000000000       -1.3089084707        0.0000000000
`
	text2 = `0.000000000        0.000000000        2.391678166
     -2.274263181        0.000000000        0.000000000
      2.274263181        0.000000000        0.000000000
      0.000000000        0.000000000       -2.391678166
`
)

func TestPattern(t *testing.T) {
	tests := []struct {
		msg    string
		want   [][]int
		inp    string
		ndummy int
		negate bool
	}{
		{
			msg: "first test",
			want: [][]int{
				{2, 1, 1},
				{4, 2, 1},
				{1, 2, 1},
				{2, 4, 1},
			},
			inp: `0.000000000        2.391678166        0.000000000
     -2.274263181        0.000000000        0.000000000
      2.274263181        0.000000000        0.000000000
      0.000000000       -2.391678166        0.000000000
`,
			ndummy: 0,
			negate: false,
		},
		{
			msg: "second test",
			want: [][]int{
				{4, 2, 1},
				{1, 2, 1},
				{2, 1, 1},
				{2, 4, 1},
			},
			inp: `-1.2426875991        0.0000000000        0.0000000000
          1.2426875991        0.0000000000        0.0000000000
          0.0000000000        1.3089084707        0.0000000000
          0.0000000000       -1.3089084707        0.0000000000
`,
			ndummy: 0,
			negate: false,
		},
		{
			msg: "not working",
			want: [][]int{
				{1, 2, 5},
				{1, 4, 4},
				{1, 3, 3},
				{1, 5, 1},
				{1, 1, 2},
			},
			inp: `      0.000000000        1.338056698       -3.132279476
      0.000000000       -0.112155251       -2.059535480
      0.000000000        0.015124412        1.148256948
      0.000000000       -2.612344930        2.549883346
      0.000000000        2.649704220        2.536471967
`,
			ndummy: 0,
			negate: false,
		},
	}
	for _, test := range tests {
		got, _ := Pattern(test.inp, test.ndummy, test.negate)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("%s: got %v, wanted %v\n",
				test.msg, got, test.want)
		}
	}
}

func TestSwap(t *testing.T) {
	start := [][]int{
		{2, 1, 1},
		{4, 2, 1},
		{1, 2, 1},
		{2, 4, 1},
	}
	got := Swap(start, 0, 1)
	want := [][]int{
		{1, 2, 1},
		{2, 4, 1},
		{2, 1, 1},
		{4, 2, 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		msg          string
		geom1, geom2 string
		ndum1, ndum2 int
		neg1, neg2   bool
		want         []int
	}{
		{
			msg:   "columns match",
			geom1: text,
			ndum1: 0,
			neg1:  false,
			geom2: text1,
			ndum2: 0,
			neg2:  false,
			want:  []int{2, 0, 1, 3},
		},
		{
			msg:   "column mismatch",
			geom1: text,
			ndum1: 0,
			neg1:  false,
			geom2: text2,
			ndum2: 0,
			neg2:  false,
			want:  []int{0, 1, 2, 3},
		},
	}
	for _, test := range tests {
		p1, _ := Pattern(test.geom1, test.ndum1, test.neg1)
		p2, _ := Pattern(test.geom2, test.ndum2, test.neg2)
		_, got, _ := MatchPattern(p1, p2)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("%s: got %v, wanted %v\n",
				test.msg, got, test.want)
		}
	}
}

func TestSwapStr(t *testing.T) {
	txt := []string{
		"1 2 3",
		"4 5 6",
		"7 8 9",
	}
	swps := [][]int{
		{0, 1},
		{1, 2},
	}
	got := SwapStr(swps, txt, "%s %s %s")
	want := []string{
		"2 3 1",
		"5 6 4",
		"8 9 7",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestApplyPattern(t *testing.T) {
	p1, _ := Pattern(text, 0, false)
	p2, _ := Pattern(text1, 0, false)
	_, tr, _ := MatchPattern(p1, p2)
	s := []string{"Al", "Al", "O", "O"}
	got := ApplyPattern(tr, s)
	want := []string{"O", "Al", "Al", "O"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestSecondLine(t *testing.T) {
	i, _ := LoadIntder("testfiles/load/intder.full")
	got := i.SecondLine(false)
	want := `    4    7    6    4    0    3    2    0    0    1    3    0    0    0    0    0`
	if got != want {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestConvertCart(t *testing.T) {
	tests := []struct {
		msg        string
		cart       string
		intderFile string
		want       []string
		wantDum    []Dummy
		match      bool
	}{
		{
			msg: "Columns in right order",
			cart: `Al        -1.2426875991        0.0000000000        0.0000000000
 Al         1.2426875991        0.0000000000        0.0000000000
 O          0.0000000000        1.3089084707        0.0000000000
 O          0.0000000000       -1.3089084707        0.0000000000
`,
			intderFile: "testfiles/load/intder.full",
			want:       []string{"O", "Al", "Al", "O"},
			wantDum:    []Dummy{},
		},
		{
			msg: "columns swapped order",
			cart: `Al        -1.2426875991        0.0000000000        0.0000000000
	 Al         1.2426875991        0.0000000000        0.0000000000
	 O          0.0000000000        0.0000000000        1.3089084707
	 O          0.0000000000        0.0000000000       -1.3089084707
	`,
			intderFile: "testfiles/load/intder.full",
			want:       []string{"O", "Al", "Al", "O"},
			wantDum:    []Dummy{},
		},
		{
			msg: "signs opposite",
			cart: `O          0.000000000    0.000000000   -1.138365613
     C          0.000000000    0.000000000    1.141275963
     H          0.000000000    1.772444039    2.234905449
     H          0.000000000   -1.772444039    2.234905449
    `,
			intderFile: "testfiles/load/intder.signs",
			want:       []string{"O", "C", "H", "H"},
			wantDum:    []Dummy{},
		},
		{
			msg: "nomatch",
			cart: `     H          0.000000000   -2.393101439   -2.390871871
     S          0.000000000    0.086966784   -1.813648935
     S          0.000000000   -0.011729646    1.888815977
`,
			intderFile: "testfiles/load/ally.in",
			want:       []string{"H", "S", "S"},
			wantDum:    []Dummy{},
			match:      true,
		},
		{
			msg: "dummy atom in intder",
			cart: `C       0.000000000    0.000000000   -1.079963204
	O       0.000000000    0.000000000    1.008829581
	H       0.000000000    0.000000000   -3.144264495
	`,
			intderFile: "testfiles/lin.intder",
			want:       []string{"O", "C", "H"},
			wantDum: []Dummy{
				{
					Coords: []float64{
						0.000000000, 1.111111111, -1.079963204,
					},
					Matches: []int{0, -1, 5},
				},
			},
		},
	}
	for _, test := range tests {
		tmp := *nomatch
		*nomatch = test.match
		i, _ := LoadIntder(test.intderFile)
		got := i.ConvertCart(test.cart)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("ConvertCart(%s): got %v, wanted %v\n",
				test.msg, got, test.want)
		}
		if !reflect.DeepEqual(i.Dummies, test.wantDum) {
			t.Errorf("%s: got %v, wanted %v\n",
				test.msg, i.Dummies, test.wantDum)
		}
		*nomatch = tmp
	}
}

func TestAddDummy(t *testing.T) {
	i, _ := LoadIntder("testfiles/lin.intder")
	i.Dummies[0].Coords[2] = 3.4 // check that matching works
	cart := `C       0.000000000    0.000000000   -1.079963204
O       0.000000000    0.000000000    1.008829581
H       0.000000000    0.000000000   -3.144264495
`
	i.ConvertCart(cart) // add dummy happens in ConvertCart
	got := i.Geometry
	want := `      0.000000000        0.000000000        1.008829581
      0.000000000        0.000000000       -1.079963204
      0.000000000        0.000000000       -3.144264495
      0.000000000        1.111111111       -1.079963204`
	if got != want {
		t.Errorf("got\n%q, wanted\n%q\n", got, want)
	}
}

func TestNewIntder(t *testing.T) {
	cart, _ := ReadLog("testfiles/read/al2o2.log")
	got, _ := LoadIntder("testfiles/load/intder.full")
	got.ConvertCart(cart)
	want := &Intder{Geometry: `      0.000000000        2.473478532        0.000000000
     -2.348339221        0.000000000        0.000000000
      2.348339221        0.000000000        0.000000000
      0.000000000       -2.473478532        0.000000000`}
	if got.Geometry != want.Geometry {
		t.Errorf("got\n%v\n, wanted\n%v\n", got.Geometry, want.Geometry)
	}
}

func TestWritePtsIntder(t *testing.T) {
	write := "testfiles/write/intder.in"
	right := "testfiles/right/intder.in"
	cart, _ := ReadLog("testfiles/read/al2o2.log")
	i, _ := LoadIntder("testfiles/load/intder.full")
	i.ConvertCart(cart)
	i.WritePts(write)
	if !compareFile(write, right) {
		t.Errorf("mismatch between %s and %s\n", write, right)
	}
}

func workDir(filename string, f func(string), test bool) {
	temp := "tmp"
	if _, err := os.Stat(temp); os.IsExist(err) && !test {
		panic("tmp already exists")
	}
	os.Mkdir(temp, 0755)
	current, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	infile, _ := os.Open(filename)
	defer infile.Close()
	base := filepath.Base(filename)
	outfile, _ := os.Create(temp + "/" + base)
	defer outfile.Close()
	io.Copy(outfile, infile)
	os.Chdir(temp)
	f(base)
	os.Chdir(current)
	if !test {
		os.RemoveAll(temp)
	}
}

func TestWorkDir(t *testing.T) {
	workDir("testfiles/write/opt.inp", func(s string) {
	}, true)
	if !compareFile("tmp/opt.inp", "testfiles/write/opt.inp") {
		t.Errorf("mismatch\n")
	}
}

func TestRunIntder(t *testing.T) {
	temp := Conf.IntderCmd
	defer func() {
		Conf.IntderCmd =  temp
	}()
	Conf.IntderCmd =  "../bin/intder"
	workDir("testfiles/write/intder.in", func(s string) {
		RunIntder(s[:len(s)-3])
	}, false)
}

func diffIntder(a, b *Intder) string {
	switch {
	case a.Name != b.Name:
		fmt.Printf("got %v, wanted %v", a.Name, b.Name)
		return "name mismatch"
	case a.Head != b.Head:
		fmt.Printf("got %v, wanted %v", a.Head, b.Head)
		return "head mismatch"
	case a.Geometry != b.Geometry:
		return "geometry mismatch"
	case a.Tail != b.Tail:
		return "tail mismatch"
	case !reflect.DeepEqual(a.Pattern, b.Pattern):
		return "pattern mismatch"
	case !reflect.DeepEqual(a.Dummies, b.Dummies):
		return "dummies mismatch"
	default:
		panic("looks the same to me")
	}
}

func TestLoadIntder(t *testing.T) {
	tests := []struct {
		infile  string
		outfile string
	}{
		{
			infile:  "testfiles/load/intder.full",
			outfile: "testfiles/right/intder.full.json",
		},
		{
			infile:  "testfiles/load/intder.nosic",
			outfile: "testfiles/right/intder.nosic.json",
		},
	}
	for _, test := range tests {
		got, _ := LoadIntder(test.infile)
		data, err := os.ReadFile(test.outfile)
		if err != nil {
			panic(err)
		}
		want := new(Intder)
		err = json.Unmarshal(data, want)
		if err != nil {
			panic(err)
		}
		if !reflect.DeepEqual(got, want) {
			want.WritePts("/tmp/want.intder")
			got.WritePts("/tmp/got.intder")
			t.Errorf("%s: (diff %q %q)\n",
				diffIntder(got, want),
				"/tmp/want.intder", "/tmp/got.intder")
		}
	}
}

func TestWriteIntderGeom(t *testing.T) {
	cart, _ := ReadLog("testfiles/read/al2o2.log")
	i, _ := LoadIntder("testfiles/load/intder.full")
	i.ConvertCart(cart)
	longLine, _ := GetLongLine("testfiles/read/anpass1.out")
	write := "testfiles/write/intder_geom.in"
	right := "testfiles/right/intder_geom.in"
	i.WriteGeom(write, longLine)
	if !compareFile(right, write) {
		t.Errorf("mismatch between %s and %s\n", right, write)
	}
}

func TestReadGeom(t *testing.T) {
	tests := []struct {
		msg     string
		logfile string
		intder  string
		geom    string
		want    string
	}{
		{
			msg:     "no dummy atoms",
			logfile: "testfiles/read/al2o2.log",
			intder:  "testfiles/load/intder.full",
			geom:    "testfiles/read/intder_geom.out",
			want: `        0.0000000000       -0.0115666469        2.4598228639
        0.0000000000       -0.0139207809        0.2726915161
        0.0000000000        0.1184234620       -2.1785371074
        0.0000000000       -1.5591967852       -2.8818447886`,
		},
		{
			msg:     "dummy atoms",
			logfile: "testfiles/read/dummy.log",
			intder:  "testfiles/read/dummy.intder.in",
			geom:    "testfiles/dummy_geom.out",
			want: `        0.0000000000        0.0000000000        1.0109039650
        0.0000000000        0.0000000000       -1.0824085329
        0.0000000000        0.0000000000       -3.1489094311
        0.0000000000        1.1111111110       -1.0824085329
        1.1111111110        0.0000000000       -1.0824085329`,
		},
	}
	for _, test := range tests {
		cart, _ := ReadLog(test.logfile)
		i, _ := LoadIntder(test.intder)
		i.ConvertCart(cart)
		i.ReadGeom(test.geom)
		if i.Geometry != test.want {
			t.Errorf("%s: got %v, wanted %v",
				test.msg, i.Geometry, test.want)
		}
	}
}

func TestReadIntderOut(t *testing.T) {
	cart, _ := ReadLog("testfiles/read/al2o2.log")
	i, _ := LoadIntder("testfiles/load/intder.full")
	i.ConvertCart(cart)
	i.ReadGeom("testfiles/read/intder_geom.out")
	got := i.ReadOut("testfiles/read/fintder.out")
	f := []float64{437.8, 496.8, 1086.4, 1267.6, 2337.7, 3811.4}
	sort.Sort(sort.Reverse(sort.Float64Slice(f)))
	want := f
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestRead9903(t *testing.T) {
	cart, _ := ReadLog("testfiles/read/al2o2.log")
	i, _ := LoadIntder("testfiles/load/intder.full")
	i.ConvertCart(cart)
	i.ReadGeom("testfiles/read/intder_geom.out")
	i.Read9903("testfiles/read/fort.9903", false)
	got := i.Tail
	bytes, _ := os.ReadFile("testfiles/right/fort.9903")
	want := string(bytes)
	if got != want {
		t.Errorf("got\n%v, wanted\n%v\n", got, want)
	}
}

func TestWriteIntderFreqs(t *testing.T) {
	cart, _ := ReadLog("testfiles/read/al2o2.log")
	i, _ := LoadIntder("testfiles/load/intder.full")
	order := i.ConvertCart(cart)
	i.ReadGeom("testfiles/read/intder_geom.out")
	i.Read9903("testfiles/read/prob.9903", false)
	write := "testfiles/write/intder.freqs.in"
	right := "testfiles/right/intder.freqs.in"
	i.WriteFreqs(write, order, false)
	if !compareFile(write, right) {
		t.Errorf("mismatch between %s and %s\n", write, right)
	}
}
