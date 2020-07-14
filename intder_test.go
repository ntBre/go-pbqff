package main

import (
	"reflect"
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
		got, _ := Pattern(text, 0)
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
		got, _ := Pattern(text1, 0)
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
		p1, _ := Pattern(text, 0)
		p2, _ := Pattern(text1, 0)
		_, got, _ := MatchPattern(p1, p2)
		want := []int{2, 0, 1, 3}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
	t.Run("column mismatch", func(t *testing.T) {
		p1, _ := Pattern(text, 0)
		p2, _ := Pattern(text2, 0)
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
	p1, _ := Pattern(text, 0)
	p2, _ := Pattern(text1, 0)
	_, tr, _ := MatchPattern(p1, p2)
	s := []string{"Al", "Al", "O", "O"}
	got := ApplyPattern(tr, s)
	want := []string{"O", "Al", "Al", "O"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestSecondLine(t *testing.T) {
	i := LoadIntder("testfiles/intder.full")
	i.SecondLine()
}

func TestConvertCart(t *testing.T) {
	t.Run("columns in the right order", func(t *testing.T) {
		cart := `Al        -1.2426875991        0.0000000000        0.0000000000
 Al         1.2426875991        0.0000000000        0.0000000000
 O          0.0000000000        1.3089084707        0.0000000000
 O          0.0000000000       -1.3089084707        0.0000000000
`
		i := LoadIntder("testfiles/intder.full")
		got := i.ConvertCart(cart)
		want := []string{"O", "Al", "Al", "O"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
	t.Run("columns swapped order", func(t *testing.T) {
		cart := `Al        -1.2426875991        0.0000000000        0.0000000000
 Al         1.2426875991        0.0000000000        0.0000000000
 O          0.0000000000        0.0000000000        1.3089084707
 O          0.0000000000        0.0000000000       -1.3089084707
`
		i := LoadIntder("testfiles/intder.full")
		got := i.ConvertCart(cart)
		want := []string{"O", "Al", "Al", "O"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
	t.Run("dummy atom in intder", func(t *testing.T) {
		cart := `C       0.000000000    0.000000000   -1.079963204
O       0.000000000    0.000000000    1.008829581
H       0.000000000    0.000000000   -3.144264495
`
		i := LoadIntder("testfiles/lin.intder")
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
	i := LoadIntder("testfiles/lin.intder")
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
