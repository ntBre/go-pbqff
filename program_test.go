package main

import (
	"reflect"
	"testing"
)

func TestDispToStep(t *testing.T) {
	got := DispToStep(
		[][]int{
			{0, 0, -1},
			{0, 0, 1},
			{0, 0, -2},
			{0, 0, 2},
			{0, 0, -3},
		})
	want := [][]int{
		{-3},
		{3},
		{-3, -3},
		{3, 3},
		{-3, -3, -3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}
