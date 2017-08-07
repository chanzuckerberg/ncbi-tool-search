package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type context struct {
	searchDirA       string
	searchDirB       string
	prefixCache      map[string]prefixResult
	outFile          *os.File
	notFoundPrefixes map[string]int
	curPrefix        string
}

// Example of a caller function for matching sequences from a big file to
// smaller files found in the search directories.
func matchSequencesCaller() error {
	home := getUserHome()
	var err error

	// Setup
	input := home + "/sequence_lists/blast/db/FASTA/nr.gz.trimmed.sorted" +
		".reduced.txt"
	output := home + "/sequence_lists/blast/db/FASTA/nr_run_1.txt"
	ctx := context{}
	ctx.outFile, err = os.Create(output)
	if err != nil {
		return handle("Error in creating outfile", err)
	}
	ctx.searchDirA = home + "/sequence_lists/genbank_reduced"
	ctx.searchDirB = home + "/sequence_lists/refseq_trimmed"
	ctx.prefixCache = make(map[string]prefixResult)
	ctx.notFoundPrefixes = make(map[string]int)
	if err = matchSequences(&ctx, input); err != nil {
		return handle("Error in running match sequence routine", err)
	}

	// Prefixes not found and the counts of missing sequences (point values)
	fmt.Println("NOT FOUND COUNTS:")
	notFoundTotal := 0
	for k, v := range ctx.notFoundPrefixes {
		notFoundTotal += v
		c := strconv.Itoa(v)
		fmt.Println(k + ": " + c)
	}
	// Total number of sequences that weren't matched
	c := strconv.Itoa(notFoundTotal)
	fmt.Println("Not found total: " + c)
	return err
}

// matchSequences reads in accession numbers and ranges from an input file
// and matches the point values or ranges to the same accession numbers in
// files in a search directory.
func matchSequences(ctx *context, input string) error {
	file, err := os.Open(input)
	if err != nil {
		return handle("Error in opening input file.", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	// Print header
	str := fmt.Sprintf("%-15s | %13s | %s", "Target", "Found in range", "In file")
	writeLine(str, ctx.outFile)
	// Go line-by-line
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, ": ") || !strings.Contains(line, "_") {
			continue
		}
		parts := strings.Split(line, ": ")
		prefixToFind := parts[0]
		if prefixToFind == "" {
			continue
		}
		valToFind := parts[1]
		if !strings.Contains(valToFind, "-") {
			// Dealing with a point value
			findSingleValue(ctx, prefixToFind, valToFind)
		} else {
			// Dealing with a range
			findRange(ctx, prefixToFind, valToFind)
		}
	}
	return err
}

// Matches a single accession number (prefix and number) to files in the
// search directory.
func findSingleValue(ctx *context, prefix string, toFind string) error {
	num, err := strconv.Atoi(toFind)
	if err != nil {
		return handle("Error in converting to int.", err)
	}
	res, err := accessionSearch(ctx, prefix, num)
	if res != "" {
		out := fmt.Sprintf("%s%-13d | %s", prefix, num, res)
		writeLine(out, ctx.outFile)
	} else {
		out := fmt.Sprintf("%s%d not found.", prefix, num)
		writeLine(out, ctx.outFile)
		ctx.notFoundPrefixes[prefix] += 1 // Update not found counts
	}
	return err
}

// Matches an accession number range (e.g. XM_: 100-150) to files in the
// search directory.
func findRange(ctx *context, prefix string, toFind string) error {
	p := strings.Split(toFind, "-")
	startNum, startRes, err := rangePiece(ctx, prefix, p[0])
	if err != nil {
		return handle("Error in finding results for range start.", err)
	}
	endNum, endRes, err := rangePiece(ctx, prefix, p[1])
	if err != nil {
		return handle("Error in finding results for range end.", err)
	}

	if strings.Contains(startRes, "-") && startRes == endRes {
		// Result was a range, and the start/end numbers matched to the same range.
		// This means that all the intermediate range values must also be included
		// in the result.
		out := fmt.Sprintf("%s%-13s | %s", prefix, toFind, startRes)
		writeLine(out, ctx.outFile)
	} else {
		// Otherwise just go through the range sequentially and check each point
		// value.
		for i := startNum; i <= endNum; i++ {
			if err = findSingleValue(ctx, prefix, strconv.Itoa(i)); err != nil {
				return handle("Error in searching for point value.", err)
			}
		}
	}
	return err
}

// Writes a line to stdout and the results file.
func writeLine(input string, outFile *os.File) error {
	fmt.Println(input)
	if _, err := outFile.WriteString(input + "\n"); err != nil {
		return handle("Error in writing line.", err)
	}
	return nil
}

// A prefixResult represents the matches from searching for a prefix.
// - accessionNums is a list of the number values/ranges found with the same
// prefix.
// - valueToFile is a mapping of values/ranges to the file name in which the
// match was found.
type prefixResult struct {
	accessionNums []string
	valueToFile   map[string]string
}

// Gets the results of a search for a prefix to all the matching accession
// numbers in the search directory.
func prefixToResults(ctx *context, prefix string) (prefixResult, error) {
	// Setup
	var err error
	accessionNums := []string{}
	valueToFile := make(map[string]string)

	res, present := ctx.prefixCache[prefix]
	if present {
		return res, err
	}

	// Get results from disk by calling sift.
	dest := ctx.searchDirA
	if strings.Contains(prefix, "_") {
		// Underscore is only for the Refseq files
		dest = ctx.searchDirB
	}
	template := "sift '%s' '%s' -w --binary-skip | sort -k2 -n"
	cmd := fmt.Sprintf(template, prefix, dest)
	stdout, _, err := commandVerboseOnErr(cmd)
	if err != nil {
		return res, handle("Error in calling search utility", err)
	}

	// Process output
	lines := strings.Split(stdout, "\n")
	if len(lines) == 0 { // No results. Return with empty values.
		return res, err
	}

	// Make mapping of accession num/ranges to filename.
	// Make an ascending array of the accession num/ranges.
	for _, line := range lines {
		if strings.Contains(line, ": ") {
			pieces := strings.Split(line, ": ")
			key := pieces[1]
			accessionNums = append(accessionNums, key)
			// Format the file names:lines
			snip := pieces[0][len(ctx.searchDirA):]
			snip = snip[:len(snip)-len(prefix)-1]
			valueToFile[key] = snip
		}
	}

	// Add to cache
	res = prefixResult{accessionNums, valueToFile}
	ctx.prefixCache[prefix] = res

	return res, err
}

// accessionSearch matches a single prefix and target num to matches in the
// search directory.
func accessionSearch(ctx *context, prefix string, targetNum int) (string,
	error) {
	// Setup
	var err error
	accessionNums := []string{}
	valueToFile := make(map[string]string)

	// Get prefix to file search results
	if prefix != ctx.curPrefix {
		// Clear the cache when the prefix changes due to memory issues.
		ctx.curPrefix = prefix
		ctx.prefixCache = make(map[string]prefixResult)
	}
	prefixRes, err := prefixToResults(ctx, prefix)
	if err != nil {
		return "", handle("Error in getting file results for the prefix", err)
	}
	accessionNums = prefixRes.accessionNums
	valueToFile = prefixRes.valueToFile

	// Call the binary search of the results
	res, err := arraySearch(accessionNums, targetNum)
	if err != nil {
		return "", handle("Error in searching array", err)
	}

	// Format results
	if res > 0 && res < len(accessionNums) {
		matched := accessionNums[res]
		resFile := valueToFile[matched]
		resFile = resFile[:len(resFile)-4]
		out := fmt.Sprintf("%-13s | %s", accessionNums[res], resFile)
		return out, err
	}
	return "", err
}

// Modified binary search on the array where toFind is a point value and
// toSearch contains either point values or ranges.
func arraySearch(toSearch []string, toFind int) (int, error) {
	n := len(toSearch)
	low, high := 0, n-1
	for low <= high {
		mid := low + (high-low)/2 // Midpoint
		lookAt := toSearch[mid]

		if !strings.Contains(lookAt, "-") {
			// Point val in array. Standard binary search iterations.
			lookAtVal, err := strconv.Atoi(lookAt)
			if err != nil {
				return 0, handle("Problem converting number", err)
			}
			if lookAtVal > toFind {
				high = mid - 1
			} else if lookAtVal < toFind {
				low = mid + 1
			} else { // Found
				return mid, err
			}
		} else {
			// Range val in array
			p := strings.Split(lookAt, "-")
			bottom, err := strconv.Atoi(p[0]) // Lower bound
			if err != nil {
				return 0, handle("Problem converting number", err)
			}
			top, err := strconv.Atoi(p[1]) // Upper bound
			if err != nil {
				return 0, handle("Problem converting number", err)
			}
			if bottom <= toFind && toFind <= top {
				return mid, err // Found in current range
			} else if toFind < bottom {
				high = mid - 1
			} else if top < toFind {
				low = mid + 1
			}
		}
	}
	return -1, nil // Not found.
}

// rangePiece gets the single value accession number search results for a
// piece of a range.
func rangePiece(ctx *context, prefix string, input string) (int, string,
	error) {
	num, err := strconv.Atoi(input)
	if err != nil {
		return 0, "", handle("Error in converting to int.", err)
	}
	res, err := accessionSearch(ctx, prefix, num)
	if err != nil {
		return 0, "", handle("Error in accession number search.", err)
	}
	return num, res, err
}
