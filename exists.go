package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

// custom dns resolver with cache
var googleAPIsIP net.IP
var googleStorageDomain string

func init() {
	googleStorageDomain = "storage.googleapis.com"
	ips, err := net.LookupIP(googleStorageDomain)
	if err != nil {
		log.Fatalf("issue getting ip of storage %v", err)
	}
	for _, record := range ips {
		// make sure ip is ipv4
		if record.To4() != nil {
			googleAPIsIP = record
			break
		}
	}

	if googleAPIsIP == nil {
		log.Fatalf("issue getting ip")
	}

	log.Println("got ip", googleAPIsIP)

	// googleAPIsIP = ips[0]

	var (
		dnsResolverIP        = "8.8.8.8:53" // Google DNS resolver.
		dnsResolverProto     = "udp"        // Protocol to use for the DNS resolver
		dnsResolverTimeoutMs = 5000         // Timeout (ms) for the DNS resolver (optional)
	)

	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Duration(dnsResolverTimeoutMs) * time.Millisecond,
				}
				return d.DialContext(ctx, dnsResolverProto, dnsResolverIP)
			},
		},
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr == googleStorageDomain+":443" {
			return net.Dial("tcp", googleAPIsIP.String()+":443")
		}
		return dialer.DialContext(ctx, network, addr)
	}
	http.DefaultTransport.(*http.Transport).DialContext = dialContext
}

func main() {
	//stats
	var numBuckets int
	var numBucketsFound int
	var numWordlist int
	// colors
	red := color.New(color.FgRed)
	green := color.New(color.FgGreen)
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
		numWordlist += 1
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
		// timeout to stop rate limit of 25 requests per second
		time.Sleep(time.Millisecond * 250)
		// ignore tls errors
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		http.DefaultClient.Timeout = time.Minute
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}

		if resp.StatusCode == 200 {
			green.Println(wordlist[bucket], "bucket found and has read access")
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			bodyString := string(bodyBytes)
			green.Println(bodyString)
			// log results
			logs.WriteString(bodyString)
			// timeout to stop rate limit of 25 requests per second
			time.Sleep(time.Millisecond * 250)
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
			numBuckets += 1

		} else if resp.StatusCode == 404 || resp.StatusCode == 400 {
			continue
		} else if resp.StatusCode == 401 {
			fmt.Println(wordlist[bucket], "bucket found but no read access")
			numBuckets += 1
		} else {
			red.Println("An error occured while getting bucket", wordlist[bucket])
			red.Println("Status Code:", resp.StatusCode)
		}

	}
	fmt.Println("Tried", numWordlist, "bucket names. Found", numBucketsFound, "buckets with read access out of", numBuckets, "buckets.")
	logs.Close()
}
