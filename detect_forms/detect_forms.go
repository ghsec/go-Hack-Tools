 package main

 import (
 	"bufio"
 	"flag"
 	"fmt"
 	"net/http"
 	"os"
 	"sync"
 	"strings"
 	"crypto/tls"

 	"golang.org/x/net/html"
 )

 func main() {
 	// Parse the -c flag to specify the number of concurrent requests
 	concurrency := flag.Int("c", 20, "number of concurrent requests")

 	// Parse the -e flag to specify a list of file extensions to exclude
 	exclude := flag.String("e", "", "comma-separated list of file extensions to exclude like js,txt,json,etc")
 	flag.Parse()

 	// Create a new HTTP client with disabled SSL certificate checking
 	client := &http.Client{
 		Transport: &http.Transport{
 			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
 		},
 	}

 	// Create a map of excluded extensions
 	excludedExtensions := make(map[string]bool)
 	if *exclude != "" {
 		for _, ext := range strings.Split(*exclude, ",") {
 			excludedExtensions[ext] = true
 		}
 	}

 	// Create a channel to send URLs to be processed
 	urlChan := make(chan string)

 	// Create a wait group to track the goroutines
 	var wg sync.WaitGroup

 	// Start the specified number of goroutines to process URLs
 	for i := 0; i < *concurrency; i++ {
 		wg.Add(1)
 		go func() {
 			defer wg.Done()
 			for url := range urlChan {
 				// Check if the URL has an excluded extension
 				if excludedExtensions[getFileExtension(url)] {
 					continue
 				}

 				// Send an HTTP GET request to the URL using the custom client
 				resp, err := client.Get(url)
 				if err != nil {
 					fmt.Printf("%s\n", url)
 					continue
 				}
 				defer resp.Body.Close()

 				// Check the response for HTML forms
 				if containsForm(resp) {
 					fmt.Println(url)
 				}
 			}
 		}()
 	}

 	// Create a new scanner to read from stdin
 	scanner := bufio.NewScanner(os.Stdin)

 	// Read each line from stdin and send the URL to be processed
 	for scanner.Scan() {
 		urlChan <- scanner.Text()
 	}
 	close(urlChan)

 	// Wait for all goroutines to finish
 	wg.Wait()
 }

 // containsForm checks the HTTP response for HTML forms.
 // It returns true if a form is found, and false otherwise.
 func containsForm(resp *http.Response) bool {
 	// Parse the response body as HTML
 	doc, err := html.Parse(resp.Body)
 	if err != nil {
 		fmt.Printf("Error parsing HTML: %s\n", err)
 		return false
 	}

 	// Search the HTML document for form elements
 	var found bool
 	var f func(*html.Node)
 	f = func(n *html.Node) {
 		if found {
 			return
 		}
 		if n.Type == html.ElementNode && n.Data == "form" {
 			found = true
 			return
 		}
 		for c := n.FirstChild; c != nil; c = c.NextSibling {
 			f(c)
 		}
 	}
 	f(doc)
 	return found
 }

 // getFileExtension returns the file extension for the given URL.
 // If the URL does not have a file extension, an empty string is returned.
 func getFileExtension(url string) string {
 	lastDot := strings.LastIndex(url, ".")
 	if lastDot == -1 {
 		return ""
 	}
 	lastSlash := strings.LastIndex(url, "/")
 	if lastSlash > lastDot {
 		return ""
 	}
 	return url[lastDot+1:]
 }

