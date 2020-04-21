/*
TODO:
 - add proper http status handling.
*/
package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/akruszewski/librarian/bookmark"
	validator "github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type bookmarkHandler struct {
	repo bookmark.Storager
	log  *log.Entry
}

func Handler(ctx context.Context, repo bookmark.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//Add unique request id to context
		reqID := uuid.New()
		ctx = context.WithValue(ctx, "ReqID", reqID)
		log := log.New().WithFields(log.Fields{"ReqID": reqID})
		var head string
		head, r.URL.Path = ShiftPath(r.URL.Path)
		if head == "bookmark" {
			BookmarkHandler(ctx, repo, log)(w, r)
			return
		}
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

//NewBookmarkRouter returns bookmark router
func BookmarkHandler(ctx context.Context, repo bookmark.Storager, log *log.Entry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bh := bookmarkHandler{repo: repo, log: log}
		if r.URL.Path == "/" {
			switch r.Method {
			case http.MethodGet:
				bh.listBookmarkHandler(ctx, w, r)
			case http.MethodPost:
				bh.createBookmarkHandler(ctx, w, r)
			}
			return
		}
		var head string
		head, r.URL.Path = ShiftPath(r.URL.Path)
		id, err := strconv.Atoi(head)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid user id %q", head), http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			bh.getBookmarkHandler(ctx, w, r, id)
		case http.MethodDelete:
			bh.deleteBookmarkHandler(ctx, w, r, id)
		case http.MethodPost:
			bh.updateBookmarkHandler(ctx, w, r, id)
		}
	}
}

// ShiftPath splits off the first component of p, which will be cleaned of
// relative components before processing. head will never contain a slash and
// tail will always be a rooted path without trailing slash.
func ShiftPath(p string) (head, tail string) {
	p = path.Clean("/" + p)
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		return p[1:], "/"
	}
	return p[1:i], p[i:]
}

func (bh *bookmarkHandler) createBookmarkHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		bh.log.Errorf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	nbm := &bookmark.NewBookmark{}
	if err := json.Unmarshal(body, nbm); err != nil {
		bh.log.Errorf("Error unmarshaling body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}
	bm, err := bh.repo.Add(ctx, nbm)
	if err != nil {
		bh.log.Errorf("Error adding bookmark: %v", err)
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			http.Error(w, "internal error", http.StatusBadRequest)
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	bh.log.WithFields(log.Fields{"BookmarkID": bm.ID}).Info("Bookmark added to repository")
	data, err := json.Marshal(bm)
	if err != nil {
		bh.log.Errorf("Error marshaling bookmark: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if _, err = w.Write(data); err != nil {
		bh.log.Errorf("Error writing data: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (bh *bookmarkHandler) updateBookmarkHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, id int) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		bh.log.Errorf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	bm := &bookmark.Bookmark{}
	if err := json.Unmarshal(body, bm); err != nil {
		bh.log.Errorf("Error unmarshaling body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}
	//TODO: hmhm...
	bm.ID = id

	bm, err = bh.repo.Update(ctx, bm)
	if err != nil {
		bh.log.Errorf("Error adding bookmark: %v", err)
		if err == bookmark.ErrNotFound {
			http.Error(w, "{\"message\": \"bookmark not found\"}", http.StatusNotFound)
			return
		}
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			http.Error(w, "internal error", http.StatusBadRequest)
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	bh.log.WithFields(log.Fields{"BookmarkID": bm.ID}).Info("Bookmark updated.")

	data, err := json.Marshal(bm)
	if err != nil {
		bh.log.Errorf("Error marshaling bookmark: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if _, err = w.Write(data); err != nil {
		bh.log.Errorf("Error writing data: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (bh *bookmarkHandler) deleteBookmarkHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, id int) {
	if err := bh.repo.Delete(ctx, id); err != nil {
		bh.log.Errorf("Couldn't delete bookmark: %v", err)
		if err == bookmark.ErrNotFound {
			http.Error(w, "{\"message\": \"bookmark not found\"}", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	bh.log.WithFields(log.Fields{"BookmarkID": id}).Info("Bookmark deleted.")
}

func (bh *bookmarkHandler) getBookmarkHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, id int) {
	bm, err := bh.repo.Get(ctx, id)
	if err != nil {
		bh.log.Errorf("Error retrieving bookmark: %v", err)
		if err == bookmark.ErrNotFound {
			http.Error(w, "{\"message\": \"bookmark not found\"}", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(bm)
	if err != nil {
		bh.log.Errorf("Error marshaling bookmark: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if _, err = w.Write(data); err != nil {
		bh.log.Errorf("Error writing data: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	bh.log.WithFields(log.Fields{"BookmarkID": bm.ID}).Info("Bookmark retrieved.")
}

func (bh *bookmarkHandler) listBookmarkHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	bms, err := bh.repo.List(ctx)
	if err != nil {
		bh.log.Errorf("Error retrieving bookmarks: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(bms)
	if err != nil {
		bh.log.Errorf("Error marshaling bookmark: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if _, err = w.Write(data); err != nil {
		bh.log.Errorf("Error writing data: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	bh.log.Info("Bookmarks Listed.")
}
