#!/usr/bin/perl

use warnings;
use strict;
use POSIX "strftime";

my $fname = "version.go";
open(my $fh, ">", "$fname") or die "unable to open '$fname': $!";

chomp(my $commit = `git rev-parse HEAD`);
my $compiled = strftime "%a %b %e, %Y at %H:%M:%S", localtime;

print $fh "package main

const (
      VERSION = \"${commit}\"
      COMP_TIME = \"${compiled}\"
)

"
