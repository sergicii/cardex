package users

import (
	"time"
)

type User struct {
	ID             string    `json:"id" gorm:"primaryKey;type:uuid"`
	Email          string    `json:"email" gorm:"not null;unique"`
	HashedPassword string    `json:"-" gorm:"not null"`
	Name           string    `json:"name" gorm:"not null"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
