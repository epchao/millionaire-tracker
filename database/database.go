package database

import (
	"fmt"
	"log"
	"os"

	"github.com/epchao/millionaire-tracker/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Dbinstance struct {
	Db *gorm.DB
}

var DB Dbinstance

func ConnectDb() {
	mode := os.Getenv("BUILD")
	DB_HOST := ""
	DB_USER := ""
	DB_PASSWORD := ""
	DB_NAME := ""

	if mode == "DEV" {
		DB_HOST = "db"
		DB_USER = os.Getenv("DEV_DB_USER")
		DB_PASSWORD = os.Getenv("DEV_DB_PASSWORD")
		DB_NAME = os.Getenv("DEV_DB_NAME")
	} else {
		DB_HOST = os.Getenv("DB_HOST")
		DB_USER = os.Getenv("DB_USER")
		DB_PASSWORD = os.Getenv("DB_PASSWORD")
		DB_NAME = os.Getenv("DB_NAME")
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable TimeZone=America/Los_Angeles",
		DB_HOST,
		DB_USER,
		DB_PASSWORD,
		DB_NAME,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Failed to connect to database. \n", err)
		os.Exit(2)
	}

	log.Println("Successfully connected to database.")
	db.Logger = logger.Default.LogMode(logger.Info)

	log.Println("Running migrations for Shorts")
	db.AutoMigrate(&models.Short{})

	DB = Dbinstance{
		Db: db,
	}
}
