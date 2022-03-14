#!/usr/bin/python

import math
import argparse
import csv
import itertools

parser = argparse.ArgumentParser(
    description="Generate a Taylor series.",
    epilog='Example: "python taylor.py 5 24" will print the product of 24 repetitions of [0,1,2,3,4].',
)
parser.add_argument(
    "digits", help="The number of digits in the array [0..digits]", type=int
)
parser.add_argument("reps", help="The number of repetitions of the set", type=int)
parser.add_argument(
    "-s",
    "--start",
    help="Only print lines with indexes greater than or equal to 'start'",
    type=int,
)
parser.add_argument(
    "-e",
    "--end",
    help="Only print lines with indexes less than or equal to 'end'",
    type=int,
)
parser.add_argument(
    "-m",
    "--modcheck",
    help="Enables a mod check of one of more subsets of digits (first digit is index 1). Expects this value to be in the format '-m d:[s-e]' where 'd' is the mod check value, 's' is the start index, and 'e' is the end index.",
    type=str,
)
parser.add_argument(
    "-q",
    "--equivalence",
    help="Enables an equivalence check of one or more subsets of digits (first digit is index 1). Expects this value to be in the format '-m q:[s-e]' where 'q' is the equivilence check value, 's' is the start index, and 'e' is the end index.",
    type=str,
)
parser.add_argument(
    "-p",
    "--parallel",
    help="Calculates the --start and --end values based on the provided node number and number of nodes. Expected format is '--p (node number):(number of nodes)'",
    type=str,
)
parser.add_argument(
    "-f",
    "--forceConstants",
    help="Write force constant values to force.txt",
    action="store_true",
)
parser.add_argument(
    "-d",
    "--displacements",
    help="Write displacement values to disp.txt",
    action="store_true",
)
parser.add_argument(
    "-i", "--indexes", help="Write row indexes to indexes.txt", action="store_true"
)
parser.add_argument(
    "-l", "--silent", help="Suppress all output to stdout", action="store_true"
)
parser.add_argument(
    "-u",
    "--unfiltered",
    help="Skip the filtering step and produce a complete cartesian product",
    action="store_true",
)
parser.add_argument(
    "-v", "--verbose", help="Produce verbose output", action="store_true"
)
parser.add_argument(
    "--debug",
    help="Sets up the job but does not run it. Prints debug info instead",
    action="store_true",
)
parser.add_argument(
    "--summary",
    help="Only print summary information (i.e. number of rows in output files)",
    action="store_true",
)
parser.add_argument(
    "--version",
    action="version",
    version='Taylor Series generation script v1.0. Latest version and full documentation available at https://github.com/russellthackston/comp-chem-util in the "misc" folder. Report any bugs or issues at the above web address.',
)
args = parser.parse_args()

modchecks = None
eqchecks = None


# This function derives a single line from the cartesian product with
# the index 'n'
def entry(n, digits):
    combo = []
    if args.verbose and not args.silent:
        print("# Building entry(" + str(n) + ")")
    exp = 1
    while n > 0:
        div = len(digits) ** exp
        digit = n % div
        n -= digit
        digit = digit / len(digits) ** (exp - 1)
        combo.insert(0, int(digit))
        exp += 1
    for i in range(0, args.reps - len(combo)):
        combo.insert(0, 0)
    return combo


def parseRanges(rangestr):
    # n:[start-end,start-end]?n:[start-end]?n:[start-end]
    # Ex.- 4:[1-2,3-4]?3:[5-6]

    results = dict()
    # Split parameter on the question mark symbol (?)
    for rstr in rangestr.split("?"):
        tmp = rstr.split(":")
        if len(tmp) != 2:
            print(len(tmp))
            print("Error: Parameter not formatted properly: " + rstr)
            print("Are you missing a colon?")
            exit(1)
        num = int(tmp[0])
        ranges = tmp[1]
        if ranges[:1] != "[" or ranges[-1:] != "]":
            print("Error: Parameter not formatted properly: " + rstr)
            print("Is your range properly bracketed (i.e. [])?")
            exit(1)
        if num in results:
            print(
                "Duplicate value '"
                + str(num)
                + "' encountered while parsing range '"
                + str(rangestr)
                + "'"
            )
            exit(1)
        results[num] = []
        for range in ranges[1:-1].split(","):
            startend = range.split("-")
            if len(startend) != 2:
                print("Error: Parameter not formatted properly: " + rstr)
                print("Do you have both a start and end value separated with a dash?")
                exit(1)
            start = int(startend[0])
            end = int(startend[1])
            results[num].append([start, end])
    if args.verbose and not args.silent:
        print("# Parsed range " + str(rangestr) + " as " + str(results))
    return results


def eqCheck(e, eqchecks):
    if eqchecks is None or len(eqchecks.keys()) == 0:
        if args.verbose and not args.silent:
            print("# No equivalence checks defined")
        return True
    if args.verbose and not args.silent:
        print("# Equivalence checking is enabled.")
    for k in eqchecks.keys():
        for r in eqchecks[k]:
            start = r[0]
            end = r[1]
            sub = e[start - 1 : end]
            failedEq = sum(e[start - 1 : end]) != k
            if failedEq:
                if args.verbose and not args.silent:
                    print(
                        "# Equivalence checking of "
                        + str(k)
                        + " for sum("
                        + str(sub)
                        + "). Failed."
                    )
                return False
            else:
                if args.verbose and not args.silent:
                    print(
                        "# Equivalence checking of "
                        + str(k)
                        + " for sum("
                        + str(sub)
                        + "). Passed."
                    )
    return True


def modCheck(e, modchecks):
    if modchecks is None or len(modchecks.keys()) == 0:
        if args.verbose and not args.silent:
            print("# No mod checks defined")
        return True
    if args.verbose and not args.silent:
        print("# Mod checking is enabled.")
    for k in modchecks.keys():
        for r in modchecks[k]:
            start = r[0]
            end = r[1]
            sub = e[start - 1 : end]
            failedMod = sum(e[start - 1 : end]) % k != 0
            if failedMod:
                if args.verbose and not args.silent:
                    print(
                        "# Mod checking %" + str(k) + " for " + str(sub) + ". Failed."
                    )
                return False
            else:
                if args.verbose and not args.silent:
                    print(f"# Mod checking %{k} for {sub}. Passed.")
    return True


def displacements(e):
    if args.verbose and not args.silent:
        print("# Converting force constants to displacements for " + str(e))
    indexes = []
    values = []
    for i, digit in enumerate(e):
        if digit != 0:
            indexes.append(i)
            values.append(digit)
    if len(values) == 0:
        if args.verbose and not args.silent:
            print("# No non-zero values found in list " + str(e))
        return [e]
    if args.verbose and not args.silent:
        print(
            "# Non-zero value(s) "
            + str(values)
            + " located in index(es) "
            + str(indexes)
        )
    prods = []
    for i, digit in enumerate(values):
        tmp = []
        for j in range(-1 * digit, digit + 1, 2):
            tmp.append(j)
        prods.append(tmp)
    if args.verbose and not args.silent:
        print("prods: ", prods)
    newrows = map(list, itertools.product(*prods))
    if args.verbose and not args.silent:
        print("# Displacement combinations: " + str(list(newrows)))
    # Build each new displacement row by copying the force constant row then
    #   overwriting the non-zero values with the calculated values
    result = []
    for row in newrows:
        # make a copy of 'e'
        r = list(e)
        for i, index in enumerate(indexes):
            r[index] = row[i]
        result.append(r)
    if args.verbose and not args.silent:
        print("# Calculated a total of " + str(len(result)) + " displacements")
        print(f"# Displacement list: {result}")
    return result


def main(startIndex, endIndex):
    # Set up summary variables
    rowsIndexes = 0
    rowsForce = 0
    rowsDisp = 0

    # Set up summary only values
    if args.summary:
        args.indexes = False
        args.displacements = False
        args.forceConstants = False

    # Build the array of digits
    digits = []
    for i in range(0, args.digits):
        digits.append(i)
    if args.verbose and not args.silent:
        print("# Digits array: " + str(digits))

    # If mod checks are enabled, build a list of checks to be performed
    if args.modcheck:
        modchecks = parseRanges(args.modcheck)
        if args.verbose and not args.silent:
            print(
                "# Parsed mod check " + str(args.modcheck) + " into " + str(modchecks)
            )

    # If equivalence checks are enabled, build a list of checks to be performed
    if args.equivalence:
        eqchecks = parseRanges(args.equivalence)
        if args.verbose and not args.silent:
            print(
                "# Parsed mod check " + str(args.equivalence) + " into " + str(eqchecks)
            )

    # Open the output file, if requested
    writeAll = (
        not args.indexes
        and not args.displacements
        and not args.forceConstants
        and not args.summary
    )
    if args.indexes or writeAll:
        fIndexes = open("indexes.txt", "w")
    if args.displacements or writeAll:
        fDisp = open("disp.txt", "w")
        writerDisp = csv.writer(fDisp)
    if args.forceConstants or writeAll:
        fForce = open("force.txt", "w")
        writerForce = csv.writer(fForce)

    rowCount = 0
    i = startIndex
    while i < endIndex:
        e = entry(i, digits)
        if args.verbose and not args.silent:
            print("# Processing index " + str(i))
        if args.unfiltered:
            rowCount += 1
            if args.summary:
                rowsIndexes += 1
                rowsForce += 1
                rowsDisp += len(displacements(e))
            if args.indexes or writeAll:
                fIndexes.write(str(i) + "\n")
            if args.displacements or writeAll:
                lst = displacements(e)
                for l in lst:
                    writerDisp.writerow(l)
            if args.forceConstants or writeAll:
                writerForce.writerow(e)
            i += 1
        else:
            # Check if numbers in array total to (args.digits - 1) or greater
            # and skip to next block
            if sum(e) >= (args.digits - 1):
                done = False
                # copy the row
                etemp = e[:]
                # zero the right-most non-zero value
                for idx in range(args.reps - 1, -1, -1):
                    if etemp[idx] > 0:
                        etemp[idx] = 0
                        if idx > 0:
                            etemp[idx - 1] = etemp[idx - 1] + 1
                        else:
                            done = True
                        break
                # translate new array into an index
                if done:
                    i = endIndex
                else:
                    i = 0
                    for idx in range(0, args.reps):
                        p = args.reps - idx - 1
                        i += etemp[idx] * (args.digits**p)
            else:
                i += 1
            if sum(e) <= (args.digits - 1):
                if args.modcheck is not None:
                    if args.verbose and not args.silent:
                        print("# Performing mod check")
                    mc = modCheck(e, modchecks)
                else:
                    mc = True
                if args.equivalence is not None:
                    if args.verbose and not args.silent:
                        print("# Performing equivalence check")
                    ec = eqCheck(e, eqchecks)
                else:
                    ec = True
                if (
                    (args.modcheck is None and not ec)
                    or (args.equivalence is None and not mc)
                    or (not ec and not mc)
                ):
                    if args.verbose and not args.silent:
                        if not mc:
                            print("# Failed mod check")
                        if not ec:
                            print("# Failed equivalence check")
                    rowCount += 1
                else:
                    if args.summary:
                        rowsIndexes += 1
                        rowsForce += 1
                        rowsDisp += len(displacements(e))
                    if args.indexes or writeAll:
                        fIndexes.write(str(i) + "\n")
                    if args.displacements or writeAll:
                        lst = displacements(e)
                        for l in lst:
                            writerDisp.writerow(l)
                    if args.forceConstants or writeAll:
                        writerForce.writerow(e)
            else:
                if args.verbose and not args.silent:
                    print(
                        "# Skipped "
                        + str(e)
                        + " due to array total greater than (args.digits - 1)"
                    )

    if args.indexes or writeAll:
        fIndexes.close()
    if args.displacements or writeAll:
        fDisp.close()
    if args.forceConstants or writeAll:
        fForce.close()

    if args.summary:
        print("Job Summary")
        print("  Indexes: " + str(rowsIndexes))
        print("  Force Constants: " + str(rowsForce))
        print("  Displacements: " + str(rowsDisp))

    if args.verbose and not args.silent:
        print("# Done creating cartesion product")


def magic(reps, node, nodes):
    return (2 * reps * node) ** int((reps**2) / (21 * math.sqrt(nodes)))


# ******* Begin main program **********

# Set up indexes
# If parallelization is enabled, calculate the start and end based on the node number
#   and number of nodes
if args.parallel:
    if args.verbose and not args.silent:
        print("Calculating parallelization values")
    p = args.parallel.split(":")
    if len(p) != 2:
        print("Error: Parameter not formatted properly: " + args.parallel)
        print("Is your start/end node list properly formatted as '(start):(end)'?")
        exit(1)
    if int(p[0]) == 1:
        # If it's the first node in the node set, then the start index is 0
        sIndex = 0
        if args.verbose and not args.silent:
            print("First node. Setting sIndex to 0")
    else:
        sIndex = magic(args.reps, int(p[0]), int(p[1]))
        if args.verbose and not args.silent:
            print("sIndex set to " + str(sIndex))
    if int(p[0]) == int(p[1]):
        # if it's the last node in the node set, then the end is the last index
        eIndex = args.digits**args.reps
        if args.verbose and not args.silent:
            print("Last node. eIndex set to " + str(eIndex))
    else:
        eIndex = magic(args.reps, int(p[0]) + 1, int(p[1]))
        if args.verbose and not args.silent:
            print("eIndex set to " + str(eIndex))

    if sIndex == eIndex:
        eIndex = eIndex + 1
else:
    if args.start:
        sIndex = args.start
    else:
        sIndex = 0
    if args.end:
        eIndex = args.end
    else:
        eIndex = args.digits**args.reps

if args.debug:
    print(str(args))
    print("sIndex=" + str(sIndex))
    print("eIndex=" + str(eIndex))
    exit(0)


main(sIndex, eIndex)

# just code up algos 5 and 4 from Thackston18
