#!/usr/bin/perl

use warnings;
use strict;

my $fname = "version.go";
open(my $fh, ">", "$fname") or die "unable to open '$fname': $!";

chomp(my $commit = `git rev-parse HEAD`);
my $compiled = localtime;

print $fh "package main

const (
      VERSION = \"${commit}\"
      COMP_TIME = \"${compiled}\"
)

"
