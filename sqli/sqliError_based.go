package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

var regexPatterns = []*regexp.Regexp{
	regexp.MustCompile(`You have an error in your SQL syntax`),
	regexp.MustCompile(`SQL query error`),
	regexp.MustCompile(`the right syntax to use near`),
	regexp.MustCompile(`String or binary data would be truncated`),
	regexp.MustCompile(`Invalid Input for SQL`),
	regexp.MustCompile(`has occurred in the vicinity of:`),
	regexp.MustCompile(`Unexpected end of command in statement \[`),
	regexp.MustCompile(`An illegal character has been found in the statement`),
	regexp.MustCompile(`org.hibernate.QueryException: unexpected char:`),
	regexp.MustCompile(`org.hibernate.QueryException: expecting '`),
	regexp.MustCompile(`java.sql.SQLSyntaxErrorException`),
	regexp.MustCompile(`System.Data.OleDb.OleDbException`),
	regexp.MustCompile(`Unclosed quotation mark after the character string`),
	regexp.MustCompile(`mssql_query()`),
	regexp.MustCompile(`Incorrect syntax near`),
	regexp.MustCompile(`Sintaxis incorrecta cerca de`),
	regexp.MustCompile(`Syntax error in string in query expression`),
	regexp.MustCompile(`Unclosed quotation mark before the character string`),
	regexp.MustCompile(`Data type mismatch in criteria expression.`),
	regexp.MustCompile(`the used select statements have different number of columns`),
	regexp.MustCompile(`supplied argument is not a valid MySQL`),
	regexp.MustCompile(`Column count doesn't match value count at row`),
	regexp.MustCompile(`on MySQL result index`),
	regexp.MustCompile(`MySQL server version for the right syntax to use`),
	regexp.MustCompile(`Column count doesn't match`),
	regexp.MustCompile(`Ambiguous column name`),
	regexp.MustCompile(`valid MySQL result`),
	regexp.MustCompile(`Microsoft OLE DB Provider for SQL Server error`),
	regexp.MustCompile(`Oracle error`),
	regexp.MustCompile(`SQLite.Exception`),
	regexp.MustCompile(`System.Data.SQLite.SQLiteException`),
	regexp.MustCompile(`System.Data.SqlClient.SqlException`),
	regexp.MustCompile(`SQLITE_ERROR`),
	regexp.MustCompile(`SQL error`),
	regexp.MustCompile(`Dynamic SQL Error SQL error code`),
	regexp.MustCompile(`Procedure or function`),
	regexp.MustCompile(`SqlClient: Exception.`),
	regexp.MustCompile(`SQL syntax`),
	regexp.MustCompile(`PostgreSQL.`),
	regexp.MustCompile(`PG::`),
	regexp.MustCompile(`"SQLite3::"`),
	regexp.MustCompile(`OleDbException`),
	regexp.MustCompile(`com.mysql.jdbc.exceptions`),
	regexp.MustCompile(`syntax error at or near`),
	regexp.MustCompile(`unterminated quoted string at or near`),
	regexp.MustCompile(`UNION, INTERSECT or EXCEPT`),
	regexp.MustCompile(`Syntax error or access violation`),
	regexp.MustCompile(`SQLSTATE\[`),
	regexp.MustCompile(`EOF`),
}

func main() {
	payloads := []string{"'", "\"", "\\", "%"}

	verbose := flag.Bool("v", false, "Enable verbose output")
	proxy := flag.String("p", "", "Proxy address in the format http://host:port")
	flag.Parse()

	scanner := bufio.NewScanner(os.Stdin)

	var transport http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if *proxy != "" {
		proxyURL, err := url.Parse(*proxy)
		if err != nil {
			fmt.Printf("Error parsing proxy URL: %s\n", err)
			return
		}
		transport = &http.Transport{Proxy: http.ProxyURL(proxyURL), TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	}

	client := &http.Client{Transport: transport}

	for scanner.Scan() {
		rawURL := scanner.Text()

		u, err := url.Parse(rawURL)
		if err != nil {
			fmt.Printf("Error parsing URL: %s\n", err)
			continue
		}

		// Test parameters
		queryParams := u.Query()
		for param, values := range queryParams {
			for _, value := range values {
				for _, payload := range payloads {
					injectedValue := value + payload

					// Create a new URL object with the injected parameter
					injectedURL := *u
					injectedParams := url.Values{}
					for p, v := range queryParams {
						if p == param {
							injectedParams[p] = []string{injectedValue}
						} else {
							injectedParams[p] = v
						}
					}
					injectedURL.RawQuery = injectedParams.Encode()

					if *verbose {
						fmt.Printf("Sending request to %s\n", injectedURL.String())
					}

					req, err := http.NewRequest("GET", injectedURL.String(), nil)
					if err != nil {
						fmt.Printf("Error creating request: %s\n", err)
						continue
					}

					resp, err := client.Do(req)
					if err != nil {
						fmt.Printf("Error requesting %s: %s\n", injectedURL.String(), err)
						continue
					}

					body := bufio.NewScanner(resp.Body)
					for body.Scan() {
						line := body.Text()
						for _, pattern := range regexPatterns {
							if pattern.MatchString(line) {
								fmt.Printf("[Vulnerable]: %s Requested URL: %s\n", line, injectedURL.String())
								resp.Body.Close()
								return // Found vulnerability, stop script execution
							}
						}
					}

					if body.Err() != nil {
						fmt.Printf("Error reading response from %s: %s\n", injectedURL.String(), body.Err())
					}

					resp.Body.Close()
				}
			}
		}

		// Test path
		pathSegments := strings.Split(u.Path, "/")
		for i, segment := range pathSegments {
			for _, payload := range payloads {
				injectedSegment := segment + payload
				pathSegments[i] = injectedSegment

				// Create a new URL object with the injected path
				injectedURL := *u
				injectedURL.Path = strings.Join(pathSegments, "/")

				if *verbose {
					fmt.Printf("Sending request to %s\n", injectedURL.String())
				}

				req, err := http.NewRequest("GET", injectedURL.String(), nil)
				if err != nil {
					fmt.Printf("Error creating request: %s\n", err)
					continue
				}

				resp, err := client.Do(req)
				if err != nil {
					fmt.Printf("Error requesting %s: %s\n", injectedURL.String(), err)
					continue
				}

				body := bufio.NewScanner(resp.Body)
				for body.Scan() {
					line := body.Text()
					for _, pattern := range regexPatterns {
						if pattern.MatchString(line) {
							fmt.Printf("Vulnerable: %s Requested URL: %s\n", line, injectedURL.String())
							resp.Body.Close()
							return // Found vulnerability, stop script execution
						}
					}
				}

				if body.Err() != nil {
					fmt.Printf("Error reading response from %s: %s\n", injectedURL.String(), body.Err())
				}

				resp.Body.Close()
			}

			// Reset the segment to its original value
			pathSegments[i] = segment
		}
	}

	if scanner.Err() != nil {
		fmt.Printf("Error reading input: %s\n", scanner.Err())
	}
}

