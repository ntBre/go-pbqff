#!/bin/bash

{
	echo "package main"

	echo -n "var $1 = map[string]Calc"

	sed -e 's/"\(Energy\|Gradient\)"/\1/g' -e 's/null/nil,/g' -e '/}$/d' -e 's/\]/},/g' -e 's/\[/[]float64{/g' -e 's/[0-9]$/&,/g' "$1.json"

	echo -e '},\n}\n'
} > "$1.go"
