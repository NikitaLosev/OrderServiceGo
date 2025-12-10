package service

import (
	"fmt"

	"github.com/go-playground/validator/v10"

	"orderservice/pkg/models"
)

var validate = validator.New()

func ValidateOrder(o models.Order) error {
	if err := validate.Struct(o); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	return nil
}
