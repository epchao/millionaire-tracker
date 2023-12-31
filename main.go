package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "password"
	dbname   = "millionaire_tracker"
)

type Message struct {
	Items []Item
}

type Item struct {
	Shorts        []Short
	NextPageToken string
}

type Short struct {
	VideoID string
	Title   string
}

func main() {
	fmt.Println("Initializing PostgreSQL database")
	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlconn)
	throwError(err)

	fmt.Println("Querying third-party YouTube API endpoint")
	channelId := "UC1htp5BzPQ6ScCL6VpepuvA"
	apiUrl := "https://yt.lemnoslife.com/channels?part=shorts&id=" + channelId

	shorts, pageToken, err := getShorts(apiUrl)
	throwError(err)

	for _, short := range shorts {
		err = insertShort(db, short)
		throwError(err)
	}

	for len(pageToken) > 0 {
		apiUrl = "https://yt.lemnoslife.com/channels?part=shorts&id=" + channelId + "&pageToken=" + pageToken

		newShorts, newPageToken, err := getShorts(apiUrl)
		throwError(err)

		for _, short := range newShorts {
			err = insertShort(db, short)
			throwError(err)
		}

		pageToken = newPageToken
	}

	defer db.Close()
}

func getShorts(apiUrl string) (shorts []Short, pageToken string, err error) {
	request, err := http.NewRequest("GET", apiUrl, nil)
	throwError(err)

	request.Header.Set("Content-Type", "application/json; charset=utf-8")

	client := &http.Client{}
	response, err := client.Do(request)
	throwError(err)

	responseBody, err := io.ReadAll(response.Body)
	throwError(err)

	var formattedData Message

	err = json.Unmarshal(responseBody, &formattedData)
	throwError(err)

	defer response.Body.Close()

	item := formattedData.Items[0]
	return item.Shorts, item.NextPageToken, nil
}

func insertShort(db *sql.DB, short Short) (err error) {
	insertShort := `insert into "Shorts" ("VideoID", "Title") values ($1, $2)`
	short.VideoID = "https://www.youtube.com/shorts/" + short.VideoID

	_, err = db.Exec(insertShort, short.VideoID, short.Title)
	throwError(err)

	return nil
}

func throwError(err error) {
	if err != nil {
		panic(err)
	}
}
