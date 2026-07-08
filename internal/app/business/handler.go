package business

import (
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/utility"

	"github.com/go-playground/validator/v10"
)

type Handler struct {
	svc       *Service
	Validator *validator.Validate
	Logger    *utility.Logger
	Db        *database.Database
	TestDB    *database.Database
}

func NewHandler(validator *validator.Validate, logger *utility.Logger, db, testDB *database.Database) *Handler {
	prodDB := dbinit.InitDB(db.Postgresql.DB(), false)
	testDBConn := dbinit.InitDB(testDB.Postgresql.DB(), false)
	svc := NewServiceWithDB(prodDB, testDBConn)
	return &Handler{
		svc:       svc,
		Validator: validator,
		Logger:    logger,
		Db:        db,
		TestDB:    testDB,
	}
}
