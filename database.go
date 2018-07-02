package main

import (
	"fmt"
	"log"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

// CreateConnection to Postgresql
func CreateConnection() *gorm.DB {

	db, err := gorm.Open(
		"postgres",
		fmt.Sprintf(
			"host=%s user=%s dbname=%s port=%s sslmode=disable password=%s",
			dbHost, dbUser, dbName, dbPort, dbPassword,
		),
	)
	if err != nil {
		log.Panic("Connection Error to Postgres: ", err)
	}
	return db
}
