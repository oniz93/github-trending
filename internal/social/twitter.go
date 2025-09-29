package social

import (
	"fmt"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/teomiscia/github-trending/internal/models"
)

// TwitterClient implements the Poster interface for Twitter.
type TwitterClient struct {
	client *twitter.Client
}

// NewTwitterClient creates a new TwitterClient.
func NewTwitterClient(apiKey, apiSecretKey, accessToken, accessSecret string) *TwitterClient {
	config := oauth1.NewConfig(apiKey, apiSecretKey)
	token := oauth1.NewToken(accessToken, accessSecret)
	httpClient := config.Client(oauth1.NoContext, token)

	// Twitter client
	client := twitter.NewClient(httpClient)

	return &TwitterClient{
		client: client,
	}
}

// Post posts a repository to Twitter.
func (c *TwitterClient) Post(repo models.Repository) error {
	message := fmt.Sprintf("Check out this trending repository on GitHub: %s\n\n%s\n\n%s", repo.FullName, repo.Description.String, repo.HTMLURL)

	// Truncate message if it's too long for Twitter
	if len(message) > 280 {
		message = message[:277] + "..."
	}

	_, _, err := c.client.Statuses.Update(message, nil)
	return err
}
