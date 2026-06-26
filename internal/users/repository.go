package users

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	Create(user *User) error
	FindByEmail(email string) (*User, error)
	FindByID(id string) (*User, error)
	UpdateName(id, name string) error
	UpdatePassword(id, hashedPassword string) error
	SetVerificationCode(id, code string, expiresAt time.Time) error
	FindByEmailAndCode(email, code string) (*User, error)
	CompleteRegistration(id, email, name, hashedPassword string) error
	SetRefreshToken(id, refreshToken string) error
	FindByRefreshToken(refreshToken string) (*User, error)
	UpgradeGuest(id, email, name, hashedPassword string) error
	DeleteUser(id string) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(user *User) error {
	return r.db.Create(user).Error
}

func (r *repository) FindByEmail(email string) (*User, error) {
	var user User
	result := r.db.Where("email = ?", email).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (r *repository) FindByID(id string) (*User, error) {
	var user User
	result := r.db.Where("id = ?", id).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (r *repository) UpdateName(id, name string) error {
	result := r.db.Model(&User{}).Where("id = ?", id).Update("name", name)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *repository) UpdatePassword(id, hashedPassword string) error {
	result := r.db.Model(&User{}).Where("id = ?", id).Update("hashed_password", hashedPassword)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *repository) SetVerificationCode(id, code string, expiresAt time.Time) error {
	result := r.db.Model(&User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"verification_code":           code,
		"verification_code_expires_at": expiresAt,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *repository) FindByEmailAndCode(email, code string) (*User, error) {
	var user User
	result := r.db.Where("email = ? AND verification_code = ?", email, code).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidVerificationCode
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (r *repository) CompleteRegistration(id, email, name, hashedPassword string) error {
	result := r.db.Model(&User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"email":                        email,
		"name":                         name,
		"hashed_password":              hashedPassword,
		"email_verified":               true,
		"verification_code":            "",
		"verification_code_expires_at": nil,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *repository) SetRefreshToken(id, refreshToken string) error {
	result := r.db.Model(&User{}).Where("id = ?", id).Update("refresh_token", refreshToken)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *repository) FindByRefreshToken(refreshToken string) (*User, error) {
	var user User
	result := r.db.Where("refresh_token = ?", refreshToken).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrRefreshTokenInvalid
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (r *repository) DeleteUser(id string) error {
	result := r.db.Where("id = ?", id).Delete(&User{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *repository) UpgradeGuest(id, email, name, hashedPassword string) error {
	result := r.db.Model(&User{}).Where("id = ? AND is_guest = ?", id, true).Updates(map[string]interface{}{
		"email":                        email,
		"name":                         name,
		"hashed_password":              hashedPassword,
		"is_guest":                     false,
		"email_verified":               true,
		"verification_code":            "",
		"verification_code_expires_at": nil,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotAGuest
	}
	return nil
}
