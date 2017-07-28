package main

import (
	"log"
	"fmt"
	"os/user"
	"strings"
	"strconv"
	"os"
	"bufio"
)

type Context struct {
	searchDest  string
	prefixCache map[string]prefixResult
	outFile     *os.File
}

func matchSequencesCaller() error {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	input := usr.HomeDir + "/sequence_lists/blast/db/FASTA/nt.gz.reduced.sorted.natural.dedup.reduced.txt"
	output := usr.HomeDir + "/sequence_lists/blast/db/FASTA/big_run_2.txt"
	// Setup
	ctx := Context{}
	ctx.outFile, err = os.Create(output)
	if err != nil {
		return handle("Error in creating outfile.", err)
	}
	ctx.searchDest = usr.HomeDir + "/sequence_lists/genbank_reduced"
	ctx.prefixCache = make(map[string]prefixResult)
	matchSequences(ctx, input)
	return err
}

func matchSequences(ctx Context, input string) error {
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
		if !strings.Contains(line, ": ") {
			continue
		}
		parts := strings.Split(line, ": ")
		prefix := parts[0]
		toFind := parts[1]
		if !strings.Contains(toFind, "-") {
			// Dealing with a point value
			findSingleValue(ctx, prefix, toFind)
		} else {
			// Dealing with a range
			findRange(ctx, prefix, toFind)
		}
	}
	return err
}

func findSingleValue(ctx Context, prefix string, toFind string) error {
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
	}
	return err
}

func findRange(ctx Context, prefix string, toFind string) error {
	parts := strings.Split(toFind, "-")
	startNum, startRes, err := rangePiece(ctx, prefix, parts[0])
	if err != nil {
		return handle("Error in finding results for range start.", err)
	}
	endNum, endRes, err := rangePiece(ctx, prefix, parts[1])
	if err != nil {
		return handle("Error in finding results for range end.", err)
	}

	// Result was a range, and the start/end numbers matched to the same range.
	// This means that all the intermediate range values must also be included in
	// the result.
	if strings.Contains(startRes, "-") && startRes == endRes {
		out := fmt.Sprintf("%s%-13s | %s", prefix, toFind, startRes)
		writeLine(out, ctx.outFile)
	} else {
		// Otherwise just go through the range sequentially and check each one.
		for i := startNum; i <= endNum; i++ {
			err = findSingleValue(ctx, prefix, strconv.Itoa(i))
			if err != nil {
				handle("Error in searching for point value.", err)
			}
		}
	}

	return err
}

func writeLine(input string, outFile *os.File) error {
	fmt.Println(input)
	_, err := outFile.WriteString(input + "\n")
	if err != nil {
		return handle("Error in writing line.", err)
	}
	return err
}

type prefixResult struct {
	accessionNums []string
	valueToFile   map[string]string
}

func prefixToResults(ctx Context, prefix string) (prefixResult, error) {
	// Setup
	var err error
	accessionNums := []string{}
	valueToFile := make(map[string]string)

	res, present := ctx.prefixCache[prefix]
	if present {
		return res, err
	}

	// Get results from disk
	cmd := fmt.Sprintf("sift '%s' '%s' -w --binary-skip | sort -k2 -n", prefix, ctx.searchDest)
	stdout, _, err := commandVerboseOnErr(cmd)
	if err != nil {
		return res, handle("Error in calling search utility.", err)
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
			snip := pieces[0][len(ctx.searchDest)+1:]
			snip = snip[:len(snip)-len(prefix)-1]
			valueToFile[key] = snip
		}
	}

	// Add to cache
	res = prefixResult{accessionNums, valueToFile}
	ctx.prefixCache[prefix] = res

	return res, err
}

func accessionSearch(ctx Context, prefix string, targetNum int) (string, error) {
	// Setup
	var err error
	accessionNums := []string{}
	valueToFile := make(map[string]string)

	// Get prefix to file search results
	prefixRes, err := prefixToResults(ctx, prefix)
	if err != nil {
		return "", handle("Error in getting file results for the prefix.", err)
	}
	accessionNums = prefixRes.accessionNums
	valueToFile = prefixRes.valueToFile

	res, err := arraySearch(accessionNums, targetNum)
	if err != nil {
		return "", handle("Error in searching array", err)
	}

	// Format results
	if res > 0 && res < len(accessionNums) {
		resFile := valueToFile[accessionNums[res]]
		resFile = resFile[:len(resFile)-4]
		return fmt.Sprintf("%-13s | %s", accessionNums[res], resFile), err
	}
	return "", err
}

func arraySearch(toSearch []string, toFind int) (int, error) {
	// Do a binary search to match the range
	n := len(toSearch)
	low, high := 0, n-1
	for low <= high {
		mid := low + (high-low)/2 // Midpoint
		lookAt := toSearch[mid]

		if !strings.Contains(lookAt, "-") {
			// Point val in array
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

func rangePiece(ctx Context, prefix string, input string) (int, string, error) {
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