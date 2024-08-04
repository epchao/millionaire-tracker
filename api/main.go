package main

import (
	"fmt"
	"os"

	"github.com/epchao/millionaire-tracker/database"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

func main() {
	database.ConnectDb()
	engine := html.New("./views", ".html")
	app := fiber.New(fiber.Config{Views: engine})
	// cronJob := cron.New()
	setupRoutes(app)
	// @every 0h0m10s @weekly
	// cronJob.AddFunc("@every 0h0m10s", func() {
	// 	scripts.Update()
	// })
	// cronJob.Start()
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	app.Listen(fmt.Sprintf(":%s", port))
}
