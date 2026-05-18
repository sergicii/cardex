package users

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// RegisterInput contiene los datos necesarios para registrar un nuevo usuario.
type RegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginInput contiene las credenciales para iniciar sesión.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Service define el contrato de lo que nuestra aplicación puede hacer con los usuarios.
type Service interface {
	Register(input RegisterInput) (*User, error)
	Login(input LoginInput) (*User, error)
}

// service implementa la interfaz Service e inyecta el repositorio.
type service struct {
	repo Repository
}

// NewService crea una nueva instancia del servicio de usuarios.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// Register valida los datos, hashea la contraseña y crea el usuario en la base de datos.
// Devuelve error si el email ya está registrado o si los datos son inválidos.
func (s *service) Register(input RegisterInput) (*User, error) {
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.Name = strings.TrimSpace(input.Name)

	if input.Email == "" {
		return nil, fmt.Errorf("el email es obligatorio")
	}
	if input.Name == "" {
		return nil, fmt.Errorf("el nombre es obligatorio")
	}
	if len(input.Password) < 8 {
		return nil, fmt.Errorf("la contraseña debe tener al menos 8 caracteres")
	}

	// Verificar si el email ya está en uso
	existing, err := s.repo.FindByEmail(input.Email)
	if err != nil && err != ErrUserNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	// Hashear la contraseña con bcrypt (cost=12 es un buen balance de seguridad/rendimiento)
	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("error al procesar la contraseña: %w", err)
	}

	user := &User{
		ID:             uuid.NewString(),
		Name:           input.Name,
		Email:          input.Email,
		HashedPassword: string(hashed),
	}

	if err := s.repo.Create(user); err != nil {
		return nil, fmt.Errorf("error al crear el usuario: %w", err)
	}

	return user, nil
}

// Login verifica las credenciales del usuario y devuelve el usuario si son correctas.
// Nunca revela si el email no existe (devuelve ErrInvalidCredentials en ambos casos)
// para evitar enumeración de usuarios.
func (s *service) Login(input LoginInput) (*User, error) {
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))

	if input.Email == "" || input.Password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.FindByEmail(input.Email)
	if err != nil {
		// Enmascarar ErrUserNotFound para no revelar si el email existe
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}
