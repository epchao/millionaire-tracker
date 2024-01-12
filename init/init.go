package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kkdai/youtube/v2"
	_ "github.com/lib/pq"
	"github.com/otiai10/gosseract/v2"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"gocv.io/x/gocv"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "password"
	dbname   = "millionaire_tracker"
)

// PORT TO global database supported code (dockerized)

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
	defer db.Close()

	channelId := "UC1htp5BzPQ6ScCL6VpepuvA"
	apiUrl := "https://yt0.lemnoslife.com/channels?part=shorts&id=" + channelId
	shorts, pageToken, err := getShorts(apiUrl)
	throwError(err)
	for _, short := range shorts {
		err = insertShort(db, short)
		throwError(err)
	}

	for len(pageToken) > 0 {
		apiUrl = "https://yt0.lemnoslife.com/channels?part=shorts&id=" + channelId + "&pageToken=" + pageToken
		newShorts, newPageToken, err := getShorts(apiUrl)
		throwError(err)
		for _, short := range newShorts {
			err = insertShort(db, short)
			throwError(err)
		}
		pageToken = newPageToken
	}
}

// //////////////
//
//	UPDATE DB  //
//
// //////////////
// NOT TESTED
func updateVideo() {
	fmt.Println("Initializing PostgreSQL database")
	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlconn)
	throwError(err)
	defer db.Close()

	channelId := "UC1htp5BzPQ6ScCL6VpepuvA"
	apiUrl := "https://yt0.lemnoslife.com/channels?part=shorts&id=" + channelId
	shorts, _, err := getShorts(apiUrl)
	throwError(err)
	for _, short := range shorts {
		result, err := isShortInDB(db, short)
		throwError(err)
		if result {
			break // we've hit a point where there is a short in the database, so all next ones will be in the db
		} else {
			err = insertShort(db, short)
			throwError(err)
		}
	}

}

///////////////////
//  PARSE VIDEO  //
///////////////////

func extractIncome(videoID string) (text string, err error) {
	URL := getVideoData(videoID)
	imagePath, err := downloadLastFrame(videoID, URL)
	throwError(err)
	preProcessImage(imagePath)
	text, err = applyOCR(imagePath)
	throwError(err)
	return text, nil
}

func getVideoData(videoID string) (URL string) {
	fmt.Println("Downloading:", videoID)
	yt := youtube.Client{}
	video, err := yt.GetVideoContext(context.Background(), videoID)
	throwError(err)
	format := video.Formats[0] // best quality video is always first (easiest for us to parse)
	return format.URL
}

func downloadLastFrame(videoID string, URL string) (imagePath string, err error) {
	imagePath = "./out/" + videoID + ".png"
	err = ffmpeg.Input(URL).
		Filter("reverse", ffmpeg.Args{}).
		Output(imagePath, ffmpeg.KwArgs{"frames:v": 1}).
		OverWriteOutput().
		ErrorToStdOut().
		Run()
	throwError(err)
	return imagePath, err
}

func preProcessImage(imagePath string) {
	img := gocv.IMRead(imagePath, gocv.IMReadGrayScale)
	defer img.Close()

	rm := img.Clone()
	defer rm.Close()

	white := gocv.NewMatWithSize(rm.Rows(), rm.Cols(), rm.Type())
	defer white.Close()
	scalar := gocv.NewScalar(255.0, 255.0, 255.0, 255.0)
	white.SetTo(scalar)
	gocv.Subtract(white, rm, &rm)

	// biniarization
	threshImg := gocv.NewMat()
	defer threshImg.Close()
	gocv.Threshold(img, &threshImg, 0, 255, gocv.ThresholdOtsu+gocv.ThresholdBinary)

	// skeleton
	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Pt(2, 2))
	defer kernel.Close()

	// thinning
	dilateImg := gocv.NewMat()
	defer dilateImg.Close()
	gocv.MorphologyExWithParams(rm, &dilateImg, gocv.MorphErode, kernel, 1, gocv.BorderConstant)

	preResult := gocv.NewMat()
	defer preResult.Close()
	gocv.CvtColor(threshImg, &threshImg, gocv.ColorGrayToBGR)
	gocv.CvtColor(dilateImg, &dilateImg, gocv.ColorGrayToBGR)
	gocv.BitwiseAnd(dilateImg, threshImg, &preResult)

	result := gocv.NewMat()
	defer result.Close()
	gocv.MorphologyExWithParams(preResult, &result, gocv.MorphOpen, kernel, 2, gocv.BorderConstant)

	// denoising
	final := gocv.NewMat()
	defer final.Close()
	gocv.BitwiseAnd(result, threshImg, &final)

	// optimize OCR by setting image to be black text and white bg
	gocv.CvtColor(final, &final, gocv.ColorBGRToGray)
	threshFinalImg := gocv.NewMat()
	defer threshFinalImg.Close()
	gocv.Threshold(final, &threshFinalImg, 0, 255, gocv.ThresholdOtsu+gocv.ThresholdBinary)

	whiteFinal := gocv.NewMatWithSize(threshFinalImg.Rows(), threshFinalImg.Cols(), threshFinalImg.Type())
	defer whiteFinal.Close()
	whiteFinal.SetTo(scalar)
	gocv.Subtract(whiteFinal, threshFinalImg, &threshFinalImg)

	gocv.IMWrite(imagePath, threshFinalImg)
}

func applyOCR(imagePath string) (text string, err error) {
	ocr := gosseract.NewClient()
	defer ocr.Close()
	err = ocr.SetImage(imagePath)
	throwError(err)
	text, err = ocr.Text()
	throwError(err)
	return text, nil
}

//////////////////
//  LEMNOS API  //
//////////////////

func getShorts(apiUrl string) (shorts []Short, pageToken string, err error) {
	fmt.Println("Querying:", apiUrl)
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

// NOT TESTED
func isShortInDB(db *sql.DB, short Short) (found bool, err error) {
	fmt.Println("Pulling", short.VideoID, "from the database")

	if strings.Contains(short.Title, "#millionaireinthemaking") || isDate(short.Title) || short.Title == "#millionareinthemaking" {
		getShortFromDB := `SELECT * FROM public."Shorts" WHERE "VideoID" = $1`
		shortInDB, err := db.Query(getShortFromDB, short.VideoID)
		throwError(err)

		return shortInDB.Next(), nil
	}
	return false, nil
}

func insertShort(db *sql.DB, short Short) (err error) {
	fmt.Println("Adding", short.VideoID, short.Title, "to DB.")
	text, err := extractIncome(short.VideoID)
	throwError(err)
	if strings.Contains(short.Title, "#millionaireinthemaking") || isDate(short.Title) || short.Title == "#millionareinthemaking" {
		// revenue := "-123456789"
		// revenueCheck := regexp.MustCompile(`\+\s\$(\d+)`)
		// revenueMatch := revenueCheck.FindStringSubmatch(text)
		// if len(revenueMatch) > 1 {
		// 	revenue = revenueMatch[1]
		// }
		// revenueNum, err := strconv.Atoi(revenue)
		// throwError(err)
		// expenses := "-123456789"
		// expensesCheck := regexp.MustCompile(`\-\s\$(\d+)`)
		// expensesMatch := expensesCheck.FindStringSubmatch(text)
		// if len(expensesMatch) > 1 {
		// 	expenses = expensesMatch[1]
		// }
		// expensesNum, err := strconv.Atoi(expenses)
		// throwError(err)

		title, err := verifyNumberData(short.Title, "title")
		throwError(err)
		revenue, err := verifyNumberData(text, "revenue")
		throwError(err)
		expenses, err := verifyNumberData(text, "expenses")
		throwError(err)
		insertShort := `insert into "Shorts" ("VideoID", "Title", "Revenue", "Expenses", "NetResult") 
			values ($1, $2, $3, $4, $5)`
		_, err = db.Exec(insertShort, short.VideoID, title, revenue, expenses, revenue-expenses)
		throwError(err)
	}
	return nil
}

/////////////
//  UTILS  //
/////////////

// NOT TESTED
func verifyNumberData(text string, dataType string) (num int, err error) {
	num = -123456789
	check := regexp.MustCompile(``)
	switch dataType {
	case "revenue":
		check = regexp.MustCompile(`\+\s\$(\d+)` + `\(revenue\)`)
	case "expenses":
		check = regexp.MustCompile(`\-\s\$(\d+)` + `\(expenses\)`)
	case "title":
		check = regexp.MustCompile(`Day` + `(\d+)` + `#millionaireinthemaking`)
	}
	match := check.FindStringSubmatch(text)
	if len(match) > 1 {
		num, err = strconv.Atoi(match[1])
		throwError(err)
	}
	return num, nil
}

func isDate(str string) bool {
	layout := "January 2, 2006"
	_, err := time.Parse(layout, str)
	return err == nil
}

func throwError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
