package core

import (
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/utility"

	"github.com/go-playground/validator/v10"
)

// Container holds all application dependencies and services
type Container struct {
	Config    *config.Configuration
	DB        *database.Database
	TestDB    *database.Database
	Logger    *utility.Logger
	Validator *validator.Validate
	Keys      *utility.CryptoKeys
}

// NewContainer initializes a new dependency injection container
func NewContainer(
	cfg *config.Configuration,
	db *database.Database,
	testDb *database.Database,
	logger *utility.Logger,
	validator *validator.Validate,
	keys *utility.CryptoKeys,
) *Container {
	c := &Container{
		Config:    cfg,
		DB:        db,
		TestDB:    testDb,
		Logger:    logger,
		Validator: validator,
		Keys:      keys,
	}

	return c
}
