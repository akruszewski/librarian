/*
TODO:
 - add proper http response status handling.
*/
package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/akruszewski/librarian/bookmark"
)

type Client struct {
	url        *url.URL
	httpClient *http.Client
}

func (c *Client) Add(nbm *bookmark.NewBookmark) (*bookmark.Bookmark, error) {
	body, err := json.Marshal(nbm)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Post(
		c.url.String(),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	bm := &bookmark.Bookmark{}
	if err := json.Unmarshal(respBody, bm); err != nil {
		return nil, err
	}
	return bm, nil
}

func (c *Client) Update(bm *bookmark.Bookmark) (*bookmark.Bookmark, error) {
	body, err := json.Marshal(bm)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Post(
		buildURL(*c.url, strconv.Itoa(bm.ID)),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	bm = &bookmark.Bookmark{}
	if err := json.Unmarshal(respBody, bm); err != nil {
		return nil, err
	}
	return bm, nil
}

func (c *Client) Get(id string) (*bookmark.Bookmark, error) {
	resp, err := c.httpClient.Get(buildURL(*c.url, id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	bm := &bookmark.Bookmark{}
	if err := json.Unmarshal(body, bm); err != nil {
		return nil, err
	}
	return bm, nil
}

func (c *Client) Delete(id string) error {
	req, err := http.NewRequest(http.MethodDelete, buildURL(*c.url, id), nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got unexpected status: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) List() ([]bookmark.BookmarkSummary, error) {
	resp, err := c.httpClient.Get(c.url.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	bm := []bookmark.BookmarkSummary{}
	if err := json.Unmarshal(body, &bm); err != nil {
		return nil, err
	}
	return bm, nil
}

//NewClient instantiate Client. TODO: move args to application configuration
//structure.
func NewClient(URL string, timeout time.Duration) (*Client, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	return &Client{
		url:        u,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

func buildURL(u url.URL, p string, args ...string) string {
	u.Path = path.Join(u.Path, p)
	return u.String()
}
