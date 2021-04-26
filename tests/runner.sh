#!/bin/sh

export PATH=/home/brent/Projects/go/src/github.com/ntBre/pbqff/tests:$PATH

cd cart
pbqff -o cart.in

cd ../grad
pbqff -o grad.in

cd ../sic
pbqff -o sic.in
