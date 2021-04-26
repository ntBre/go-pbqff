#!/bin/sh

# path to mock qsub command
base="/home/brent/Projects/go/src/github.com/ntBre/chemutils"
export PATH="$base/qsub":"$base/molpro":$PATH

cd tests/cart
pbqff -o cart.in

# cd ../grad
# pbqff -o grad.in

# cd ../sic
# pbqff -o sic.in
