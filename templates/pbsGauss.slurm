#!/bin/bash
#SBATCH --job-name={{.Name}}
#SBATCH --ntasks=1
#SBATCH --cpus-per-task=1
#SBATCH -o {{.Filename}}.out
#SBATCH --no-requeue
#SBATCH --mem={{.PBSMem}}gb

/home/qc/bin/g16b01.sh {{.Filename}}
