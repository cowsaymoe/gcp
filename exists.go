package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func check_perms(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	// convert body to string
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	body := buf.String()
	fmt.Println(body)
	return body
}

func main() {
	// wordlist to check
	wordlist := []string{}
	// flags
	bucketFile := flag.String("file", "list.txt", "File of buckets to check")
	flag.Parse()
	fmt.Println("Checking buckets in", *bucketFile)
	// open file
	readFile, err := os.Open(*bucketFile)
	if err != nil {
		fmt.Println(err)
	}
	// read file
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	// add each line to array of buckets
	for fileScanner.Scan() {
		wordlist = append(wordlist, fileScanner.Text())
	}
	readFile.Close()
	// base url to crawl
	base := ".storage.googleapis.com"

	// function to check permissions of gcp storage bucket

	for bucket := range wordlist {
		// compile url for crawling
		url := "https://" + wordlist[bucket] + base
		fmt.Println("Checking ", wordlist[bucket])
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}

		if resp.StatusCode == 200 {
			fmt.Println("Bucket found and has read access")
		} else if resp.StatusCode == 404 {
			fmt.Println("Not a real bucket")
		} else if resp.StatusCode == 403 {
			fmt.Println("Bucket found but no read access")
		}
	}
}
