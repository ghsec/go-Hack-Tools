package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
)

func main() {
	// Parse the flags
	var concurrency int
	flag.IntVar(&concurrency, "c", 1, "maximum number of concurrent requests")
	var matchCode int
	flag.IntVar(&matchCode, "mc", 0, "match response with provided status code")
	flag.Parse()

	// Set up a buffered channel to limit the number of concurrent requests
	requestChan := make(chan struct{}, concurrency)

	// Set up an HTTP client with a transport that ignores invalid certificates
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	var wg sync.WaitGroup

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		url := scanner.Text()

		// Increment the wait group counter
		wg.Add(1)

		// Launch a goroutine to make the HTTP request
		go func() {
			// Add a value to the request channel to block if the maximum concurrency has been reached
			requestChan <- struct{}{}

			// Make an HTTP GET request to the URL using the client
			resp, err := client.Get(url)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error making request:", err)
				return
			}
			defer resp.Body.Close()

			// Print the URL, response status code, and content length if the status code matches the filter
			if matchCode == 0 || resp.StatusCode == matchCode {
				contentLength := resp.Header.Get("Content-Length")
				fmt.Printf("[%d]    %s    [%s]\n", resp.StatusCode, url, contentLength)
			}

			// Decrement the wait group counter
			wg.Done()

			// Remove a value from the request channel
			<-requestChan
		}()
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading from stdin:", err)
		os.Exit(1)
	}

	// Wait for all goroutines to complete
	wg.Wait()
}

