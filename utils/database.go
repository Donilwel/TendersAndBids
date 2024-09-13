package utils

import (
	"fmt"
	"log"
	"os"
	"testAvito/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_USERNAME"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DATABASE"),
		os.Getenv("POSTGRES_PORT"),
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	if err = DB.AutoMigrate(
		&models.Tender{},
		&models.TenderVersion{},
		&models.BidVersion{},
		&models.Bid{},
		&models.Employee{},
		&models.BidFeedback{},
		&models.BidDecision{},
		&models.Organization{},
	); err != nil {
		log.Println("Ошибка миграции базы данных", err.Error())
		return
	}

}
