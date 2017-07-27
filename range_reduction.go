package main

import (
	"log"
	"os/user"
	"io/ioutil"
	"fmt"
	"os"
	"bufio"
	"strconv"
	"strings"
)

func rangeReduction() {
	usr, err := user.Current()
	if err != nil {
		log.Print("Couldn't get user's home directory.")
		log.Fatal(err)
	}

	dirPath := usr.HomeDir + "/sequence_lists/genbank"
	files, _ := ioutil.ReadDir(dirPath)
	for _, f := range files {
		folder := usr.HomeDir + "/sequence_lists/genbank_reduced"
		os.MkdirAll(folder, os.ModePerm)
		outFile, err := os.Create(folder + "/" + f.Name())
		if err != nil {
			log.Fatal(err)
		}
		processFile(dirPath + "/" + f.Name(), outFile)
	}
}

func rangeReductionSingle() {
	usr, err := user.Current()
	if err != nil {
		log.Print("Couldn't get user's home directory.")
		log.Fatal(err)
	}
	folder := usr.HomeDir + "/sequence_lists/blast/db/FASTA/"
	fname := "nt.gz.reduced.sorted.natural.dedup.txt"
	outFile, err := os.Create(folder + "nt.gz.reduced.sorted.natural.dedup.reduced.txt")
	if err != nil {
		log.Fatal(err)
	}
	processFile(folder + fname, outFile)
}

func formatOneFile() {
	usr, _ := user.Current()
	pathName := usr.HomeDir + "/sequence_lists/blast/db/FASTA/nt.gz.txt"
	folder := usr.HomeDir + "/sequence_lists/blast/db/FASTA/"
	outFile, err := os.Create(folder + "nt.gz.trimmed.txt")
	if err != nil {
		log.Fatal(err)
	}

	var prefix string
	var number int
	// Open the file
	file, err := os.Open(pathName)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}
}

func processFile(pathName string, outFile *os.File) {
	fmt.Println("File: " + pathName)
	//count := 0
	curPrefix := ""
	curNumber := 0
	rangeStart := 0
	var prefix string
	var number int

	// Open the file
	file, err := os.Open(pathName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	// Go line by line
	for scanner.Scan() {
		//if count > 1000 {
		//	return
		//}

		line := scanner.Text()
		prefix, number, err = splitLine(line)
		if err != nil {
			continue
		}
		if curPrefix == "" && curNumber == 0 {
			curPrefix = prefix
			curNumber = number
			rangeStart = number
			continue
		}
		// Continued sequence
		if prefix == curPrefix && number == (curNumber+1) {
			curNumber = number
		} else { // Sequence broken
			if rangeStart == curNumber {
				outFile.WriteString(fmt.Sprintf("%s%d\n", prefix, rangeStart))
			} else {
				outFile.WriteString(fmt.Sprintf("%s%d-%d\n", prefix, rangeStart, curNumber))
			}
			curPrefix = prefix
			curNumber = number
			rangeStart = number
		}

		//count += 1
	}
	// Last write out
	if rangeStart == curNumber {
		outFile.WriteString(fmt.Sprintf("%s%d\n", prefix, rangeStart))
	} else {
		outFile.WriteString(fmt.Sprintf("%s%d-%d\n", prefix, rangeStart, curNumber))
	}
	curPrefix = prefix
	curNumber = number
	rangeStart = number

	if err = scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func splitLine(line string) (string, int, error) {
	var number int
	var err error
	for i, char := range line {
		// Is a number
		if _, err := strconv.Atoi(string(char)); err == nil {
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
	}
	return "", 0, err
}