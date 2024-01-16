package main

import (
	"github.com/epchao/millionaire-tracker/database"
	"github.com/epchao/millionaire-tracker/scripts"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/robfig/cron"
)

func main() {
	database.ConnectDb()
	engine := html.New("./views", ".html")
	app := fiber.New(fiber.Config{Views: engine})
	cronJob := cron.New()
	setupRoutes(app)
	// @every 0h0m10s @weekly
	cronJob.AddFunc("@weekly", func() {
		scripts.Update()
	})
	cronJob.Start()
	app.Listen(":3000")
}
