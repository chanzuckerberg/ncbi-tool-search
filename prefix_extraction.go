package main

import (
	"log"
	"io/ioutil"
	"os"
	"fmt"
	"bufio"
	"strings"
	"os/user"
	"github.com/skarademir/naturalsort"
	"sort"
)

// Not really used in final search routine but was helpful for learning about
// the prefxies found in each file.

// Reduces files to just a unique list of prefixes found in each file.
// Example values on genbank files.
func prefixExtraction() error {
	home := getUserHome()
	inputDir := home + "/sequence_lists/genbank_reduced"
	files, err := ioutil.ReadDir(inputDir)
	if err != nil {
		return handle("Error in reading directory", err)
	}
	for _, f := range files {
		outputDir := home + "/sequence_lists/genbank_prefixes"
		if err = os.MkdirAll(outputDir, os.ModePerm); err != nil {
			return handle("Error in making dest folder", err)
		}
		outFile, err := os.Create(outputDir + "/" + f.Name())
		if err != nil {
			return handle("Error in creating out file", err)
		}
		processFilePrefixes(inputDir+ "/" + f.Name(), outFile)
	}
	return err
}

// Gets a unique list of the prefixes found in a single file.
func prefixExtractionSingle() error {
	home := getUserHome()
	inFile := home + "/sequence_lists/blast/db/FASTA/inFile.txt"
	outPath := home + "/sequence_lists/blast/db/FASTA/outFile.txt"
	outFile, err := os.Create(outPath)
	if err != nil {
		return handle("Error in prefix extraction from file", err)
	}
	if err = processFilePrefixes(inFile, outFile); err != nil {
		return handle("Error in getting file prefixes", err)
	}
	return err
}

// Gets the prefixes from a file and writes them to a new out file.
func processFilePrefixes(pathName string, outFile *os.File) error {
	fmt.Println("File: " + pathName)

	// Open the file
	file, err := os.Open(pathName)
	if err != nil {
		return handle("Error in opening file", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	prefixSet := make(map[string]bool)

	// Go line by line
	for scanner.Scan() {
		line := scanner.Text()
		// Get set of seen prefixes
		if prefix := getPrefix(line); prefix != "" {
			prefixSet[prefix] = true
		}
	}
	// Write results to file
	for k, _ := range prefixSet {
		outFile.WriteString(fmt.Sprintf("%s\n", k))
	}
	if err = scanner.Err(); err != nil {
		return handle("Error in scanning lines", err)
	}
	return err
}

// Gets the prefix from a line.
func getPrefix(line string) string {
	if strings.Contains(line, ":") {
		res := strings.Split(line, ":")
		return res[0]
	}
	return ""
}

// Gets a sorted list of the prefixes from a pre-processed directory
// structure of files of prefix names.
func prefixListing() error {
	inputDir := getUserHome() + "/sequence_lists/genbank_prefixes"
	files, err := ioutil.ReadDir(inputDir)
	if err != nil {
		return handle("Error in reading dir", err)
	}
	// Gets a list of the prefixes found in the files and puts them into a
	// sorted list.
	res := []string{}
	for _, f := range files {
		fname := f.Name()
		fileResult, err := prefixListForFile(inputDir+ "/" + fname, fname)
		if err != nil {
			return handle("Error in getting prefix list from file", err)
		}
		res = append(res, fileResult)
	}
	sort.Sort(naturalsort.NaturalSort(res))
	for _, v := range res {
		fmt.Println(v)
	}
	return err
}

// Makes a comma-separated list of the lines in the file.
func prefixListForFile(fname string, base string) (string, error) {
	res := base[:len(base)-4] + ": "
	file, err := os.Open(fname)
	if err != nil {
		return "", handle("Error in opening file", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		res += line + ", "
	}
	res = res[:len(res)-2]
	return res, err
}