package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
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
	//stats
	var numBuckets int
	var numBucketsFound int
	// wordlist to check
	wordlist := []string{}
	// flags
	bucketFile := flag.String("file", "list.txt", "File of buckets to check.")
	logFile := flag.String("log", "logs.txt", "File to output results.")
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
		// check if bucket name contains www.
		if strings.Contains(fileScanner.Text(), "www.") {
			// remove www.
			bucket := strings.Replace(fileScanner.Text(), "www.", "", -1)
			// add to array
			wordlist = append(wordlist, bucket)
		} else {
			// add to array
			wordlist = append(wordlist, fileScanner.Text())
		}
		numBuckets += 1
	}
	readFile.Close()
	// create logs file
	logs, logErr := os.Create(*logFile)
	if logErr != nil {
		fmt.Println(logErr)
	}
	// base url to crawl
	permUrl := "https://storage.googleapis.com/storage/v1/b/"

	// function to check permissions of gcp storage bucket
	for bucket := range wordlist {
		// compile url for crawling
		url := permUrl + wordlist[bucket]
		objectUrl := url + "/o"
		fmt.Println("Checking ", wordlist[bucket])
		// ignore tls errors
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}

		if resp.StatusCode == 200 {
			fmt.Println("Bucket found and has read access")
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			bodyString := string(bodyBytes)
			fmt.Println(bodyString)
			// log results
			logs.WriteString(bodyString)
			//get content listing
			contentResp, err := http.Get(objectUrl)
			if err != nil {
				log.Fatal(err)
			}
			contentBytes, err := io.ReadAll(contentResp.Body)
			if err != nil {
				log.Fatal(err)
			}
			contentString := string(contentBytes)
			// log results
			logs.WriteString(contentString)
			numBucketsFound += 1

		} else if resp.StatusCode == 404 {
			fmt.Println("Not a real bucket")
		} else if resp.StatusCode == 401 {
			fmt.Println("Bucket found but no read access")
		}

	}
	fmt.Println("Found", numBucketsFound, "buckets with read access out of", numBuckets, "buckets.")
	/**
	for bucket := range wordlist {
		// compile url for crawling
		url := "https://" + wordlist[bucket] + contentUrl
		fmt.Println("Checking ", wordlist[bucket])
		// ignore tls errors
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}

		if resp.StatusCode == 200 {
			fmt.Println("Bucket found and has read access")
			// log results
			logs.WriteString(wordlist[bucket] + " - " + "Bucket found and has read access" + "\n")
		} else if resp.StatusCode == 404 {
			fmt.Println("Not a real bucket")
			// log results
			logs.WriteString(wordlist[bucket] + " - " + "Not a real bucket" + "\n")
		} else if resp.StatusCode == 403 {
			fmt.Println("Bucket found but no read access")
			// log results
			logs.WriteString(wordlist[bucket] + " - " + "Bucket found but no read access" + "\n")
		}

	} **/
	logs.Close()
}
