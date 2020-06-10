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

func TestConvertCart(t *testing.T) {
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
}

func TestMatchPattern(t *testing.T) {
	p1 := Pattern(text)
	p2 := Pattern(text1)
	got, _ := MatchPattern(p1, p2)
	want := []int{2, 0, 1, 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestApplyPattern(t *testing.T) {
	p1 := Pattern(text)
	p2 := Pattern(text1)
	tr, _ := MatchPattern(p1, p2)
	s := []string{"Al", "Al", "O", "O"}
	got := ApplyPattern(tr, s)
	want := []string{"O", "Al", "Al", "O"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}
