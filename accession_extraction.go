package main

import (
	"log"
	"os"
	"bufio"
	"fmt"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"path/filepath"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"strings"
	"sync"
	"time"
)

func accessionExtraction() {
	// Setup
	//sess := session.Must(session.NewSession())
	//downloader := s3manager.NewDownloader(sess)

	// Concurrency setup
	wg := sync.WaitGroup{}
	queue := make(chan string)

	for worker := 0; worker < 10; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range queue {
				singleFileFlow(work) // blocking wait for work
			}
		}()
	}

	// Go through files in source list
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

func singleFileFlow(file string) {
	_, err := os.Stat("sequence_lists" + file + ".txt")
	if err == nil {
		log.Printf("File %s is processed already.", file)
		return
	}
	log.Printf("Started: %s", file)

	// Download file
	err = rsyncFile(file)
	if err != nil {
		err = newErr("Error in downloading file from S3.", err)
		log.Print(err)
		return
	}

	// Process
	input := "source_files" + file
	dest := "sequence_lists" + file + ".txt"
	dir := filepath.Dir(dest) // Make sub-folders
	os.MkdirAll(dir, os.ModePerm)
	var cmd string
	defer timeTrack(time.Now(), "Processing " + file)
	if strings.Contains(file, "genbank") {
		cmd = fmt.Sprintf("zgrep 'ACCESSION' %s | awk '{print $2}' > %s", input,
			dest)
	} else if strings.Contains(file, "FASTA") {
		cmd = fmt.Sprintf("zgrep '>' %s | awk '{print $1}' | cut -c 2- > %s", input, dest)
		//log.Print("No processing match found.")
		//return
	}
	_, _, err = commandVerboseOnErr(cmd)

	// Delete file
	if err = os.Remove(input); err != nil {
		err = newErr("Error in removing file.", err)
		log.Print(err)
		return
	}

	log.Printf("Finished: %s", file)
}

func rsyncFile(file string) error {
	// Skip if file exists
	_, err := os.Stat("source_files" + file)
	if err == nil {
		return err
	}

	dir := filepath.Dir(file)
	os.MkdirAll("source_files" + dir, os.ModePerm)

	track := "Rsync download from mirror of " + file
	server := "mirrors.vbi.vt.edu::ftp.ncbi.nih.gov"
	//if rand.Float32() < 0.2 { // Randomly use different servers
	//	server = "rsync://ftp.ncbi.nlm.nih.gov"
	//	track = "Rsync download from NCBI actual"
	//}
	defer timeTrack(time.Now(), track)

	origin := server + file
	dest := "source_files" + file
	cmd := fmt.Sprintf("rsync -arzv --no-motd %s %s", origin, dest)
	_, _, err = commandVerboseOnErr(cmd)
	if err != nil {
		log.Print("Error in rsync.")
		return err
	}
	//log.Print("File downloaded. Head:")
	//stdout, _, err := commandVerboseOnErr("zcat < " + dest + " | head -n 10")
	//log.Print(stdout)
	return err
}

func downloadFile(downloader *s3manager.Downloader, file string) error {
	// Skip if file exists
	_, err := os.Stat("source_files" + file)
	if err == nil {
		return err
	}

	dir := filepath.Dir(file)
	os.MkdirAll("source_files" + dir, os.ModePerm)
	to_create := "source_files" + file
	log.Print("File to create: " + to_create)
	f, err := os.Create(to_create)
	if err != nil {
		return fmt.Errorf("Failed to create file %q, %v", to_create, err)
	}
	_, err = downloader.Download(f, &s3.GetObjectInput{
		Bucket: aws.String("czbiohub-ncbi-store"),
		Key:    aws.String(file),
	})
	if err != nil {
		return fmt.Errorf("Failed to download file, %v", err)
	}
	log.Print("File downloaded: " + file)
	return err
}