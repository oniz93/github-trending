package main

import (
	"fmt"
	"log"

	"github.com/dghubble/oauth1"
)

func main() {
	// 1. Get consumer key & secret
	var consumerKey, consumerSecret string
	fmt.Print("Enter your Twitter Consumer Key: ")
	fmt.Scanln(&consumerKey)
	fmt.Print("Enter your Twitter Consumer Secret: ")
	fmt.Scanln(&consumerSecret)

	config := oauth1.Config{
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
		CallbackURL:    "oob", // "Out of Band" for CLI apps
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: "https://api.twitter.com/oauth/request_token",
			AuthorizeURL:    "https://api.twitter.com/oauth/authorize",
			AccessTokenURL:  "https://api.twitter.com/oauth/access_token",
		},
	}

	// 2. Get request token
	requestToken, requestSecret, err := config.RequestToken()
	if err != nil {
		log.Fatalf("Failed to get request token: %v", err)
	}

	// 3. Get authorization URL
	authorizationURL, err := config.AuthorizationURL(requestToken)
	if err != nil {
		log.Fatalf("Failed to get authorization URL: %v", err)
	}

	// 4. Redirect user to authorization URL
	fmt.Printf("\nGo to the following URL to authorize the application:\n\n%s\n\n", authorizationURL.String())

	// 5. Get PIN
	var verifier string
	fmt.Print("Enter the PIN: ")
	fmt.Scanln(&verifier)

	// 6. Get access token
	accessToken, accessSecret, err := config.AccessToken(requestToken, requestSecret, verifier)
	if err != nil {
		log.Fatalf("Failed to get access token: %v", err)
	}

	fmt.Printf("\nAuthentication successful!\n")
	fmt.Printf("Access Token: %s\n", accessToken)
	fmt.Printf("Access Token Secret: %s\n", accessSecret)
	fmt.Println("\nPlease add these to your .env file as TWITTER_ACCESS_TOKEN and TWITTER_ACCESS_SECRET.")
}
