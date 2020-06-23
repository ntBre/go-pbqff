package main

import (
	"reflect"
	"testing"
)

func TestCleanSplit(t *testing.T) {
	t.Run("trailing newline", func(t *testing.T) {
		got := CleanSplit("this is\nan\nexample\n", "\n")
		want := []string{"this is", "an", "example"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})

	t.Run("internal newline", func(t *testing.T) {
		got := CleanSplit("this is\nan\n\nexample\n", "\n")
		want := []string{"this is", "an", "example"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
}
