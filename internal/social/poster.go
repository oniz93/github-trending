package social

import "github.com/teomiscia/github-trending/internal/models"

// Poster is an interface for posting to social media.
type Poster interface {
	Post(repo models.Repository) error
}
