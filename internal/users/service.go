package users

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/operaodev/cardex/internal/mailer"
	"golang.org/x/crypto/bcrypt"
)

type RegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Code     string `json:"code"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ChangePasswordInput struct {
	UserID      string `json:"-"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type ChangeNameInput struct {
	UserID string `json:"-"`
	Name   string `json:"name"`
}

type SendVerificationCodeInput struct {
	Email string `json:"email"`
}

type UpgradeGuestInput struct {
	UserID   string `json:"-"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Code     string `json:"code"`
}

type Service interface {
	RegisterGuest() (*User, error)
	SendVerificationCode(input SendVerificationCodeInput) error
	Register(input RegisterInput) (*User, error)
	Login(input LoginInput) (*User, error)
	GetByID(id string) (*User, error)
	UpgradeGuest(input UpgradeGuestInput) (*User, error)
	RefreshSession(userID, refreshTokenHash string) (*User, error)
	StoreRefreshToken(userID, refreshTokenHash string) error
	ChangePassword(input ChangePasswordInput) error
	ChangeName(input ChangeNameInput) (*User, error)
}

type service struct {
	repo   Repository
	mailer mailer.Mailer
}

func NewService(repo Repository, mailer mailer.Mailer) Service {
	return &service{repo: repo, mailer: mailer}
}

func (s *service) RegisterGuest() (*User, error) {
	user := &User{
		ID:      uuid.NewString(),
		IsGuest: true,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, fmt.Errorf("error al crear usuario invitado: %w", err)
	}

	return user, nil
}

func (s *service) SendVerificationCode(input SendVerificationCodeInput) error {
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	if input.Email == "" {
		return fmt.Errorf("el email es obligatorio")
	}

	existing, err := s.repo.FindByEmail(input.Email)
	if err != nil && err != ErrUserNotFound {
		return err
	}

	if existing != nil {
		if existing.EmailVerified {
			return ErrEmailAlreadyExists
		}

		if existing.VerificationCodeExpiresAt != nil && time.Now().UTC().Before(*existing.VerificationCodeExpiresAt) {
			if err := s.mailer.SendVerificationCode(input.Email, existing.VerificationCode); err != nil {
				return fmt.Errorf("error al reenviar código: %w", err)
			}
			return nil
		}
	}

	code := generateVerificationCode()

	if existing != nil {
		expiresAt := time.Now().UTC().Add(5 * time.Minute)
		if err := s.repo.SetVerificationCode(existing.ID, code, expiresAt); err != nil {
			return err
		}
	} else {
		expiresAt := time.Now().UTC().Add(5 * time.Minute)
		user := &User{
			ID:                       uuid.NewString(),
			Email:                    input.Email,
			VerificationCode:         code,
			VerificationCodeExpiresAt: &expiresAt,
		}
		if err := s.repo.Create(user); err != nil {
			return fmt.Errorf("error al crear usuario: %w", err)
		}
	}

	if err := s.mailer.SendVerificationCode(input.Email, code); err != nil {
		return fmt.Errorf("error al enviar código: %w", err)
	}

	return nil
}

func (s *service) Register(input RegisterInput) (*User, error) {
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.Name = strings.TrimSpace(input.Name)

	if input.Email == "" {
		return nil, fmt.Errorf("el email es obligatorio")
	}
	if input.Name == "" {
		return nil, fmt.Errorf("el nombre es obligatorio")
	}
	if input.Code == "" {
		return nil, fmt.Errorf("el código de verificación es obligatorio")
	}
	if len(input.Password) < 8 {
		return nil, ErrPasswordTooShort
	}

	user, err := s.repo.FindByEmailAndCode(input.Email, input.Code)
	if err != nil {
		return nil, err
	}

	if user.VerificationCodeExpiresAt == nil || time.Now().UTC().After(*user.VerificationCodeExpiresAt) {
		return nil, ErrVerificationCodeExpired
	}

	if user.IsGuest {
		return nil, fmt.Errorf("los usuarios invitados deben usar el endpoint de upgrade")
	}

	if user.EmailVerified {
		return nil, ErrEmailAlreadyExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("error al procesar la contraseña: %w", err)
	}

	if err := s.repo.CompleteRegistration(user.ID, user.Email, input.Name, string(hashed)); err != nil {
		return nil, fmt.Errorf("error al completar registro: %w", err)
	}

	user.Name = input.Name
	user.EmailVerified = true
	user.VerificationCode = ""
	user.VerificationCodeExpiresAt = nil

	return user, nil
}

func (s *service) Login(input LoginInput) (*User, error) {
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))

	if input.Email == "" || input.Password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.FindByEmail(input.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if user.IsGuest {
		return nil, ErrInvalidCredentials
	}

	if user.HashedPassword == "" {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	return user, nil
}

func (s *service) GetByID(id string) (*User, error) {
	return s.repo.FindByID(id)
}

func (s *service) UpgradeGuest(input UpgradeGuestInput) (*User, error) {
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.Name = strings.TrimSpace(input.Name)

	if input.Email == "" {
		return nil, fmt.Errorf("el email es obligatorio")
	}
	if input.Name == "" {
		return nil, fmt.Errorf("el nombre es obligatorio")
	}
	if input.Code == "" {
		return nil, fmt.Errorf("el código de verificación es obligatorio")
	}
	if len(input.Password) < 8 {
		return nil, ErrPasswordTooShort
	}

	guest, err := s.repo.FindByID(input.UserID)
	if err != nil {
		return nil, err
	}

	if !guest.IsGuest {
		return nil, ErrNotAGuest
	}

	preReg, err := s.repo.FindByEmailAndCode(input.Email, input.Code)
	if err != nil {
		return nil, err
	}

	if preReg.VerificationCodeExpiresAt == nil || time.Now().UTC().After(*preReg.VerificationCodeExpiresAt) {
		return nil, ErrVerificationCodeExpired
	}

	if preReg.EmailVerified {
		return nil, ErrEmailAlreadyExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("error al procesar la contraseña: %w", err)
	}

	if err := s.repo.DeleteUser(preReg.ID); err != nil {
		return nil, fmt.Errorf("error al eliminar pre-registro: %w", err)
	}

	if err := s.repo.UpgradeGuest(input.UserID, input.Email, input.Name, string(hashed)); err != nil {
		return nil, err
	}

	guest.IsGuest = false
	guest.Name = input.Name
	guest.Email = input.Email
	guest.EmailVerified = true
	guest.HashedPassword = string(hashed)

	return guest, nil
}

func (s *service) RefreshSession(userID, refreshTokenHash string) (*User, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	if user.RefreshToken != refreshTokenHash {
		return nil, ErrRefreshTokenInvalid
	}

	return user, nil
}

func (s *service) StoreRefreshToken(userID, refreshTokenHash string) error {
	return s.repo.SetRefreshToken(userID, refreshTokenHash)
}

func (s *service) ChangePassword(input ChangePasswordInput) error {
	if len(input.NewPassword) < 8 {
		return ErrPasswordTooShort
	}

	user, err := s.repo.FindByID(input.UserID)
	if err != nil {
		return err
	}

	if user.HashedPassword == "" {
		return ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(input.OldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), 12)
	if err != nil {
		return fmt.Errorf("error al procesar la contraseña: %w", err)
	}

	return s.repo.UpdatePassword(user.ID, string(hashed))
}

func (s *service) ChangeName(input ChangeNameInput) (*User, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return nil, fmt.Errorf("el nombre es obligatorio")
	}

	if err := s.repo.UpdateName(input.UserID, input.Name); err != nil {
		return nil, err
	}

	return s.repo.FindByID(input.UserID)
}

func generateRandomToken(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return uuid.NewString()
	}
	return hex.EncodeToString(bytes)
}

func generateVerificationCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "000000"
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
