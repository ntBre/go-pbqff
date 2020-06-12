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
		got := Pattern(text)
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
		got := Pattern(text1)
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
		p1 := Pattern(text)
		p2 := Pattern(text1)
		_, got, _ := MatchPattern(p1, p2)
		want := []int{2, 0, 1, 3}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
	t.Run("column mismatch", func(t *testing.T) {
		p1 := Pattern(text)
		p2 := Pattern(text2)
		_, got, _ := MatchPattern(p1, p2)
		want := []int{0, 1, 2, 3}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
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
	p1 := Pattern(text)
	p2 := Pattern(text1)
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

}
