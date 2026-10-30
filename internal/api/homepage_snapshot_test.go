package api

import (
	"testing"
	"time"

	"github.com/teomiscia/github-trending/internal/models"
)

func TestHomepageSnapshotPage(t *testing.T) {
	repositories := make([]homepageRepository, 120)
	for i := range repositories {
		repositories[i].Repository.ID = i + 1
	}

	service := &homepageSnapshotService{}
	service.current.Store(&homepageSnapshot{
		BuiltAt:      time.Now(),
		Repositories: repositories,
	})

	tests := []struct {
		name        string
		page        int
		wantOK      bool
		wantLen     int
		wantFirstID int
	}{
		{name: "first page", page: 0, wantOK: true, wantLen: 50, wantFirstID: 1},
		{name: "second page", page: 1, wantOK: true, wantLen: 50, wantFirstID: 51},
		{name: "partial last page", page: 2, wantOK: true, wantLen: 20, wantFirstID: 101},
		{name: "past the end", page: 3, wantOK: false},
		{name: "negative page", page: -1, wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repos, ok := service.Page(tc.page)
			if ok != tc.wantOK {
				t.Fatalf("Page(%d) ok = %v, want %v", tc.page, ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if len(repos) != tc.wantLen {
				t.Fatalf("Page(%d) returned %d repositories, want %d", tc.page, len(repos), tc.wantLen)
			}
			if repos[0].Repository.ID != tc.wantFirstID {
				t.Fatalf("Page(%d) first repository ID = %d, want %d", tc.page, repos[0].Repository.ID, tc.wantFirstID)
			}
		})
	}
}

func TestHomepageSnapshotPageWithoutSnapshot(t *testing.T) {
	service := &homepageSnapshotService{}

	if _, ok := service.Page(0); ok {
		t.Fatal("Page(0) should not succeed when no snapshot has been built")
	}
}

func TestHomepageSnapshotEncodeDecode(t *testing.T) {
	original := &homepageSnapshot{
		BuiltAt: time.Now().Truncate(time.Second),
		Repositories: []homepageRepository{
			{
				Repository: models.Repository{ID: 42, FullName: "owner/repo"},
				Owner:      models.Owner{Login: "owner"},
				Stats:      &models.RepositoryStat{StargazersCount: 1234},
			},
		},
	}

	encoded, err := encodeCompressedHomepageSnapshot(original)
	if err != nil {
		t.Fatalf("encodeCompressedHomepageSnapshot() error = %v", err)
	}

	decoded, err := decodeCompressedHomepageSnapshot(encoded)
	if err != nil {
		t.Fatalf("decodeCompressedHomepageSnapshot() error = %v", err)
	}

	if !decoded.BuiltAt.Equal(original.BuiltAt) {
		t.Fatalf("BuiltAt = %v, want %v", decoded.BuiltAt, original.BuiltAt)
	}
	if len(decoded.Repositories) != 1 {
		t.Fatalf("got %d repositories, want 1", len(decoded.Repositories))
	}
	if decoded.Repositories[0].Repository.ID != 42 {
		t.Fatalf("repository ID = %d, want 42", decoded.Repositories[0].Repository.ID)
	}
	if decoded.Repositories[0].Stats == nil || decoded.Repositories[0].Stats.StargazersCount != 1234 {
		t.Fatalf("stats were not preserved: %+v", decoded.Repositories[0].Stats)
	}
}
