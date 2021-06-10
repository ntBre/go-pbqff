package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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
	t.Run("first test", func(t *testing.T) {
		got, _ := Pattern(text, 0, false)
		want := [][]int{
			[]int{2, 1, 1},
			[]int{4, 2, 1},
			[]int{1, 2, 1},
			[]int{2, 4, 1},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})

	t.Run("second test", func(t *testing.T) {
		got, _ := Pattern(text1, 0, false)
		want := [][]int{
			[]int{4, 2, 1},
			[]int{1, 2, 1},
			[]int{2, 1, 1},
			[]int{2, 4, 1},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
}

func TestSwap(t *testing.T) {
	start := [][]int{
		[]int{2, 1, 1},
		[]int{4, 2, 1},
		[]int{1, 2, 1},
		[]int{2, 4, 1},
	}
	got := Swap(start, 0, 1)
	want := [][]int{
		[]int{1, 2, 1},
		[]int{2, 4, 1},
		[]int{2, 1, 1},
		[]int{4, 2, 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestMatchPattern(t *testing.T) {
	t.Run("columns match", func(t *testing.T) {
		p1, _ := Pattern(text, 0, false)
		p2, _ := Pattern(text1, 0, false)
		_, got, _ := MatchPattern(p1, p2)
		want := []int{2, 0, 1, 3}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
	t.Run("column mismatch", func(t *testing.T) {
		p1, _ := Pattern(text, 0, false)
		p2, _ := Pattern(text2, 0, false)
		_, got, _ := MatchPattern(p1, p2)
		want := []int{0, 1, 2, 3}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
	// 	nh3Opt := `-0.000015401   -0.128410268    0.000000000
	// 	     1.771141442    0.594753632    0.000000000
	// 	    -0.885463713    0.594841026   -1.533900726
	// 	    -0.885463713    0.594841026    1.533900726
	// `
	// 	// 	nh3OptM := `0.000015401   0.128410268    -0.000000000
	// 	//      -1.771141442    -0.594753632    -0.000000000
	// 	//     0.885463713    -0.594841026   1.533900726
	// 	//     0.885463713    -0.594841026    -1.533900726
	// 	// `
	// 	nh3Want := `0.016235768449    -1.982204963067    -0.000000000000
	//      -0.107801918438     1.824619060051     0.000000000000
	//       1.452414533971     2.499614277945     1.910688404874
	//       1.452414533971     2.499614277945    -1.910688404874
	// `
	// 	t.Run("more advanced", func(t *testing.T) {
	// 		p1, _ := Pattern(nh3Opt, 0)
	// 		p2, _ := Pattern(nh3Want, 0)
	// 		Pprint(p1)
	// 		fmt.Println("---")
	// 		Pprint(p2)
	// 		_, got, _ := MatchPattern(p1, p2)
	// 		want := []int{1, 0, 2, 3}
	// 		if !reflect.DeepEqual(got, want) {
	// 			t.Errorf("got %v, wanted %v\n", got, want)
	// 		}
	// 	})
}

func TestSwapStr(t *testing.T) {
	txt := []string{
		"1 2 3",
		"4 5 6",
		"7 8 9",
	}
	swps := [][]int{
		[]int{0, 1},
		[]int{1, 2},
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
	got := i.SecondLine()
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
		},
		{
			msg: "problematic",
			cart: `N     0.000000000    0.000000000   -0.153669476
     H     0.000000000    1.508025592    1.067723399
     H     0.000000000   -1.508025592    1.067723399
`,
			intderFile: "testfiles/load/intder.ally",
			want:       []string{"H", "N", "H"},
		},
	}
	for _, test := range tests {
		fmt.Println(test.msg)
		i, _ := LoadIntder(test.intderFile)
		got := i.ConvertCart(test.cart)
		want := test.want
		if !reflect.DeepEqual(got, want) {
			t.Errorf("ConvertCart(%s): got %v, wanted %v\n", test.msg, got, want)
		}
	}

	t.Run("dummy atom in intder", func(t *testing.T) {
		cart := `C       0.000000000    0.000000000   -1.079963204
	O       0.000000000    0.000000000    1.008829581
	H       0.000000000    0.000000000   -3.144264495
	`
		i, _ := LoadIntder("testfiles/lin.intder")
		got := i.ConvertCart(cart)
		want := []string{"O", "C", "H"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
		wantDum := []Dummy{Dummy{Coords: []float64{0.000000000, 1.111111111, -1.079963204},
			Matches: []int{0, -1, 5}}} // position in intder is 5
		if !reflect.DeepEqual(i.Dummies, wantDum) {
			t.Errorf("got %v, wanted %v\n", i.Dummies, wantDum)
		}
	})
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
	base := filepath.Base(filename)
	outfile, _ := os.Create(temp + "/" + base)
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
		fmt.Println(s)
	}, true)
	if !compareFile("tmp/opt.inp", "testfiles/write/opt.inp") {
		t.Errorf("mismatch\n")
	}
}

func TestRunIntder(t *testing.T) {
	workDir("testfiles/write/intder.in", func(s string) {
		RunIntder(s[:len(s)-3])
	}, false)
}

func TestLoadIntder(t *testing.T) {
	got, _ := LoadIntder("testfiles/load/intder.full")
	data, err := ioutil.ReadFile("testfiles/right/intder.full.json")
	if err != nil {
		panic(err)
	}
	want := new(Intder)
	err = json.Unmarshal(data, want)
	if err != nil {
		panic(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
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
	t.Run("no dummy atoms", func(t *testing.T) {
		cart, _ := ReadLog("testfiles/read/al2o2.log")
		i, _ := LoadIntder("testfiles/load/intder.full")
		i.ConvertCart(cart)
		i.ReadGeom("testfiles/read/intder_geom.out")
		want := `        0.0000000000       -0.0115666469        2.4598228639
        0.0000000000       -0.0139207809        0.2726915161
        0.0000000000        0.1184234620       -2.1785371074
        0.0000000000       -1.5591967852       -2.8818447886`
		if i.Geometry != want {
			t.Errorf("got %v, wanted %v", i.Geometry, want)
		}
	})
	t.Run("dummy atoms", func(t *testing.T) {
		cart, _ := ReadLog("testfiles/read/dummy.log")
		i, _ := LoadIntder("testfiles/read/dummy.intder.in")
		i.ConvertCart(cart)
		i.ReadGeom("testfiles/dummy_geom.out")
		want := `        0.0000000000        0.0000000000        1.0109039650
        0.0000000000        0.0000000000       -1.0824085329
        0.0000000000        0.0000000000       -3.1489094311
        0.0000000000        1.1111111110       -1.0824085329
        1.1111111110        0.0000000000       -1.0824085329`
		if i.Geometry != want {
			t.Errorf("got\n%v, wanted\n%v", i.Geometry, want)
		}
	})
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
	i.Read9903("testfiles/read/fort.9903")
	got := i.Tail
	bytes, _ := ioutil.ReadFile("testfiles/right/fort.9903")
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
	i.Read9903("testfiles/read/prob.9903")
	write := "testfiles/write/intder.freqs.in"
	right := "testfiles/right/intder.freqs.in"
	i.WriteFreqs(write, order)
	if !compareFile(write, right) {
		t.Errorf("mismatch between %s and %s\n", write, right)
	}
}
