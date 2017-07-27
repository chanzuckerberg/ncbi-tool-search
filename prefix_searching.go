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

func bigMatcher() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.Open(usr.HomeDir + "/sequence_lists/blast/db/FASTA/nt.gz" +
		".reduced.sorted.natural.dedup.reduced.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	// Go line-by-line
	count := 0
	prefixCache := make(map[string]prefixResult)
	outdir := usr.HomeDir + "/sequence_lists/blast/db/FASTA/"
	outFile, _ := os.Create(outdir + "big_run_1.txt")
	out := fmt.Sprintf("%-9s | %16s | %s \n", "Target", "Found in range", "In file")
	outFile.WriteString(out)
	fmt.Print(out)
	for scanner.Scan() {
		if count > 100000 {
			return
		}
		line := scanner.Text()
		parts := strings.Split(line, ": ")
		prefix := parts[0]
		numOrRange := parts[1]
		if strings.Contains(numOrRange, "-") { // Dealing with a range
			parts = strings.Split(numOrRange, "-")
			startNum, err := strconv.Atoi(parts[0])
			if err != nil {
				log.Fatal(err)
			}
			endNum, err := strconv.Atoi(parts[1])
			if err != nil {
				log.Fatal(err)
			}
			found, res := prefixSearching(prefix, startNum, prefixCache)
			if found {
				out := fmt.Sprintf("%s%-7d | %s\n", prefix, startNum, res)
				fmt.Print(out)
				outFile.WriteString(out)
			} else {
				fmt.Print(res)
				outFile.WriteString(res)
			}
			found, res = prefixSearching(prefix, endNum, prefixCache)
			if found {
				out := fmt.Sprintf("%s%-7d | %s\n", prefix, endNum, res)
				fmt.Print(out)
				outFile.WriteString(out)
			} else {
				fmt.Print(res)
				outFile.WriteString(res)
			}
		} else { // Dealing with a point value
			num, err := strconv.Atoi(numOrRange)
			if err != nil {
				log.Fatal()
			}
			found, res := prefixSearching(prefix, num, prefixCache)
			if found {
				out := fmt.Sprintf("%s%-7d | %s\n", prefix, num, res)
				fmt.Print(out)
				outFile.WriteString(out)
			} else {
				fmt.Print(res)
				outFile.WriteString(res)
			}
		}
		count += 1
	}
}

type prefixResult struct {
	accNums     []string
	range2fname map[string]string
}

func prefixSearching(prefix string, targetNum int, prefixCache map[string]prefixResult) (bool, string) {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	// Setup
	accNums := []string{}
	range2fname := make(map[string]string)

	// Check cache
	res, present := prefixCache[prefix]
	if present {
		accNums = res.accNums
		range2fname = res.range2fname
	} else {
		targetDir := usr.HomeDir + "/sequence_lists/genbank_reduced"
		cmd := fmt.Sprintf("sift '%s' '%s' -w --binary-skip | sort -k2 -n", prefix, targetDir)
		stdout, _, err := commandWithOutput(cmd)
		if err != nil {
			log.Fatal(err)
		}

		// Process output
		lines := strings.Split(stdout, "\n")
		if len(lines) == 0 { // No results
			return false, ""
		}

		// Make mapping of accession num/ranges to filename:lines.
		// Make an ascending array of the accession num/ranges.
		for _, line := range lines {
			if strings.Contains(line, " ") {
				pieces := strings.Split(line, ": ")
				key := pieces[1]
				accNums = append(accNums, key)
				// Format the file names:lines
				snip := pieces[0][len(targetDir)+1:]
				snip = snip[:len(snip)-len(prefix)-1]
				range2fname[key] = snip
			}
		}

		// Add to cache
		result := prefixResult{accNums, range2fname}
		prefixCache[prefix] = result
	}

	// Do a binary search to match the file
	n := len(accNums)
	i, j := 0, n
	for i < j {
		h := i + (j-i)/2 // avoid overflow when computing h
		// i ≤ h < j
		curRange := accNums[h]
		// Dealing with a range
		if strings.Contains(curRange, "-") {
			pieces := strings.Split(curRange, "-")
			endNum, err := strconv.Atoi(pieces[1])
			if err != nil {
				log.Fatal(err)
			}
			if endNum < targetNum {
				i = h + 1
				continue
			} else {
				j = h
				continue
			}
		} else { // Dealing with point values
			curVal, err := strconv.Atoi(curRange)
			if err != nil {
				log.Fatal("Problem converting.")
			}
			if curVal < targetNum {
				i = h + 1
				continue
			} else {
				j = h
				continue
			}
		}
	}

	// Format results
	if i != 0 && i < len(accNums) {
		//input := prefix + strconv.Itoa(targetNum)
		resFile := range2fname[accNums[i]]
		resFile = resFile[:len(resFile)-4]
		//fmt.Printf("%s was found: %s => %s\n", input, accNums[i], resFile)
		return true, fmt.Sprintf("%-14s | %s", accNums[i], resFile)
	}
	return false, fmt.Sprintf("%s%d not found.\n", prefix, targetNum)
}