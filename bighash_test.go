package main

// func TestNormalize(t *testing.T) {
// 	// mol := symm.Molecule{
// 	// 	Atoms: []symm.Atom{
// 	// 		{"H", []float64{0.9372476886, 0.3147300674, 0.0000000000}},
// 	// 		{"N", []float64{-0.0000081497, -0.0679517874, 0.0000000000}},
// 	// 		{"H", []float64{-0.4685672185, 0.3147763140, -0.8117053085}},
// 	// 		{"H", []float64{-0.4685672185, 0.3147763140, 0.8117053085}},
// 	// 	},
// 	// 	Axes:   nil,
// 	// 	Planes: []symm.Plane{symm.XY},
// 	// 	Group:  symm.Cs,
// 	// }
// 	// got := Normalize(mol, []float64{
// 	// 	0.9372476886, 0.3147300674, 0.0000000000,
// 	// 	-0.0000081497, -0.0679517874, 0.0000000000,
// 	// 	// step this coord by -1
// 	// 	-1.4685672185, 0.3147763140, -0.8117053085,
// 	// 	-0.4685672185, 0.3147763140, 0.8117053085,
// 	// })
// 	want := [][]float64{
// 		{
// 			0.9372476886, 0.3147300674, 0.0000000000,
// 			-0.0000081497, -0.0679517874, 0.0000000000,
// 			-0.4685672185, 0.3147763140, -0.8117053085,
// 			// now this coord is stepped by -1
// 			-1.4685672185, 0.3147763140, 0.8117053085,
// 		},
// 		{
// 			0.9372476886, 0.3147300674, 0.0000000000,
// 			-0.0000081497, -0.0679517874, 0.0000000000,
// 			-1.4685672185, 0.3147763140, -0.8117053085,
// 			-0.4685672185, 0.3147763140, 0.8117053085,
// 		},
// 	}
// 	// it should give me back what I gave it, plus stepping the
// 	// atom on the other side by the same amount
// 	fmt.Println("got")
// 	for _, row := range got {
// 		for i := 0; i < len(row)/3; i++ {
// 			fmt.Printf("%15.10f\n", row[3*i:3*i+3])
// 		}
// 		fmt.Println("---")
// 	}
// 	fmt.Println("want")
// 	for _, row := range want {
// 		for i := 0; i < len(row)/3; i++ {
// 			fmt.Printf("%15.10f\n", row[3*i:3*i+3])
// 		}
// 		fmt.Println("---")
// 	}

// 	// if there is an atom where the reflection puts this atom,
// 	// this step is equivalent to the same step on that atom

// 	// maybe I need to break it up into one atom at a time

// 	// if part of the step affects this atom, do the symmetry
// 	// operation, check if there is an atom in the new location in
// 	// the original geometry. if there is, this step is equivalent
// 	// to stepping that atom

// 	// but how does that extend to combinations of steps?

// 	// I want a generic transformation, not implementing each
// 	// n-atom transformation separately, that's what I had before
// }

// // when I perform the symmetry operation *I* know that the result is
// // equivalent to the original by symmetry. I just need to put it in a
// // form that the computer will recognize

// // just swap the pairs of atoms affected, that works for this case at
// // least

// // so I do need something like IsSame that returns the new order of
// // the atoms, and I need to do that operation on the original atoms,
// // not the stepped ones

// // the solution for my only test case currently is to swap the rows,
// // but I don't think that is generalizable. it's also not clear how to
// // identify the rows to be swapped besides just the ones different
// // from before
