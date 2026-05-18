package users

import (
	"errors"

	"gorm.io/gorm"
)

// Repository define los métodos que nuestra capa de datos de usuarios debe tener.
type Repository interface {
	Create(user *User) error
	FindByEmail(email string) (*User, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository crea una nueva instancia del repositorio de usuarios.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Create ejecuta un INSERT INTO users con los datos del usuario ya validados.
// El campo hashed_password debe venir pre-hasheado desde la capa de servicio.
func (r *repository) Create(user *User) error {
	return r.db.Create(user).Error
}

// FindByEmail busca un usuario por su dirección de correo electrónico.
// A nivel SQL ejecuta un SELECT * FROM users WHERE email = ? LIMIT 1.
// Devuelve un error específico si el usuario no existe.
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
