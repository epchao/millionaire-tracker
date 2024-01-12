package handlers

import (
	"github.com/epchao/millionaire-tracker/database"
	"github.com/epchao/millionaire-tracker/models"
	"github.com/gofiber/fiber/v2"
)

func ListShorts(c *fiber.Ctx) error {
	shorts := []models.Short{}
	database.DB.Db.Find(&shorts)

	return c.Status(200).JSON(shorts)
}
