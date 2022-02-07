#!/bin/bash
#SBATCH --job-name={{.Name}}
#SBATCH --ntasks=1
#SBATCH --cpus-per-task=1
#SBATCH -o {{.Filename}}.out
#SBATCH --no-requeue
#SBATCH --mem={{.PBSMem}}gb

# formchk at /qc/Bg16_b01_x86_64_avx2/g16/formchk

export SCRDRV=/scratch
export QCTOP=/home/qc
export GAUSS_TMPDIR="$SCRDRV/$USER"

## Determine architecture
if ! grep -c avx2 /proc/cpuinfo &> /dev/null; then
    echo "unsupported CPU architecture"
    exit 1
fi

vers="Bg16_b01_x86_64_avx2"
export g16root=$QCTOP/$vers

# converted from $g16root/g16/bsd/g16.login
gr="$g16root"
export GAUSS_EXEDIR="$gr/g16/bsd:$gr/g16"
export GAUSS_LEXEDIR="$gr/g16/linda-exe"
export GAUSS_ARCHDIR="$gr/g16/arch"
export GAUSS_BSDDIR="$gr/g16/bsd"
export _DSM_BARRIER=SHM
export _RLD_ARGS="-log /dev/null"
export GAUSS_LFLAGS2="--LindaOptions -s 10000000"
export G16BASIS="$gr/g16/basis"
gman="$gr/g16/bsd"
alias sl="$gr/g16/tests/searchlog.csh"

function run {
    input=$1
    basenm=${input%.*}
    outfile=$(basename $basenm.out)
    job_id=$(date +%F_%H_%M_%S)
    echo "running g16 on $(hostname)"
    export job_dir="$(basename $basenm)"_"$job_id"
    echo "job_dir: $job_dir"
    # remove path from chkfile
    sed -i 's/\(%chk=\).*\/\(.*\)/\1\2/' $input
    # find checkpoint file in input
    if grep -c "%chk=" $1 &> /dev/null ; then
	export CHKFILE=$(fgrep --max-count=1 "%chk=" $1 | cut -f 2 -d"=")
    else
	export CHKFILE=NO
    fi
    full_path=$(realpath $input)
    WORK=$(dirname $full_path)
    input=$(basename $input)
    export GAUSS_SCRDIR=$GAUSS_TMPDIR/$job_dir
    echo "$g16root"
    echo "Scratch dir = $GAUSS_SCRDIR"
    echo "Working dir = $WORK"
    echo "Input  file = $input"
    echo "Output file = $outfile"
    if [ $CHKFILE == "NO" ]; then
	echo "No checkpoint file defined"
    elif [ $CHKFILE == '' ]; then
	echo "ERROR: CHKFILE undefined!"
	echo "Ask sys admin to check the g16 script."
	exit
    else
	echo "Chkpnt file = $CHKFILE"
	echo "making scrdir: $GAUSS_SCRDIR on $(hostname)"
	mkdir -p $GAUSS_SCRDIR
	echo "$WORK" > $GAUSS_SCRDIR/workdir.txt
    fi
    export GAUSS_ARCNAME=$WORK/$input
    echo "arcname: $GAUSS_ARCNAME"

    /bin/cp $WORK/$input $GAUSS_SCRDIR
    if [ $CHKFILE != "NO" ] && [ $CHKFILE != "" ]; then
	/bin/mv -f $WORK/$CHKFILE $GAUSS_SCRDIR  &> /dev/null
    fi
    cd $GAUSS_SCRDIR
    echo "running $g16root/g16/g16 < $input &> $outfile in $(pwd -P)"
    TIME=$(time $g16root/g16/g16 < $input &> $outfile)

    pwd
    echo " "				>> $outfile
    echo " "				>> $outfile
    echo "-----JOB STATISTICS FROM G09 SCRIPT----"	>> $outfile
    echo "--Machine Info--"			>> $outfile
    uname -n -s -r -p -m			>> $outfile
    echo " "				>> $outfile
    echo "--Timing  Info--"			>> $outfile
    echo "    $TIME"			>> $outfile
    echo " "				>> $outfile
    echo "--FileSys Info--"			>> $outfile
    df -h $SCRDRV				>> $outfile
    df -h /home/				>> $outfile
    ls -l $GAUSS_SCRDIR			>> $outfile
    echo "running $g16root/g16/formchk $CHKFILE $WORK/$(basename $basenm).fchk"
    $g16root/g16/formchk $CHKFILE $WORK/$(basename $basenm).fchk
    /bin/mv $CHKFILE $WORK/. &> /dev/null
    /bin/mv $outfile $WORK/. &> /dev/null

    cd $WORK

    extern_flag=$(echo $WORK | grep -c ExtFiles)
    if [ -n $extern_flag ] ; then
	/bin/mv $CHKFILE $WORK/$basenm.chk >& /dev/null
    fi

    rm -rf $GAUSS_SCRDIR
}
run {{.Filename}}
