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

func TestPop(t *testing.T) {
	got := Pop([]int{1, 2, 3}, 1)
	want := []int{1, 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestCompare(t *testing.T) {
	t.Run("are the same", func(t *testing.T) {
		x := []int{1, 2, 3}
		y := []int{3, 2, 1}
		got := Compare(x, y)
		want := true
		if got != want {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
	t.Run("not the same", func(t *testing.T) {
		x := []int{1, 2, 3}
		y := []int{4, 2, 1}
		got := Compare(x, y)
		want := false
		if got != want {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
	t.Run("are the same (7)", func(t *testing.T) {
		x := []int{1, 2, 3, 4, 5, 6, 7}
		y := []int{7, 6, 5, 4, 3, 2, 1}
		got := Compare(x, y)
		want := true
		if got != want {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
	t.Run("not the same (7)", func(t *testing.T) {
		x := []int{1, 2, 3, 4, 5, 6, 7}
		y := []int{4, 2, 1, 3, 2, 4, 5}
		got := Compare(x, y)
		want := false
		if got != want {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
}

func TestTranspose(t *testing.T) {
	got := Transpose([][]int{
		[]int{1, 2, 3},
		[]int{4, 5, 6},
		[]int{4, 5, 6},
		[]int{4, 5, 6},
	})
	want := [][]int{
		[]int{1, 4, 4, 4},
		[]int{2, 5, 5, 5},
		[]int{3, 6, 6, 6},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestMatchPattern(t *testing.T) {
	t.Run("columns match", func(t *testing.T) {
		p1 := Pattern(text)
		p2 := Pattern(text1)
		got, _ := MatchPattern(p1, p2)
		want := []int{2, 0, 1, 3}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
}

func TestMatchCols(t *testing.T) {
	p1 := Pattern(text)
	p2 := Pattern(text2)
	got := MatchCols(p1, p2)
	want := []int{0, 2, 1}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestMoveCols(t *testing.T) {
	tr := []string{"0 1 2", "0 1 2"}
	got := MoveCols([]int{0, 2, 1}, tr)
	want := []string{
		"                0                  2                  1",
		"                0                  2                  1",
	}
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

func TestSecondLine(t *testing.T) {
	i := LoadIntder("testfiles/intder.full")
	i.SecondLine()
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
