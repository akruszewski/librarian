package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/akruszewski/librarian/bookmark"
	librarianHttp "github.com/akruszewski/librarian/http"
	"github.com/asdine/storm/v3"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestAddBookmarkReturnOKCodeIfValidBookmarkWasAdded(t *testing.T) {
	withTestRepositoryLogAndContext(func(ctx context.Context, repo bookmark.Storager, log *log.Entry) {
		r := require.New(t)

		bm := bookmark.NewBookmark{Title: "Test", URL: "http://test.com"}

		data, err := json.Marshal(bm)
		r.NoError(err)

		req, err := http.NewRequest(http.MethodPost, "/", bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(librarianHttp.BookmarkHandler(ctx, repo, log))

		handler.ServeHTTP(rr, req)

		r.Equal(http.StatusOK, rr.Code)
	})
}

func Test_AddBookmarkReturnsStatusBadRequestWhenRequiredFieldsAreMissing(t *testing.T) {
	withTestRepositoryLogAndContext(func(ctx context.Context, repo bookmark.Storager, log *log.Entry) {
		r := require.New(t)

		data, err := json.Marshal(&bookmark.NewBookmark{})
		r.NoError(err)

		req, err := http.NewRequest(http.MethodPost, "/", bytes.NewReader(data))
		r.NoError(err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(librarianHttp.BookmarkHandler(ctx, repo, log))

		handler.ServeHTTP(rr, req)

		r.Equal(http.StatusBadRequest, rr.Code)
	})
}

func Test_CanGetBookmark(t *testing.T) {
	withTestRepositoryLogAndContext(func(ctx context.Context, repo bookmark.Storager, log *log.Entry) {
		r := require.New(t)

		nbm := bookmark.NewBookmark{Title: "Test", URL: "http://test.com"}

		bm, err := repo.Add(ctx, &nbm)
		r.NotNil(bm)
		r.NoError(err)

		req, err := http.NewRequest(http.MethodGet, "/1", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(librarianHttp.BookmarkHandler(ctx, repo, log))

		handler.ServeHTTP(rr, req)

		r.Equal(http.StatusOK, rr.Code)
	})
}

func Test_CannotGetBookmarkWhichDoesntExist(t *testing.T) {
	withTestRepositoryLogAndContext(func(ctx context.Context, repo bookmark.Storager, log *log.Entry) {
		r := require.New(t)

		req, err := http.NewRequest(http.MethodGet, "/1", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(librarianHttp.BookmarkHandler(ctx, repo, log))

		handler.ServeHTTP(rr, req)

		r.Equal(http.StatusNotFound, rr.Code)
	})
}

func Test_CanUpdateBookmark(t *testing.T) {
	withTestRepositoryLogAndContext(func(ctx context.Context, repo bookmark.Storager, log *log.Entry) {
		r := require.New(t)

		nbm := bookmark.NewBookmark{Title: "Test", URL: "http://test.com"}

		bm, err := repo.Add(ctx, &nbm)
		r.NotNil(bm)
		r.NoError(err)

		bm.Title = "Test test"

		data, err := json.Marshal(bm)
		r.NoError(err)

		req, err := http.NewRequest(
			http.MethodPost,
			fmt.Sprintf("/%d", bm.ID),
			bytes.NewReader(data),
		)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(librarianHttp.BookmarkHandler(ctx, repo, log))

		handler.ServeHTTP(rr, req)

		r.Equal(http.StatusOK, rr.Code)
	})
}

func Test_UpdateBookmarkReturnsStatusBadRequestWhenInvalidDataIsPassed(t *testing.T) {
	withTestRepositoryLogAndContext(func(ctx context.Context, repo bookmark.Storager, log *log.Entry) {
		r := require.New(t)

		nbm := bookmark.NewBookmark{Title: "Test", URL: "http://test.com"}

		bm, err := repo.Add(ctx, &nbm)
		r.NotNil(bm)
		r.NoError(err)

		bm.Title = ""

		data, err := json.Marshal(bm)
		r.NoError(err)

		req, err := http.NewRequest(
			http.MethodPost,
			fmt.Sprintf("/%d", bm.ID),
			bytes.NewReader(data),
		)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(librarianHttp.BookmarkHandler(ctx, repo, log))

		handler.ServeHTTP(rr, req)

		r.Equal(http.StatusBadRequest, rr.Code)
	})
}

func Test_CannotUpdateBookmarkWhichDoesntExist(t *testing.T) {
	withTestRepositoryLogAndContext(func(ctx context.Context, repo bookmark.Storager, log *log.Entry) {
		r := require.New(t)

		bm := bookmark.Bookmark{ID: 100, Title: "Test", URL: "http://test.com"}

		data, err := json.Marshal(bm)
		r.NoError(err)

		req, err := http.NewRequest(
			http.MethodPost,
			fmt.Sprintf("/%d", bm.ID),
			bytes.NewReader(data),
		)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(librarianHttp.BookmarkHandler(ctx, repo, log))

		handler.ServeHTTP(rr, req)

		r.Equal(http.StatusNotFound, rr.Code)
	})
}

func Test_CanDeleteBookmark(t *testing.T) {
	withTestRepositoryLogAndContext(func(ctx context.Context, repo bookmark.Storager, log *log.Entry) {
		r := require.New(t)

		nbm := bookmark.NewBookmark{Title: "Test", URL: "http://test.com"}

		bm, err := repo.Add(ctx, &nbm)
		r.NotNil(bm)
		r.NoError(err)

		req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("/%d", bm.ID), nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(librarianHttp.BookmarkHandler(ctx, repo, log))

		handler.ServeHTTP(rr, req)

		r.Equal(http.StatusOK, rr.Code)
	})
}

func Test_CannotDeleteBookmarkWhichDoesntExist(t *testing.T) {
	withTestRepositoryLogAndContext(func(ctx context.Context, repo bookmark.Storager, log *log.Entry) {
		r := require.New(t)

		req, err := http.NewRequest(http.MethodDelete, "/1", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(librarianHttp.BookmarkHandler(ctx, repo, log))

		handler.ServeHTTP(rr, req)

		r.Equal(http.StatusNotFound, rr.Code)
	})
}

func Test_CanListBookmark(t *testing.T) {
	withTestRepositoryLogAndContext(func(ctx context.Context, repo bookmark.Storager, log *log.Entry) {
		r := require.New(t)

		nbm := bookmark.NewBookmark{Title: "Test", URL: "http://test.com"}

		bm, err := repo.Add(ctx, &nbm)
		r.NotNil(bm)
		r.NoError(err)

		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(librarianHttp.BookmarkHandler(ctx, repo, log))

		handler.ServeHTTP(rr, req)

		r.Equal(http.StatusOK, rr.Code)
	})
}

func withTestRepositoryLogAndContext(f func(ctx context.Context, repo bookmark.Storager, log *log.Entry)) {
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

	//Add unique request id to context
	reqID := uuid.New()
	ctx := context.WithValue(context.Background(), "ReqID", reqID)
	log := log.New().WithFields(log.Fields{"ReqID": reqID})

	repo := bookmark.NewStore(db)
	f(ctx, repo, log)
}
