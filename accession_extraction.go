package main

import (
	"bufio"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Example of getting all the accession numbers from the files on an NCBI
// folder. Conditions for refseq/release.
func remoteFolderAccessionExtraction() {
	topFolder := "rsync://ftp.ncbi.nih.gov/refseq/release"
	// Sub-folders to process
	subFolders := []string{"complete"}

	// Go through all the sub-folders and get a list of files to process.
	template := "rsync -arzvn --itemize-changes --no-motd --copy-links --prune-empty-dirs %s %s"
	destPath := getUserHome() + "/sequence_lists"
	toProcess := []string{}
	for _, folder := range subFolders {
		originPath := topFolder + "/" + folder
		cmd := fmt.Sprintf(template, originPath, destPath)
		// Call rsync on the folder to get a recursive file listing.
		stdout, _, _ := commandWithOutput(cmd)
		lines := strings.Split(stdout, "\n")
		lines = lines[2 : len(lines)-4]
		for _, line := range lines {
			end := line[len(line)-7:]
			if !strings.Contains(line, "tmpold") &&
				len(line) > 10 &&
				(end == ".faa.gz" || end == ".fna.gz") {
				name := line[12:]
				name = "/refseq/release/" + name
				toProcess = append(toProcess, name)
			}
		}
	}

	// Concurrency setup. Creates up to 10 worker routines to process a single
	// file each.
	wg := sync.WaitGroup{}
	queue := make(chan string)
	for worker := 0; worker < 10; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range queue {
				singleFileFlow(work)
			}
		}()
	}

	// Send the files to process to the workers.
	for _, file := range toProcess {
		queue <- file
	}
	close(queue)
	wg.Wait()
	log.Print("Finished with everything.")
}

// Overall routine used for extracting all the accession numbers from the
// top-level Genbank files.
func accessionExtraction() {
	// Concurrency setup. Creates up to 10 worker routines to process a single
	// file each.
	wg := sync.WaitGroup{}
	queue := make(chan string)
	for worker := 0; worker < 10; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range queue {
				singleFileFlow(work)
			}
		}()
	}

	// Go through files in a source list and send them to the workers.
	if file, err := os.Open("source_list.txt"); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			item := scanner.Text()
			queue <- item
		}
		if err = scanner.Err(); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal(err)
	}
	close(queue)
	wg.Wait()
	log.Print("Finished with everything.")
}

// singleFileFlow downloads a file from the remote server and extracts the
// accession numbers.
func singleFileFlow(file string) error {
	home := getUserHome()
	var err error
	// Puts results in a secondary directory structure of sequence_lists.
	destPath := home + "/sequence_lists"

	if _, err = os.Stat(destPath + file + ".txt"); err == nil {
		log.Printf("File %s is processed already.", file)
		return err
	}
	log.Printf("Started: %s", file)

	// Download file
	if err = rsyncFile(file); err != nil {
		return handle("Error in downloading file", err)
	}

	// Process
	input := home + "/source_files" + file
	dest := home + "/sequence_lists" + file + ".txt"
	dir := filepath.Dir(dest) // Make sub-folders
	if err = os.MkdirAll(dir, os.ModePerm); err != nil {
		return handle("Error in creating sub-folders", err)
	}
	var cmd string
	// Time benchmarks for optimization hints
	defer timeTrack(time.Now(), "Processing "+file)
	if strings.Contains(file, "genbank") {
		// Genbank formatting: Get the line that says ACCESSION | Get the second
		// column.
		template := "sift -z --blocksize 10M 'ACCESSION' %s | awk '{print $2}' > %s"
		cmd = fmt.Sprintf(template, input, dest)
	} else {
		// FASTA file formatting: Get the header line with '>' | Get the first
		// column | Remove the first character '>'.
		template := "sift -z --blocksize 10M '>' %s | awk '{print $1}' | cut -c 2- > %s"
		cmd = fmt.Sprintf(template, input, dest)
	}
	_, _, err = commandVerboseOnErr(cmd)

	// Delete temp downloaded file
	if err = os.Remove(input); err != nil {
		return handle("Error in removing file.", err)
	}

	log.Printf("Finished: %s", file)
	return err
}

// rsyncFile downloads the file from remote.
func rsyncFile(file string) error {
	var err error
	// Skip if file exists
	if _, err = os.Stat("source_files" + file); err == nil {
		log.Printf("File %s downloaded already.", file)
		return err
	}

	home := getUserHome()
	dir := filepath.Dir(file)
	if err = os.MkdirAll(home+"/source_files"+dir, os.ModePerm); err != nil {
		return handle("Error in making destination dir", err)
	}

	track := "Rsync download from mirror of " + file
	server := "rsync://ftp.ncbi.nlm.nih.gov"
	//server := "mirrors.vbi.vt.edu::ftp.ncbi.nih.gov"
	//if rand.Float32() < 0.2 { // Randomly use different servers
	//	server = "rsync://ftp.ncbi.nlm.nih.gov"
	//	track = "Rsync download from NCBI actual"
	//}
	defer timeTrack(time.Now(), track) // Time benchmark

	origin := server + file
	dest := home + "/source_files" + file
	cmd := fmt.Sprintf("rsync -arzv --no-motd %s %s", origin, dest)
	_, _, err = commandVerboseOnErr(cmd)
	if err != nil {
		return handle("Error in downloading file", err)
	}
	//log.Print("File downloaded. Head:")
	//stdout, _, err := commandVerboseOnErr("zcat < " + dest + " | head -n 10")
	return err
}

// Download file from S3
func downloadFile(downloader *s3manager.Downloader, file string) error {
	var err error
	// Skip if file exists
	if _, err = os.Stat("source_files" + file); err == nil {
		log.Printf("File %s downloaded already.", file)
		return err
	}

	dir := filepath.Dir(file)
	if err = os.MkdirAll("source_files"+dir, os.ModePerm); err != nil {
		return handle("Error in making source_files dir", err)
	}
	to_create := "source_files" + file
	log.Print("File to create: " + to_create)
	f, err := os.Create(to_create)
	if err != nil {
		return handle("Failed to create file: "+to_create, err)
	}
	_, err = downloader.Download(f, &s3.GetObjectInput{
		Bucket: aws.String("czbiohub-ncbi-store"),
		Key:    aws.String(file),
	})
	if err != nil {
		return handle("Error in downloading file from S3", err)
	}
	log.Print("File downloaded: " + file)
	return err
}
