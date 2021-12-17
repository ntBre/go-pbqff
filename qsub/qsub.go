package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	f      = flag.Bool("f", false, "placeholder for qsub foreground flag")
	molpro = "molpro/molpro"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "qsub: not enough arguments in call to qsub")
		os.Exit(253)
	}
	infile, err := os.Open(args[0])
	defer infile.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, "qsub:", err)
		os.Exit(254)
	}
	scanner := bufio.NewScanner(infile)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "molpro") &&
			!strings.Contains(line, "module") {
			// line of the form "molpro -t 1 filename"
			fields := strings.Fields(line)
			Molpro(fields[3])
		} else if strings.Contains(line, "parallel") {
			fields := strings.Fields(line)
			i := 0
			for i = range fields {
				if fields[i] == "<" {
					break
				}
			}
			cmdfile, err := os.Open(fields[i+1])
			if err != nil {
				fmt.Fprintln(os.Stderr, "qsub:", err)
				os.Exit(255)
			}
			s := bufio.NewScanner(cmdfile)
			for s.Scan() {
				fields := strings.Fields(s.Text())
				cmd := exec.Command(molpro, fields[1:]...)
				cmd.Stderr = os.Stderr
				err := cmd.Run()
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(215)
				}
			}
		}
	}
	fmt.Printf("%d.maple\n", rand.Intn(1_000_000))
}
