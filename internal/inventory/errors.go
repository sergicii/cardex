package inventory

import "errors"

var (
	// ErrInventoryNotFound se devuelve cuando no existe un registro de inventario para ese usuario y carta.
	ErrInventoryNotFound = errors.New("registro de inventario no encontrado")

	// ErrInsufficientStock se devuelve cuando la operación requiere más unidades de las disponibles.
	ErrInsufficientStock = errors.New("stock insuficiente para realizar la operación")

	// ErrInvalidDelta se devuelve cuando el delta de una operación es 0 o negativo en contextos que lo prohíben.
	ErrInvalidDelta = errors.New("la cantidad de la operación debe ser mayor que cero")

	// ErrInvalidPrice se devuelve cuando el precio indicado es negativo.
	ErrInvalidPrice = errors.New("el precio no puede ser negativo")
)
