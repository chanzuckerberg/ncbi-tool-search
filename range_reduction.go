package main

import (
	"io/ioutil"
	"fmt"
	"os"
	"bufio"
	"strconv"
	"strings"
	"path/filepath"
)

// Takes in a directory and creates copies of the files with point values
// reduced into ranges. E.g. AC1, AC2, AC3 -> AC: 1-3.
func rangeReduction() error {
	home := getUserHome()
	dir := home + "/sequence_lists/genbank"
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return handle("Error in range reduction", err)
	}
	for _, f := range files {
		folder := home + "/sequence_lists/genbank_reduced"
		if err = os.MkdirAll(folder, os.ModePerm); err != nil {
			return handle("Error in making results folder", err)
		}
		outFile, err := os.Create(folder + "/" + f.Name())
		if err != nil {
			return handle("Error in making out file", err)
		}
		processFile(dir + "/" + f.Name(), outFile)
	}
	return err
}

// Runs the range reduction process on a single file. E.g. AC1, AC2, AC3 ->
// AC: 1-3.
func rangeReductionSingle() error {
	home := getUserHome()
	folder := home + "/sequence_lists/blast/db/FASTA/"
	fname := "nr.gz.trimmed.sorted.txt"
	outFile, err := os.Create(folder + "nr.gz.trimmed.sorted.reduced.txt")
	if err != nil {
		return handle("Error in creating out file", err)
	}
	processFile(folder + fname, outFile)
	return err
}

// Trims version numbers from lines of accession number sequences from a
// whole directory.
func trimWholeDir() {
	dir := getUserHome() + "/sequence_lists/refseq"
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || string(filepath.Base(path)[0]) == "." {
			return nil
		}
		if err = formatOneFile(path); err != nil {
			return handle("Error in formatting file: " + path, err)
		}
		return nil
	})
}

// Trims version numbers and formats lines of accession numbers in a single
// file. Doesn't reduce ranges. Only formats existing point value lines.
func formatOneFile(input string) error {
	// Setup
	trimFolder := getUserHome() + "/sequence_lists/refseq_trimmed"
	dirSnip := filepath.Dir(input)
	dirSnip = dirSnip[len("/Users/jsheu/sequence_lists/refseq"):]
	folder := trimFolder + dirSnip
	if err := os.MkdirAll(folder, os.ModePerm); err != nil {
		return handle("Error in making out folder", err)
	}
	name := filepath.Base(input)
	name = name[:len(name)-4]
	outFile, err := os.Create(folder + "/" + name + ".trimmed.txt")
	if err != nil {
		return handle("Error in creating out file", err)
	}

	var prefix string
	var number int
	// Open the file
	file, err := os.Open(input)
	if err != nil {
		return handle("Error in opening file: " + input, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	// Go line by line
	for scanner.Scan() {
		line := scanner.Text()
		prefix, number, err = splitLine(line)
		if err != nil {
			continue
		}
		out := fmt.Sprintf("%s: %d\n", prefix, number)
		outFile.WriteString(out)
		fmt.Print(out)
	}
	if err = scanner.Err(); err != nil {
		return handle("Error in reading lines from file", err)
	}
	return err
}

// processFile takes a file and creates a copy with reduced and formatted
// ranges. E.g. AC1, AC2, AC3 -> AC: 1-3.
func processFile(pathName string, outFile *os.File) error {
	fmt.Println("File: " + pathName)
	var prefix, curPrefix string
	var curNumber, rangeStart, number int

	// Open the file
	file, err := os.Open(pathName)
	if err != nil {
		return handle("Error in processing single file", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	// Go line by line
	for scanner.Scan() {
		line := scanner.Text()
		prefix, number, err = splitLine(line)
		if err != nil {
			continue
		}
		if curPrefix == "" && curNumber == 0 {
			// First prefix and number
			curPrefix = prefix
			curNumber = number
			rangeStart = number
			continue
		}
		if prefix == curPrefix && number == (curNumber+1) {
			// Continued sequence. Incrementing curNumber.
			curNumber = number
		} else {
			// Sequence broken. Write out the point value or range.
			if rangeStart == curNumber {
				outFile.WriteString(fmt.Sprintf("%s: %d\n", prefix, rangeStart))
			} else {
				outFile.WriteString(fmt.Sprintf("%s: %d-%d\n", prefix, rangeStart,
					curNumber))
			}
			// Update
			curPrefix = prefix
			curNumber = number
			rangeStart = number
		}
	}
	// Last write out
	if rangeStart == curNumber {
		outFile.WriteString(fmt.Sprintf("%s: %d\n", prefix, rangeStart))
	} else {
		outFile.WriteString(fmt.Sprintf("%s: %d-%d\n", prefix, rangeStart,
			curNumber))
	}
	if err = scanner.Err(); err != nil {
		return handle("Error in reading lines from file", err)
	}
	return err
}

// Splits the line into the prefix and the numerical value. Could probably
// replace with some RegEx...
func splitLine(line string) (string, int, error) {
	var number int
	var err error
	for i, char := range line {
		if _, err := strconv.Atoi(string(char)); err != nil {
			continue // Skip if not a number
		}
		if i == 0 { // Beginning of the line
			number, err = strconv.Atoi(string(line))
			if err != nil {
				return "", 0, err
			}
			return "", number, err
		}
		prefix := line[:i]
		numStr := line[i:]
		if strings.Contains(numStr, ".") {
			parts := strings.Split(numStr, ".")
			numStr = parts[0]
		}
		if number, err = strconv.Atoi(numStr); err != nil {
			return "", 0, err
		}
		return prefix, number, err
	}
	return "", 0, err
}