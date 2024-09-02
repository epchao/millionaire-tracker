package main

import (
	"fmt"
	"os"

	"github.com/epchao/millionaire-tracker/database"
	"github.com/epchao/millionaire-tracker/scripts"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/robfig/cron/v3"
)

func main() {
	database.ConnectDb()
	engine := html.New("./views", ".html")
	app := fiber.New(fiber.Config{Views: engine})
	cronJob := cron.New()

	cronJob.AddFunc("@weekly", func() {
		scripts.PopulateShortsEveryPage("http://yt.lemnoslife.com/channels?part=shorts&id=UC1htp5BzPQ6ScCL6VpepuvA")
	})
	cronJob.Start()

	setupRoutes(app)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	app.Listen(fmt.Sprintf("0.0.0.0:%s", port))
}
