package main
import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"crypto/tls"
)

func main() {
	// Define a command line flag for the number of concurrent requests
	var concurrency int
	flag.IntVar(&concurrency, "c", 1, "Number of concurrent requests")
	flag.Parse()

	// Define the pattern to search for
	pattern := `
(?i)((access_key|access_token|admin_pass|admin_user|algolia_admin_key|algolia_api_key|alias_pass|alicloud_access_key|amazon_secret_access_key|amazonaws|ansible_vault_password|aos_key|api_key|api_key_secret|api_key_sid|api_secret|api.googlemaps AIza|apidocs|apikey|apiSecret|app_debug|app_id|app_key|app_log_level|app_secret|appkey|appkeysecret|application_key|appsecret|appspot|auth_token|authorizationToken|authsecret|aws_access|aws_access_key_id|aws_bucket|aws_key|aws_secret|aws_secret_key|aws_token|AWSSecretKey|b2_app_key|bashrc password|bintray_apikey|bintray_gpg_password|bintray_key|bintraykey|bluemix_api_key|bluemix_pass|browserstack_access_key|bucket_password|bucketeer_aws_access_key_id|bucketeer_aws_secret_access_key|built_branch_deploy_key|bx_password|cache_driver|cache_s3_secret_key|cattle_access_key|cattle_secret_key|certificate_password|ci_deploy_password|client_secret|client_zpk_secret_key|clojars_password|cloud_api_key|cloud_watch_aws_access_key|cloudant_password|cloudflare_api_key|cloudflare_auth_key|cloudinary_api_secret|cloudinary_name|codecov_token|config|conn.login|connectionstring|consumer_key|consumer_secret|credentials|cypress_record_key|database_password|database_schema_test|datadog_api_key|datadog_app_key|db_password|db_server|db_username|dbpasswd|dbpassword|dbuser|deploy_password|digitalocean_ssh_key_body|digitalocean_ssh_key_ids|docker_hub_password|docker_key|docker_pass|docker_passwd|docker_password|dockerhub_password|dockerhubpassword|dot-files|dotfiles|droplet_travis_password|dynamoaccesskeyid|dynamosecretaccesskey|elastica_host|elastica_port|elasticsearch_password|encryption_key|encryption_password|env.heroku_api_key|env.sonatype_password|eureka.awssecretkey)[a-z0-9_ .\-,]{0,25})(=|>|:=|\|\|:|<=|=>|:).{0,5}['\"]([0-9a-zA-Z\-_=]{8,64})['\"]`

	// Compile the pattern
	r, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Println("Error compiling pattern:", err)
		return
	}

	// Create a new HTTP client
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}

	// Read URLs from standard input
	scanner := bufio.NewScanner(os.Stdin)

	// Use a WaitGroup to wait for all goroutines to complete
	var wg sync.WaitGroup

	// Create a channel to limit the number of concurrent requests
	requestChan := make(chan struct{}, concurrency)

	for scanner.Scan() {
		url := scanner.Text()

		// Add one to the WaitGroup count
		wg.Add(1)

		// Launch a goroutine to handle the request
		go func(url string) {
			// Decrement the WaitGroup count when the goroutine completes
			defer wg.Done()

			// Wait for a slot to open up in the channel
			requestChan <- struct{}{}
			defer func() { <-requestChan }()

			// Send a GET request to the URL
			resp, err := client.Get(url)
			if err != nil {
				//fmt.Println("Error sending request:", err)
				return
			}
			defer resp.Body.Close()

			// Check the Content-Type header to make sure it's a JavaScript file
			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(contentType, "javascript") {
				fmt.Printf("Skipping URL %s, not a JavaScript file\n", url)
				return
			}

			// Read the file line by line
			reader := bufio.NewReader(resp.Body)
			for {
				line, err := reader.ReadString('\n')
				if err == io.EOF {
					break
				}
				if err != nil {
					fmt.Println("Error reading file:", err)
					break
				}

				// Check if the pattern matches the current line
				if r.MatchString(line) {
					fmt.Println("URL ", url)
					fmt.Println("Secret ", line)
					fmt.Println()
				}
			}
		}(url)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading URLs:", err)
	}
}

