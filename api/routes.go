package main

import (
	"github.com/epchao/millionaire-tracker/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupRoutes(app *fiber.App) {
	app.Get("/", handlers.Visualizer)
	app.Get("/analysis", handlers.Analysis)
}
