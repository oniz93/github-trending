package social

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dghubble/oauth1"
	twitter "github.com/g8rswimmer/go-twitter/v2"
	"github.com/teomiscia/github-trending/internal/models"
)

type oauth1Authorizer struct{}

func (a *oauth1Authorizer) Add(req *http.Request) {
	// The http.Client handles authorization
}

// TwitterClient implements the Poster interface for Twitter.
type TwitterClient struct {
	client *twitter.Client
}

// NewTwitterClient creates a new TwitterClient.
func NewTwitterClient(apiKey, apiSecretKey, accessToken, accessSecret string) *TwitterClient {
	config := oauth1.NewConfig(apiKey, apiSecretKey)
	token := oauth1.NewToken(accessToken, accessSecret)
	httpClient := config.Client(oauth1.NoContext, token)

	client := &twitter.Client{
		Authorizer: &oauth1Authorizer{},
		Client:     httpClient,
		Host:       "https://api.twitter.com",
	}

	return &TwitterClient{
		client: client,
	}
}

// Post posts a repository to Twitter.
func (c *TwitterClient) Post(repo models.Repository) error {
	// Create a more creative message
	summary := ""
	if repo.Description.Valid {
		summary = repo.Description.String
	}

	link := fmt.Sprintf("https://app.gitfinder.dev/repository/%d", repo.ID)

	var hashtags []string
	for lang := range repo.Languages {
		hashtags = append(hashtags, "#"+strings.ReplaceAll(lang, " ", ""))
	}
	if len(repo.Topics) > 0 {
		for _, topic := range repo.Topics {
			// Limit to 4 hashtags to avoid being spammy
			if len(hashtags) >= 4 {
				break
			}
			// avoid long topics
			if len(topic) > 15 {
				continue
			}
			hashtags = append(hashtags, "#"+strings.ReplaceAll(topic, "-", ""))
		}
	}

	baseMessage := fmt.Sprintf("🚀 Trending on GitHub: %s\n\n%s\n\n🔗 %s\n\n%s",
		repo.FullName,
		"%s", // placeholder for summary
		link,
		strings.Join(hashtags, " "),
	)

	// 280 is the limit, but let's use 270 to be safe with URL shorteners and other twitter things
	remainingChars := 270 - len(fmt.Sprintf(baseMessage, ""))
	if len(summary) > remainingChars {
		summary = summary[:remainingChars-3] + "..."
	}

	message := fmt.Sprintf(baseMessage, summary)

	req := twitter.CreateTweetRequest{
		Text: message,
	}

	_, err := c.client.CreateTweet(context.Background(), req)
	if err != nil {
		var twitterErr *twitter.ErrorResponse
		if errors.As(err, &twitterErr) {
			return fmt.Errorf("twitter API error: %v", twitterErr.Errors)
		}
		return err
	}

	return nil
}