/*
TODO:
 - Add sad paths
*/
package bookmark_test

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/akruszewski/librarian/bookmark"
	"github.com/asdine/storm/v3"
	validator "github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
)

func Test_CanCreateBookmarkWithAllFilledFields(t *testing.T) {
	withTestStore(func(repo *bookmark.Store) {
		r := require.New(t)
		expectedBm := &bookmark.NewBookmark{
			Title: "test title",
			URL:   "https://test.com",
			Tags:  []string{"tag"},
			Notes: "test Note",
		}
		bm, err := repo.Add(context.Background(), expectedBm)
		r.NoError(err)
		r.NotNil(bm)
		r.Equal(expectedBm.Title, bm.Title)
		r.Equal(expectedBm.URL, bm.URL)
		r.Len(bm.Tags, len(expectedBm.Tags))
		for _, tag := range expectedBm.Tags {
			r.Contains(bm.Tags, tag)
		}
		r.Equal(expectedBm.Notes, bm.Notes)
	})
}

func Test_CanCreateBookmarkWithRequiredFields(t *testing.T) {
	withTestStore(func(repo *bookmark.Store) {
		r := require.New(t)
		expectedBm := &bookmark.NewBookmark{
			Title: "test title",
			URL:   "https://test.com",
		}
		bm, err := repo.Add(context.Background(), expectedBm)
		r.NoError(err)
		r.NotNil(bm)
		r.Equal(expectedBm.Title, bm.Title)
		r.Equal(expectedBm.URL, bm.URL)
	})
}

func Test_CannotCreateBookmarkWithoutRequiredFields(t *testing.T) {
	withTestStore(func(repo *bookmark.Store) {
		r := require.New(t)
		bm, err := repo.Add(context.Background(), &bookmark.NewBookmark{})
		r.Error(err)
		r.Nil(bm)

		var ve validator.ValidationErrors

		r.True(errors.As(err, &ve))

		r.Len(ve, 2)
	})
}

func Test_CanGetBookmark(t *testing.T) {
	withTestStore(func(repo *bookmark.Store) {
		r := require.New(t)
		expectedBm := &bookmark.NewBookmark{
			Title: "test title",
			URL:   "https://test.com",
			Tags:  []string{"tag"},
			Notes: "test Note",
		}
		bm, err := repo.Add(context.Background(), expectedBm)
		r.NoError(err)
		r.NotNil(bm)

		bm, err = repo.Get(context.Background(), bm.ID)
		r.NoError(err)
		r.NotNil(bm)

		r.Equal(expectedBm.Title, bm.Title)
		r.Equal(expectedBm.URL, bm.URL)
		r.Len(bm.Tags, len(expectedBm.Tags))
		for _, tag := range expectedBm.Tags {
			r.Contains(bm.Tags, tag)
		}
		r.Equal(expectedBm.Notes, bm.Notes)
	})
}

func Test_CannotGetBookmarkWhichDoesntExist(t *testing.T) {
	withTestStore(func(repo *bookmark.Store) {
		r := require.New(t)
		bm, err := repo.Get(context.Background(), 1)
		r.Error(err)
		r.Nil(bm)
		r.Equal(bookmark.ErrNotFound, err)
	})
}

func Test_CanGetBookmarkByURL(t *testing.T) {
	withTestStore(func(repo *bookmark.Store) {
		r := require.New(t)
		expectedBm := &bookmark.NewBookmark{
			Title: "test title",
			URL:   "https://test.com",
			Tags:  []string{"tag"},
			Notes: "test Note",
		}
		bm, err := repo.Add(context.Background(), expectedBm)
		r.NoError(err)
		r.NotNil(bm)

		bm, err = repo.GetByURL(context.Background(), bm.URL)
		r.NoError(err)
		r.NotNil(bm)

		r.Equal(expectedBm.Title, bm.Title)
		r.Equal(expectedBm.URL, bm.URL)
		r.Len(bm.Tags, len(expectedBm.Tags))
		for _, tag := range expectedBm.Tags {
			r.Contains(bm.Tags, tag)
		}
		r.Equal(expectedBm.Notes, bm.Notes)
	})
}

func Test_CannotGetBookmarkByURLWhichDoesntExist(t *testing.T) {
	withTestStore(func(repo *bookmark.Store) {
		r := require.New(t)

		bm, err := repo.GetByURL(context.Background(), "http://null.com")
		r.Error(err)
		r.Nil(bm)

		r.Equal(bookmark.ErrNotFound, err)
	})
}

func Test_CanUpdateBookmark(t *testing.T) {
	withTestStore(func(repo *bookmark.Store) {
		r := require.New(t)

		expectedBm, err := repo.Add(context.Background(), &bookmark.NewBookmark{
			Title: "test title",
			URL:   "https://test.com",
			Tags:  []string{"tag"},
			Notes: "test Note",
		})
		r.NoError(err)
		r.NotNil(expectedBm)

		expectedBm.Title = "test title 2"
		expectedBm.URL = "http://test.com"
		expectedBm.Tags = []string{"tag2", "tag3"}

		bm, err := repo.Update(context.Background(), expectedBm)
		r.NoError(err)
		r.NotNil(bm)

		r.Equal(expectedBm.Title, bm.Title)
		r.Equal(expectedBm.URL, bm.URL)
		r.Len(bm.Tags, len(expectedBm.Tags))
		for _, tag := range expectedBm.Tags {
			r.Contains(bm.Tags, tag)
		}
		r.Equal(expectedBm.Notes, bm.Notes)
	})
}

func Test_CannotUpdateBookmarkWithInvalidData(t *testing.T) {
	withTestStore(func(repo *bookmark.Store) {
		r := require.New(t)

		expectedBm, err := repo.Add(context.Background(), &bookmark.NewBookmark{
			Title: "test title",
			URL:   "https://test.com",
			Tags:  []string{"tag"},
			Notes: "test Note",
		})
		r.NoError(err)
		r.NotNil(expectedBm)

		expectedBm.Title = ""
		expectedBm.URL = ""
		expectedBm.Tags = []string{"tag2", "tag3"}

		bm, err := repo.Update(context.Background(), expectedBm)
		r.Error(err)
		r.Nil(bm)

		var ve validator.ValidationErrors
		r.True(errors.As(err, &ve))
		r.Len(ve, 2)

	})
}

func Test_CannotUpdateBookmarkWhichDoesntExist(t *testing.T) {
	withTestStore(func(repo *bookmark.Store) {
		r := require.New(t)

		bm, err := repo.Update(context.Background(), &bookmark.Bookmark{
			ID:    1,
			Title: "asd",
			URL:   "http://asd.com",
		})

		r.Error(err)
		r.Nil(bm)
		r.Equal(bookmark.ErrNotFound, err)
	})
}

func Test_CanDeleteBookmark(t *testing.T) {
	withTestStore(func(repo *bookmark.Store) {
		r := require.New(t)
		expectedBm := &bookmark.NewBookmark{
			Title: "test title",
			URL:   "https://test.com",
			Tags:  []string{"tag"},
			Notes: "test Note",
		}
		bm, err := repo.Add(context.Background(), expectedBm)
		r.NoError(err)
		r.NotNil(bm)

		err = repo.Delete(context.Background(), bm.ID)
		r.NoError(err)

		bm, err = repo.Get(context.Background(), bm.ID)
		r.Nil(bm)
		r.Error(err)
		r.Equal(bookmark.ErrNotFound, err)
	})
}

func Test_CannotDeleteBookmarkWHichDoesntExist(t *testing.T) {
	withTestStore(func(repo *bookmark.Store) {
		r := require.New(t)

		err := repo.Delete(context.Background(), 1)
		r.Error(err)
		r.Equal(bookmark.ErrNotFound, err)
	})
}

func Test_CanListBookmark(t *testing.T) {
	withTestStore(func(repo *bookmark.Store) {
		r := require.New(t)
		expectedBms := []*bookmark.NewBookmark{{
			Title: "test title",
			URL:   "https://test.com",
			Tags:  []string{"tag"},
			Notes: "test Note",
		}}
		for _, expectedBm := range expectedBms {
			bm, err := repo.Add(context.Background(), expectedBm)
			r.NoError(err)
			r.NotNil(bm)
		}

		bms, err := repo.List(context.Background())
		r.NoError(err)
		r.NotNil(bms)
		r.Len(bms, len(expectedBms))
	})
}

func Test_CanImportCSV(t *testing.T) {
	withTestStore(func(repo *bookmark.Store) {
		r := require.New(t)

		testCSV := `title|url|tags|notes|document|created_at|updated_at
test title|https://test.com|tag|test Note||2020-03-04T18:23:43Z|2020-03-04T18:23:43Z
`
		expectedBms := []*bookmark.NewBookmark{{
			Title: "test title",
			URL:   "https://test.com",
			Tags:  []string{"tag"},
		}}

		err := repo.ImportCSV(context.Background(), strings.NewReader(testCSV))
		r.NoError(err)

		bms, err := repo.List(context.Background())
		r.NoError(err)
		r.NotNil(bms)
		r.Len(bms, 1)

		r.Equal(expectedBms[0].Title, bms[0].Title)
		r.Equal(expectedBms[0].URL, bms[0].URL)
		r.Equal(expectedBms[0].Tags[0], bms[0].Tags[0])
	})
}

func withTestStore(f func(repo *bookmark.Store)) {
	dbFile, err := ioutil.TempFile("", "librarian_*.db")
	if err != nil {
		log.Fatalf("cannot create temp database file: %s", err)
	}
	dbPath := dbFile.Name()
	if err := dbFile.Close(); err != nil {
		log.Fatalf("cannot close temp database file: %s", err)
	}

	db, err := storm.Open(dbPath)
	if err != nil {
		log.Fatalf("cannot open temp database: %s", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	repo := bookmark.NewStore(db)
	f(repo)

}
