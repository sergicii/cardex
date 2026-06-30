package custompacks

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Upsert(userID string, productID uint64, delta int) (*Wishlist, error)
	Delete(userID string, wishlistID uint64) error
	GetByUserID(userID string) ([]Wishlist, error)
	IsInWishlist(userID string, productID uint64) (wishlistID uint64, found bool, err error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Upsert incrementa/decrementa la cantidad de un item en la wishlist.
// Si la cantidad resultante es <= 0, lo elimina.
func (r *repository) Upsert(userID string, productID uint64, delta int) (*Wishlist, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID no puede estar vacío")
	}
	if productID == 0 {
		return nil, fmt.Errorf("productID no puede ser 0")
	}

	// Intentamos insertar o actualizar
	wish := Wishlist{
		UserID:    userID,
		ProductID: productID,
		Quantity:  delta,
	}

	result := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "product_id"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"quantity":   gorm.Expr("wishlists.quantity + ?", delta),
			"updated_at": gorm.Expr("NOW()"),
		}),
	}).Create(&wish)

	if result.Error != nil {
		return nil, result.Error
	}

	// Recargamos para obtener el valor real post-upsert
	if err := r.db.First(&wish, wish.ID).Error; err != nil {
		return nil, err
	}

	// Si la cantidad bajó a 0 o menos, eliminamos
	if wish.Quantity <= 0 {
		if err := r.db.Delete(&wish).Error; err != nil {
			return nil, err
		}
		return nil, nil
	}

	return &wish, nil
}

// Delete elimina un item de la wishlist por su ID.
func (r *repository) Delete(userID string, wishlistID uint64) error {
	result := r.db.
		Where("user_id = ?", userID).
		Where("id = ?", wishlistID).
		Delete(&Wishlist{})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("item no encontrado")
	}
	return nil
}

// GetByUserID obtiene todos los items de la wishlist de un usuario.
func (r *repository) GetByUserID(userID string) ([]Wishlist, error) {
	var items []Wishlist
	result := r.db.
		Preload("Product").
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&items)

	if result.Error != nil {
		return nil, result.Error
	}
	return items, nil
}

// IsInWishlist verifica si una carta está en la wishlist del usuario.
// Retorna el ID de la entrada de wishlist y true si existe, o 0 y false si no.
func (r *repository) IsInWishlist(userID string, productID uint64) (uint64, bool, error) {
	var wish Wishlist
	err := r.db.
		Select("id").
		Where("user_id = ? AND product_id = ?", userID, productID).
		First(&wish).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return wish.ID, true, nil
}
