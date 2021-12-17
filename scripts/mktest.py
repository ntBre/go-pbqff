#!/usr/bin/python

import glob
import io
import json
from sys import argv


def parse(filename):
    ingeom = False
    buf = io.StringIO()
    energy = 0
    with open(filename) as f:
        for line in f:
            if "geometry={" in line:
                ingeom = True
            elif ingeom and "}" in line:
                ingeom = False
            elif ingeom:
                buf.write(line.lstrip())
            elif "energy= " in line:
                energy = float(line.split()[-1])
    return buf.getvalue(), energy


if __name__ == "__main__":
    job_glob = "job*.out"
    if argv[1] != "":
        job_glob = argv[1]
    jobs = glob.glob(job_glob)
    d = {}
    for job in jobs:
        g, e = parse(job)
        d[json.dumps(g)] = e

    print("{")
    for i, elt in enumerate(d):
        print(f'{elt}: {{\n"Energy": {d[elt]},\n"Gradient": null\n}}', end='')
        if i < len(d)-1:
            print(",")
    print("\n}")
