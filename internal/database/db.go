package database

import (
	"fmt"
	"log"
	"os"

	custompacks "github.com/operaodev/cardex/internal/custom_packs"
	"github.com/operaodev/cardex/internal/products"
	"github.com/operaodev/cardex/internal/stock"
	"github.com/operaodev/cardex/internal/users"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", ""),
		getEnv("DB_NAME", "postgres"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_SSLMODE", "require"),
		getEnv("DB_TIMEZONE", "UTC"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})

	if err != nil {
		log.Fatalf("Error conectando a la base de datos: %v", err)
	}

	if err = db.AutoMigrate(
		&products.Product{},
		&users.User{},
		&stock.Stock{},
		&stock.Log{},
		&custompacks.Wishlist{},
	); err != nil {
		log.Fatalf("Error en automigración: %v", err)
	}

	// Partial unique: solo emails no vacíos son únicos (invitados tienen email='')
	db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uni_users_email ON users(email) WHERE email <> ''`)

	log.Println("Conectado a PostgreSQL y base de datos migrada con éxito")
	DB = db
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
