#!/bin/bash
#SBATCH --job-name={{.Name}}
#SBATCH --ntasks=1
#SBATCH --cpus-per-task=1
#SBATCH -o {{.Filename}}.out
#SBATCH --no-requeue
#SBATCH --mem={{.PBSMem}}gb

{{range $j := .Jobs}}
/home/qc/bin/molpro2020.sh 1 1 {{ $j }}
{{- end }}
