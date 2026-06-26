package users

import (
	"time"
)

type User struct {
	ID                        string     `json:"id" gorm:"primaryKey;type:uuid"`
	Email                     string     `json:"email" gorm:"uniqueIndex:uni_users_email,where:email <> ''"`
	HashedPassword            string     `json:"-"`
	Name                      string     `json:"name"`
	IsGuest                   bool       `json:"is_guest" gorm:"default:false;not null"`
	EmailVerified             bool       `json:"email_verified" gorm:"default:false;not null"`
	VerificationCode          string     `json:"-" gorm:"type:varchar(6)"`
	VerificationCodeExpiresAt *time.Time `json:"-"`
	RefreshToken              string     `json:"-"`
	CreatedAt                 time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt                 time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}
