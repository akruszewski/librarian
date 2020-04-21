package cli

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/akruszewski/librarian/bookmark"
	librarianHttp "github.com/akruszewski/librarian/http"
	"github.com/asdine/storm/v3"
	"github.com/urfave/cli/v2"
)

func NewApp() (*cli.App, error) {

	client, err := librarianHttp.NewClient("http://127.0.0.1:8080/bookmark", 10*time.Second)
	if err != nil {
		return nil, err
	}

	return &cli.App{
		Name:  "librarian",
		Usage: "librarian is a bookmark manager application",
		Commands: []*cli.Command{
			{
				Name:    "serve",
				Usage:   "start librarian service",
				Aliases: []string{"s"},
				Action:  serveHandler,
			},
			{
				Name:   "import",
				Usage:  "import bookmarks from CSV file",
				Action: importCSVHandler,
			},
			{
				Name:      "add",
				Usage:     "add bookmark",
				Aliases:   []string{"a"},
				ArgsUsage: "<URL>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "title, t",
						Value: "",
						Usage: "title of the bookmark",
					},
					&cli.StringFlag{
						Name:  "tags",
						Value: "",
						Usage: "tags of the bookmark",
					},
					&cli.StringFlag{
						Name:  "note, n",
						Value: "",
						Usage: "notes to the bookmark",
					},
				},
				Action: addHandler(client),
			},
			{
				Name:      "get",
				Usage:     "get bookmark",
				Aliases:   []string{"g"},
				ArgsUsage: "<ID>",
				Action:    getHandler(client),
			},
			{
				Name:      "update",
				Usage:     "update bookmark",
				Aliases:   []string{"u", "up"},
				ArgsUsage: "<URL>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "title, t",
						Value: "NOT_UPDATED",
						Usage: "title of the bookmark",
					},
					&cli.StringFlag{
						Name:  "tags",
						Value: "NOT_UPDATED",
						Usage: "tags of the bookmark",
					},
					&cli.StringFlag{
						Name:  "note, n",
						Value: "NOT_UPDATED",
						Usage: "notes to the bookmark",
					},
				},
				Action: updateHandler(client),
			},
			{
				Name:      "delete",
				Usage:     "delete bookmark",
				Aliases:   []string{"d", "del"},
				ArgsUsage: "<ID>",
				Action:    deleteHandler(client),
			},
			{
				Name:    "list",
				Usage:   "lists all bookmarks",
				Aliases: []string{"l"},
				Action:  listHandler(client),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "fields",
						Value: "id;title;url;tags;created_at;updated_at",
						Usage: "fields which will be displayed",
					},
				},
			},
		},
	}, nil
}

func serveHandler(c *cli.Context) error {
	//TODO; db string from config
	db, err := storm.Open("data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	repo := bookmark.NewStore(db)
	handler := librarianHttp.Handler(context.Background(), repo)
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}

	return nil
}

func getHandler(client *librarianHttp.Client) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		if c.NArg() != 1 {
			return errors.New("ID argument required")
		}

		bm, err := client.Get(c.Args().First())
		if err != nil {
			return err
		}

		fmt.Printf(`ID: %d
Title:     %s
URL:       %s
Tags:      %s
CreateAt:  %s
UpdatedAt: %s
Notes:
	%s
`, bm.ID, bm.Title, bm.URL, bm.Tags, bm.CreatedAt, bm.UpdatedAt, bm.Notes)

		return nil
	}
}

func addHandler(client *librarianHttp.Client) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		if c.NArg() != 1 {
			return errors.New("URL argument required")
		}

		bm, err := client.Add(&bookmark.NewBookmark{
			Title: c.String("title"),
			URL:   c.Args().First(),
			Tags:  strings.Split(c.String("tags"), ";"),
			Notes: c.String("note"),
		})
		if err != nil {
			return err
		}

		fmt.Printf(`ID: %d
Title:     %s
URL:       %s
Tags:      %s
CreateAt:  %s
UpdatedAt: %s
Notes:
	%s
`, bm.ID, bm.Title, bm.URL, bm.Tags, bm.CreatedAt, bm.UpdatedAt, bm.Notes)
		return nil
	}

}

func updateHandler(client *librarianHttp.Client) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		if c.NArg() != 1 {
			return errors.New("ID argument required")
		}
		bm, err := client.Update(&bookmark.Bookmark{
			Title:    c.String("title"),
			URL:      c.Args().First(),
			Tags:     strings.Split(c.String("tags"), ";"),
			Notes:    c.String("note"),
			Document: ".",
		})
		if err != nil {
			return err
		}

		fmt.Printf(`ID: %d
Title:     %s
URL:       %s
Tags:      %s
CreateAt:  %s
UpdatedAt: %s
Notes:
	%s
`, bm.ID, bm.Title, bm.URL, bm.Tags, bm.CreatedAt, bm.UpdatedAt, bm.Notes)
		return nil
	}
}

func deleteHandler(client *librarianHttp.Client) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		if c.NArg() != 1 {
			return errors.New("ID argument required")
		}

		err := client.Delete(c.Args().First())
		if err != nil {
			return err
		}
		return nil
	}
}

func listHandler(client *librarianHttp.Client) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		bms, err := client.List()
		if err != nil {
			return err
		}

		//TODO: add fields validation
		fields := c.String("fields")
		for _, bm := range bms {

			if strings.Contains(fields, "ID") {
				fmt.Printf("%d\t", bm.ID)
			}
			if strings.Contains(fields, "title") {
				fmt.Printf("%s\t", bm.Title)
			}
			if strings.Contains(fields, "tags") {
				fmt.Printf("%s\t", strings.Join(bm.Tags, ","))
			}
			if strings.Contains(fields, "created_at") {
				fmt.Printf("%s\t", bm.UpdatedAt)
			}
			if strings.Contains(fields, "updated_at") {
				fmt.Printf("%s", bm.CreatedAt)
			}
			if strings.Contains(fields, "url") {
				fmt.Printf("%s\t", bm.URL)
			}
			fmt.Printf("\n")
		}
		return nil
	}
}

func importCSVHandler(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("URL argument required")
	}
	//TODO; db string from config
	db, err := storm.Open("data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	repo := bookmark.NewStore(db)

	fPath := c.Args().First()
	f, err := os.Open(fPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return repo.ImportCSV(context.Background(), f)
}
