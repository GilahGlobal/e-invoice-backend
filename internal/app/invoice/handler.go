package invoice

import (
	"einvoice-access-point/internal/app/business"
	"einvoice-access-point/internal/app/token"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/utility"

	"github.com/go-playground/validator/v10"
)

type Handler struct {
	svc         *Service
	businessSvc *business.Service
	Validator   *validator.Validate
	Logger      *utility.Logger
	Db          *database.Database
	TestDB      *database.Database
	Keys        *utility.CryptoKeys
}

func NewHandler(validator *validator.Validate, logger *utility.Logger, db, testDB *database.Database, keys *utility.CryptoKeys) *Handler {
	prodDB := dbinit.InitDB(db.Postgresql.DB(), false)
	testDBConn := dbinit.InitDB(testDB.Postgresql.DB(), false)
	tokenSvc := token.NewServiceWithDB(prodDB, testDBConn)
	businessSvc := business.NewServiceWithDB(prodDB, testDBConn)
	svc := NewServiceWithDB(prodDB, testDBConn, tokenSvc, businessSvc)
	return &Handler{
		svc:         svc,
		businessSvc: businessSvc,
		Validator:   validator,
		Logger:      logger,
		Db:          db,
		TestDB:      testDB,
		Keys:        keys,
	}
}
