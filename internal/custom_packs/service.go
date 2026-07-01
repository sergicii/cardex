package custompacks

import "fmt"

// Service define el contrato de operaciones de negocio para wishlist.
type Service interface {
	Upsert(userID string, productID uint64, delta int) (*Wishlist, error)
	Delete(userID string, wishlistID uint64) error
	GetByUserID(userID string) ([]Wishlist, error)
	IsInWishlist(userID string, productID uint64) (wishlistID uint64, found bool, err error)
	CreateBundle(userID string, items []BundleItem) (*Bundle, error)
	UpdateBundle(userID string, bundleID uint64, items []BundleItem) (*Bundle, error)
	DeleteBundle(userID string, bundleID uint64) error
	GetBundlesByUserID(userID string) ([]Bundle, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Upsert(userID string, productID uint64, delta int) (*Wishlist, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID requerido")
	}
	if productID == 0 {
		return nil, fmt.Errorf("productID requerido")
	}
	if delta == 0 {
		return nil, fmt.Errorf("delta no puede ser 0")
	}
	return s.repo.Upsert(userID, productID, delta)
}

func (s *service) Delete(userID string, wishlistID uint64) error {
	if userID == "" {
		return fmt.Errorf("userID requerido")
	}
	if wishlistID == 0 {
		return fmt.Errorf("wishlistID requerido")
	}
	return s.repo.Delete(userID, wishlistID)
}

func (s *service) GetByUserID(userID string) ([]Wishlist, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID requerido")
	}
	return s.repo.GetByUserID(userID)
}

func (s *service) IsInWishlist(userID string, productID uint64) (uint64, bool, error) {
	if userID == "" {
		return 0, false, fmt.Errorf("userID requerido")
	}
	if productID == 0 {
		return 0, false, fmt.Errorf("productID requerido")
	}
	return s.repo.IsInWishlist(userID, productID)
}

func (s *service) CreateBundle(userID string, items []BundleItem) (*Bundle, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID requerido")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("el bundle debe contener al menos un ítem")
	}
	return s.repo.CreateBundle(userID, items)
}

func (s *service) UpdateBundle(userID string, bundleID uint64, items []BundleItem) (*Bundle, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID requerido")
	}
	if bundleID == 0 {
		return nil, fmt.Errorf("bundleID requerido")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("el bundle debe contener al menos un ítem")
	}
	return s.repo.UpdateBundle(userID, bundleID, items)
}

func (s *service) DeleteBundle(userID string, bundleID uint64) error {
	if userID == "" {
		return fmt.Errorf("userID requerido")
	}
	if bundleID == 0 {
		return fmt.Errorf("bundleID requerido")
	}
	return s.repo.DeleteBundle(userID, bundleID)
}

func (s *service) GetBundlesByUserID(userID string) ([]Bundle, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID requerido")
	}
	return s.repo.GetBundlesByUserID(userID)
}
