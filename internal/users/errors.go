package users

import "errors"

var (
	// ErrUserNotFound se devuelve cuando no existe un usuario con el email dado.
	ErrUserNotFound = errors.New("usuario no encontrado")

	// ErrEmailAlreadyExists se devuelve cuando el email ya está registrado.
	ErrEmailAlreadyExists = errors.New("el email ya está registrado")

	// ErrInvalidCredentials se devuelve cuando la contraseña no coincide.
	ErrInvalidCredentials = errors.New("credenciales inválidas")
)
