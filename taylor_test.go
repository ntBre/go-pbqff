package main

import (
	"bufio"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// func TestTaylor(t *testing.T) {
// 	tmp := Conf.Intder
// 	path, _ := filepath.Abs("bin/intder")
// 	Conf.Intder = path
// 	defer func() {
// 		Conf.Intder = tmp
// 	}()
// 	intder, _ := LoadIntder("tests/sic/intder.in")
// 	Taylor([]string{"H", "O", "H"}, intder)
// }

func loadForces(filename string) (ret [][]int) {
	f, _ := os.Open(filename)
	scanner := bufio.NewScanner(f)
	var fields []string
	for scanner.Scan() {
		fields = strings.Split(scanner.Text(), ",")
		tmp := make([]int, len(fields))
		for i, s := range fields {
			tmp[i], _ = strconv.Atoi(s)
		}
		ret = append(ret, tmp)
	}
	return
}

func TestNewTaylor(t *testing.T) {
	tests := []struct {
		mods [][]int
		eqs  [][]int
		want [][]int
		m    int
		n    int
	}{
		{
			m: 5, n: 3, mods: nil, eqs: nil,
			want: [][]int{
				{0, 0, 0}, {0, 0, 1}, {0, 0, 2},
				{0, 0, 3}, {0, 0, 4}, {0, 1, 0},
				{0, 1, 1}, {0, 1, 2}, {0, 1, 3},
				{0, 2, 0}, {0, 2, 1}, {0, 2, 2},
				{0, 3, 0}, {0, 3, 1}, {0, 4, 0},
				{1, 0, 0}, {1, 0, 1}, {1, 0, 2},
				{1, 0, 3}, {1, 1, 0}, {1, 1, 1},
				{1, 1, 2}, {1, 2, 0}, {1, 2, 1},
				{1, 3, 0}, {2, 0, 0}, {2, 0, 1},
				{2, 0, 2}, {2, 1, 0}, {2, 1, 1},
				{2, 2, 0}, {3, 0, 0}, {3, 0, 1},
				{3, 1, 0}, {4, 0, 0},
			},
		},
		{
			m: 5, n: 3, mods: [][]int{
				{3, 3},
				{0, 0},
				{0, 0},
			},
			eqs: [][]int{
				{3, 3},
				{0, 0},
				{0, 0},
			},
			want: [][]int{
				{0, 0, 0}, {0, 0, 2}, {0, 0, 4},
				{0, 1, 0}, {0, 1, 2}, {0, 2, 0},
				{0, 2, 2}, {0, 3, 0}, {0, 4, 0},
				{1, 0, 0}, {1, 0, 2}, {1, 1, 0},
				{1, 1, 2}, {1, 2, 0}, {1, 3, 0},
				{2, 0, 0}, {2, 0, 2}, {2, 1, 0},
				{2, 2, 0}, {3, 0, 0}, {3, 1, 0},
				{4, 0, 0},
			},
		},
		{
			m: 5, n: 9,
			mods: [][]int{
				{5, 7},
				{8, 8},
				{9, 9},
			},
			eqs: [][]int{
				{5, 7},
				{8, 8},
				{9, 9},
			},
			want: loadForces("testfiles/load/force.txt"),
		},
	}
	for _, test := range tests {
		got := newTaylor(test.m, test.n, test.mods, test.eqs)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("got %v, wanted %v\n", got, test.want)
		}
	}
}

func TestRow(t *testing.T) {
	got := Row(40, 3, 5)
	want := []int{1, 3, 0}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestNextRow(t *testing.T) {
	got := NextRow([]int{1, 0, 3}, 3, 5)
	want := 30
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestDisps(t *testing.T) {
	got := Disps(newTaylor(5, 3, nil, nil))
	want := [][]int{
		{0, 0, 0},
		{0, 0, -1},
		{0, 0, 1},
		{0, 0, -2},
		{0, 0, 2},
		{0, 0, -3},
		{0, 0, 3},
		{0, 0, -4},
		{0, 0, 4},
		{0, -1, 0},
		{0, 1, 0},
		{0, -1, -1},
		{0, -1, 1},
		{0, 1, -1},
		{0, 1, 1},
		{0, -1, -2},
		{0, -1, 2},
		{0, 1, -2},
		{0, 1, 2},
		{0, -1, -3},
		{0, -1, 3},
		{0, 1, -3},
		{0, 1, 3},
		{0, -2, 0},
		{0, 2, 0},
		{0, -2, -1},
		{0, -2, 1},
		{0, 2, -1},
		{0, 2, 1},
		{0, -2, -2},
		{0, -2, 2},
		{0, 2, -2},
		{0, 2, 2},
		{0, -3, 0},
		{0, 3, 0},
		{0, -3, -1},
		{0, -3, 1},
		{0, 3, -1},
		{0, 3, 1},
		{0, -4, 0},
		{0, 4, 0},
		{-1, 0, 0},
		{1, 0, 0},
		{-1, 0, -1},
		{-1, 0, 1},
		{1, 0, -1},
		{1, 0, 1},
		{-1, 0, -2},
		{-1, 0, 2},
		{1, 0, -2},
		{1, 0, 2},
		{-1, 0, -3},
		{-1, 0, 3},
		{1, 0, -3},
		{1, 0, 3},
		{-1, -1, 0},
		{-1, 1, 0},
		{1, -1, 0},
		{1, 1, 0},
		{-1, -1, -1},
		{-1, -1, 1},
		{-1, 1, -1},
		{-1, 1, 1},
		{1, -1, -1},
		{1, -1, 1},
		{1, 1, -1},
		{1, 1, 1},
		{-1, -1, -2},
		{-1, -1, 2},
		{-1, 1, -2},
		{-1, 1, 2},
		{1, -1, -2},
		{1, -1, 2},
		{1, 1, -2},
		{1, 1, 2},
		{-1, -2, 0},
		{-1, 2, 0},
		{1, -2, 0},
		{1, 2, 0},
		{-1, -2, -1},
		{-1, -2, 1},
		{-1, 2, -1},
		{-1, 2, 1},
		{1, -2, -1},
		{1, -2, 1},
		{1, 2, -1},
		{1, 2, 1},
		{-1, -3, 0},
		{-1, 3, 0},
		{1, -3, 0},
		{1, 3, 0},
		{-2, 0, 0},
		{2, 0, 0},
		{-2, 0, -1},
		{-2, 0, 1},
		{2, 0, -1},
		{2, 0, 1},
		{-2, 0, -2},
		{-2, 0, 2},
		{2, 0, -2},
		{2, 0, 2},
		{-2, -1, 0},
		{-2, 1, 0},
		{2, -1, 0},
		{2, 1, 0},
		{-2, -1, -1},
		{-2, -1, 1},
		{-2, 1, -1},
		{-2, 1, 1},
		{2, -1, -1},
		{2, -1, 1},
		{2, 1, -1},
		{2, 1, 1},
		{-2, -2, 0},
		{-2, 2, 0},
		{2, -2, 0},
		{2, 2, 0},
		{-3, 0, 0},
		{3, 0, 0},
		{-3, 0, -1},
		{-3, 0, 1},
		{3, 0, -1},
		{3, 0, 1},
		{-3, -1, 0},
		{-3, 1, 0},
		{3, -1, 0},
		{3, 1, 0},
		{-4, 0, 0},
		{4, 0, 0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestCartProd(t *testing.T) {
	tests := []struct {
		msg  string
		inp  [][]int
		want [][]int
	}{
		{
			msg:  "1x4",
			inp:  [][]int{{-4, -2, 0, 2, 4}},
			want: [][]int{{-4}, {-2}, {0}, {2}, {4}},
		},
		{
			msg: "len 2x2",
			inp: [][]int{
				{-1, 1},
				{-1, 1},
			},
			want: [][]int{
				{-1, -1},
				{-1, 1},
				{1, -1},
				{1, 1},
			},
		},
		{
			msg: "len 3x2",
			inp: [][]int{
				{-1, 1},
				{-1, 1},
				{-1, 1},
			},
			want: [][]int{
				{-1, -1, -1},
				{-1, -1, 1},
				{-1, 1, -1},
				{-1, 1, 1},
				{1, -1, -1},
				{1, -1, 1},
				{1, 1, -1},
				{1, 1, 1},
			},
		},
	}
	for _, test := range tests {
		got := CartProd(test.inp)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("got %v, wanted %v\n", got, test.want)
		}
	}
}

func TestModCheck(t *testing.T) {
	tests := []struct {
		row       []int
		modchecks [][]int
		want      bool
	}{
		{
			row: []int{0, 0, 0},
			modchecks: [][]int{
				{3, 3},
				{0, 0},
				{0, 0},
			},
			want: true,
		},
		{
			row: []int{0, 0, 1},
			modchecks: [][]int{
				{3, 3},
				{0, 0},
				{0, 0},
			},
			want: false,
		},
		{
			row: []int{0, 0, 2},
			modchecks: [][]int{
				{3, 3},
				{0, 0},
				{0, 0},
			},
			want: true,
		},
	}
	for _, test := range tests {
		got := ModCheck(test.row, test.modchecks)
		if got != test.want {
			t.Errorf("got %v, wanted %v\n", got, test.want)
		}
	}
}

func TestEqCheck(t *testing.T) {
	tests := []struct {
		row      []int
		eqchecks [][]int
		want     bool
	}{
		{
			row: []int{0, 0, 0},
			eqchecks: [][]int{
				{3, 3},
				{0, 0},
				{0, 0},
			},
			want: false,
		},
		{
			row: []int{0, 0, 1},
			eqchecks: [][]int{
				{3, 3},
				{0, 0},
				{0, 0},
			},
			want: false,
		},
	}
	for _, test := range tests {
		got := EqCheck(test.row, test.eqchecks)
		if got != test.want {
			t.Errorf("got %v, wanted %v\n", got, test.want)
		}
	}
}
