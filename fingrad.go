package main

// GradMake2D makes the Job slices for finite differences second
// derivative force constants with analytical gradients
func GradMake2D(i int) []ProtoCalc {
	return []ProtoCalc{
		{1, HashName(), []int{i}, []int{i}},
		{-1, HashName(), []int{-i}, []int{i}},
	}
}
