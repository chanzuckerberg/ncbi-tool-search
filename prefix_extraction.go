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

func prefixExtraction() {
	usr, err := user.Current()
	if err != nil {
		log.Print("Couldn't get user's home directory.")
		log.Fatal(err)
	}

	inputDir := usr.HomeDir + "/sequence_lists/genbank_reduced"
	files, _ := ioutil.ReadDir(inputDir)
	for _, f := range files {
		outputDir := usr.HomeDir + "/sequence_lists/genbank_prefixes"
		os.MkdirAll(outputDir, os.ModePerm)
		outFile, err := os.Create(outputDir + "/" + f.Name())
		if err != nil {
			log.Fatal(err)
		}
		processFilePrefixes(inputDir+ "/" + f.Name(), outFile)
	}
}

func processFilePrefixes(pathName string, outFile *os.File) {
	fmt.Println("File: " + pathName)

	// Open the file
	file, err := os.Open(pathName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	prefixSet := make(map[string]bool)

	// Go line by line
	for scanner.Scan() {
		line := scanner.Text()
		prefix := getPrefix(line)
		if prefix != "" {
			prefixSet[prefix] = true
		}
	}
	for k, _ := range prefixSet {
		outFile.WriteString(fmt.Sprintf("%s\n", k))
	}
	if err = scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func getPrefix(line string) string {
	if strings.Contains(line, ":") {
		res := strings.Split(line, ":")
		return res[0]
	}
	return ""
}

func prefixListing() {
	usr, err := user.Current()
	if err != nil {
		log.Print("Couldn't get user's home directory.")
		log.Fatal(err)
	}

	inputDir := usr.HomeDir + "/sequence_lists/genbank_prefixes"
	files, _ := ioutil.ReadDir(inputDir)
	res := []string{}
	for _, f := range files {
		if err != nil {
			log.Fatal(err)
		}
		res = append(res, prefixListFile(inputDir+ "/" + f.Name(), f.Name()))
	}
	sort.Sort(naturalsort.NaturalSort(res))
	for _, v := range res {
		fmt.Println(v)
	}
}

func prefixListFile(fName string, base string) string {
	res := base[:len(base)-4] + ": "
	file, err := os.Open(fName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	// Go line-by-line
	for scanner.Scan() {
		line := scanner.Text()
		res += line + ", "
	}
	res = res[:len(res)-2]
	return res
}