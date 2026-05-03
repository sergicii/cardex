package database

import (
	"log"

	"github.com/operaodev/cardex/internal/cards"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	// DSN (Data Source Name): Reemplaza con tus credenciales de Postgres
	dns := `
	host=[IP_ADDRESS]
	user=root
	password=root
	dbname=cardex
	port=5432
	sslmode=disable
	TimeZone=America/Lima`

	db, err := gorm.Open(postgres.Open(dns), &gorm.Config{})
	if err != nil {
		log.Fatalf("Error conectando a la base de datos: %v", err)
	}

	// ¡MAGIA DE GORM! Automigración
	// Esto lee tu estructura 'Card' de internal/cards/model.go y crea o actualiza
	// la tabla 'cards' en PostgreSQL con todas las columnas, JSONBs, e índices.
	err = db.AutoMigrate(&cards.Card{})

	if err != nil {
		log.Fatalf("Error en automigración: %v", err)
	}

	log.Println("Conectado a PostgreSQL y base de datos migrada con éxito")

	DB = db
}
