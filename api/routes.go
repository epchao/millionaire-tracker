package main

import (
	"github.com/epchao/millionaire-tracker/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	// _ "github.com/epchao/millionaire-tracker.git/docs"
)

// @title			Millionaire Tracker API
// @version		1.0
// @description	All information available on https://github.com/epchao/millionaire-tracker
// @BasePath		/
func setupRoutes(app *fiber.App) {
	app.Get("/", handlers.ListShorts)
	app.Get("/swagger/*", swagger.HandlerDefault)
}
