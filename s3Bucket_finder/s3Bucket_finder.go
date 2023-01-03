package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

func main() {

	var concurrency int
	flag.IntVar(&concurrency, "c", 1, "Number of concurrent HTTP requests")
	flag.Parse()

	// Create an HTTP client with disabled certificate checking
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		urlStr := scanner.Text()

		wg.Add(1)
		go func(urlStr string) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Make HTTP request
			resp, err := client.Get(urlStr)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			// Read response body
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading response body for %s: %s\n", urlStr, err)
				return
			}

			// Check for Amazon S3 subdomains in response body
			if strings.Contains(string(body), "s3.amazonaws.com") {
				// Extract Amazon S3 subdomain URLs
				subdomainURLs := extractSubdomainURLs(string(body))

				for _, subdomainURL := range subdomainURLs {
					// Make HTTP request to Amazon S3 subdomain
					subResp, err := client.Get(subdomainURL)
					if err != nil {
						fmt.Printf("Error making request to %s: %s\n", subdomainURL, err)
						continue
					}
					defer subResp.Body.Close()

					if subResp.StatusCode == 200 {
						fmt.Printf("URL: %s\n", urlStr)
						fmt.Printf("Open bucket: %s\n\n", subdomainURL)
					} else if subResp.StatusCode == 404 {
						fmt.Printf("URL: %s\n", urlStr)
						fmt.Printf("Vulnerable to claim: %s\n\n", subdomainURL)
					}
				}
			}
		}(urlStr)
	}

	wg.Wait()
}

// extractSubdomainURLs extracts Amazon S3 subdomain URLs from the given string.
func extractSubdomainURLs(s string) []string {
	var subdomainURLs []string

	// Extract all URLs from the string
	urls := extractURLs(s)

	// Filter URLs to keep only Amazon S3 subdomain URLs
	for _, u := range urls {
		if strings.Contains(u, "s3.amazonaws.com") {
			subdomainURLs = append(subdomainURLs, u)
		}
	}

	return subdomainURLs
}

// extractURLs extracts all URLs from the given string.
func extractURLs(s string) []string {
	var urls []string

	// Use a regular expression to extract all URLs from the string
	re := regexp.MustCompile(`https?://[^\s"]+`)
	matches := re.FindAllString(s, -1)

	for _, m := range matches {
		urls = append(urls, m)
	}

	return urls
}

