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
	CreateBundle(userID string, items []BundleItem) (*Bundle, error)
	UpdateBundle(userID string, bundleID uint64, items []BundleItem) (*Bundle, error)
	DeleteBundle(userID string, bundleID uint64) error
	GetBundlesByUserID(userID string) ([]Bundle, error)
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

// CreateBundle crea un nuevo bundle con sus items asociados.
func (r *repository) CreateBundle(userID string, items []BundleItem) (*Bundle, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID no puede estar vacío")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("el bundle debe contener al menos un ítem")
	}

	bundle := &Bundle{
		UserID: userID,
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(bundle).Error; err != nil {
			return err
		}

		for i := range items {
			items[i].BundleID = bundle.ID
		}

		if err := tx.Create(&items).Error; err != nil {
			return err
		}

		bundle.Items = items
		return nil
	})

	if err != nil {
		return nil, err
	}

	var result Bundle
	if err := r.db.Preload("Items").First(&result, bundle.ID).Error; err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateBundle edita un bundle existente reemplazando sus items asociados.
func (r *repository) UpdateBundle(userID string, bundleID uint64, items []BundleItem) (*Bundle, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID no puede estar vacío")
	}
	if bundleID == 0 {
		return nil, fmt.Errorf("bundleID no puede ser 0")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("el bundle debe contener al menos un ítem")
	}

	var bundle Bundle
	if err := r.db.Where("id = ? AND user_id = ?", bundleID, userID).First(&bundle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("bundle no encontrado")
		}
		return nil, err
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("bundle_id = ?", bundleID).Delete(&BundleItem{}).Error; err != nil {
			return err
		}

		for i := range items {
			items[i].BundleID = bundleID
			items[i].ID = 0
		}

		if err := tx.Create(&items).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	var result Bundle
	if err := r.db.Preload("Items").First(&result, bundleID).Error; err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteBundle elimina un bundle y sus items asociados.
func (r *repository) DeleteBundle(userID string, bundleID uint64) error {
	if userID == "" {
		return fmt.Errorf("userID no puede estar vacío")
	}
	if bundleID == 0 {
		return fmt.Errorf("bundleID no puede ser 0")
	}

	result := r.db.
		Where("id = ? AND user_id = ?", bundleID, userID).
		Delete(&Bundle{})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("bundle no encontrado")
	}
	return nil
}

// GetBundlesByUserID obtiene todos los bundles de un usuario con sus items y stock pre-cargados.
func (r *repository) GetBundlesByUserID(userID string) ([]Bundle, error) {
	var bundles []Bundle
	result := r.db.
		Preload("Items.Stock.Product").
		Where("user_id = ?", userID).
		Find(&bundles)

	if result.Error != nil {
		return nil, result.Error
	}
	return bundles, nil
}
