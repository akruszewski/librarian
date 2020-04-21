//TODO:
// - write tests for repository
// - implement update method and query method
// - implement proper error handling
// - implement export/import methods from/to CSV
// - add woosh as search engine for advenced quering.
package bookmark

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"log"
	"strings"
	"time"

	"github.com/asdine/storm/v3"
	validator "github.com/go-playground/validator/v10"
)

const csvComma = '|'

var ErrNotFound = errors.New("bookmark not found")

var csvHeader = []string{
	"title",
	"url",
	"tags",
	"notes",
	"document",
	"created_at",
	"updated_at",
}

//NewBookmark represents new bookmark.
type NewBookmark struct {
	Title string   `json:"title" validate:"required"`
	URL   string   `json:"url" validate:"required"`
	Tags  []string `json:"tags"`
	Notes string   `json:"notes"`
}

//BookmarkSummary structure represents summary of bookmark.
type BookmarkSummary struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

//Bookmark structure represents single bookmark in repository. For now it repre
//sents just url object, but it's worth considering to be able to bookmark also
//other things, like terminal commands or similar entries. In that case type
//field should be introduced which would describe resource type (URL,CMD,OTHER).
//It's also worth considering some kind of rank, based on frequency of searches.
type Bookmark struct {
	ID    int      `json:"id" validate:"required" storm:"id,increment"`
	Title string   `json:"title" validate:"required" storm:"unique"`
	URL   string   `json:"url" validate:"required" storm:"unique"`
	Tags  []string `json:"tags" storm:"index"`
	Notes string   `json:"notes"`

	Document  string    `json:"document"`
	CreatedAt time.Time `json:"created_at" storm:"index"`
	UpdatedAt time.Time `json:"updated_at" storm:"index"`
}

type Storager interface {
	Add(context.Context, *NewBookmark) (*Bookmark, error)
	Update(context.Context, *Bookmark, ...string) (*Bookmark, error)
	Delete(context.Context, int) error
	Get(context.Context, int) (*Bookmark, error)
	GetByURL(context.Context, string) (*Bookmark, error)
	List(context.Context) ([]*BookmarkSummary, error)
	//TODO: List is not necessary, remove it.
	ImportCSV(context.Context, io.Reader) error
}

//Store structure represents bookmark repository.
type Store struct {
	db       *storm.DB
	validate *validator.Validate
}

//Add bookmark to repository.
func (r *Store) Add(ctx context.Context, nbm *NewBookmark) (*Bookmark, error) {
	if err := r.validate.Struct(nbm); err != nil {
		return nil, err
	}
	bm := &Bookmark{
		Title:     nbm.Title,
		URL:       nbm.URL,
		Tags:      nbm.Tags,
		Notes:     nbm.Notes,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := r.db.Save(bm); err != nil {
		return nil, err
	}
	return bm, nil
}

//Update updates bookmark in database, all fields passed in bookmark
//structure will be updated.
//TODO: partial update shpuld be possible. In this case, function should get
//optional arguments onlyFields ...string which contains arguments to update.
func (r *Store) Update(ctx context.Context, bm *Bookmark, onlyFields ...string) (*Bookmark, error) {
	if len(onlyFields) != 0 {
		return nil, errors.New("Not implemented")
	}
	if err := r.validate.Struct(bm); err != nil {
		return nil, err
	}
	if err := r.db.Update(bm); err != nil {
		if err == storm.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return bm, nil
}

//Delete bookmark from repository.
func (r *Store) Delete(ctx context.Context, id int) error {
	if err := r.db.DeleteStruct(&Bookmark{ID: id}); err != nil {
		if err == storm.ErrNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}

//Get retrieves bookmark from repository
func (r *Store) Get(ctx context.Context, id int) (*Bookmark, error) {
	bm := &Bookmark{}
	if err := r.db.One("ID", id, bm); err != nil {
		if err == storm.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return bm, nil
}

func (r *Store) GetByURL(ctx context.Context, url string) (*Bookmark, error) {
	bm := &Bookmark{}
	if err := r.db.One("URL", url, bm); err != nil {
		if err == storm.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return bm, nil
}

//List all bookmarks
func (r *Store) List(ctx context.Context) ([]*BookmarkSummary, error) {
	bms := []Bookmark{}
	if err := r.db.All(&bms); err != nil {
		return nil, err
	}
	bs := []*BookmarkSummary{}
	for _, bm := range bms {
		bs = append(bs, &BookmarkSummary{
			ID:        bm.ID,
			Title:     bm.Title,
			URL:       bm.URL,
			Tags:      bm.Tags,
			CreatedAt: bm.CreatedAt,
			UpdatedAt: bm.UpdatedAt,
		})
	}
	return bs, nil
}

func (rep *Store) ImportCSV(ctx context.Context, r io.Reader) error {
	csvReader := newCSVReader(r)

	header, err := csvReader.Read()
	if err != nil {
		return err
	}
	if !validateCSVHeader(header) {
		return errors.New("invalid csv file")
	}
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		bm, err := parseBookmark(record)
		if err != nil {
			return err
		}

		if err = rep.db.Save(bm); err != nil {
			return err
		}
		log.Printf("Bookmark %+v added to database", bm)
	}

	return nil
}

//Init inits bookmark repository.
func (r *Store) Init(ctx context.Context) error {
	if err := r.db.Init(&Bookmark{}); err != nil {
		return err
	}
	return nil
}

//NewStore initialisate repository structure with given database.
func NewStore(db *storm.DB) *Store {
	return &Store{
		db:       db,
		validate: validator.New(),
	}
}

func parseBookmark(data []string) (*Bookmark, error) {
	tags := strings.Split(data[2], ";")
	cr, err := time.Parse(time.RFC3339, data[5])
	if err != nil {
		return nil, err
	}
	up, err := time.Parse(time.RFC3339, data[6])
	if err != nil {
		return nil, err
	}
	return &Bookmark{
		Title:     data[0],
		URL:       data[1],
		Tags:      tags,
		Notes:     data[3],
		Document:  data[4],
		CreatedAt: cr,
		UpdatedAt: up,
	}, nil
}

func newCSVReader(r io.Reader) *csv.Reader {
	w := csv.NewReader(r)
	w.Comma = csvComma
	return w
}

func validateCSVHeader(header []string) bool {
	for i, h := range header {
		if h != csvHeader[i] {
			return false
		}
	}
	return true
}
