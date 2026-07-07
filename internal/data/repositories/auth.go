package repositories

import (
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
	"fmt"
	"net/http"
)

type AuthRepository interface {
	GetDB(isSandbox bool) database.DatabaseManager
	GetAccessTokens(a *entities.AccessToken, isSandbox bool) error
	GetByOwnerID(a *entities.AccessToken, isSandbox bool) (int, error)
	GetByID(ID string, isSandbox bool) (entities.AccessToken, error)
	GetByIDBoolean(a *entities.AccessToken, isSandbox bool) (int, error)
	GetLatestByOwnerIDAndIsLive(a *entities.AccessToken, isSandbox bool) (int, error)
	CreateAccessToken(a *entities.AccessToken, isSandbox bool, tokenData interface{}) error
	RevokeAccessToken(a *entities.AccessToken, isSandbox bool) error
}

type authRepository struct {
	prodDB database.DatabaseManager
	testDB database.DatabaseManager
}

func NewAuthRepository(prodDB, testDB database.DatabaseManager) AuthRepository {
	return &authRepository{
		prodDB: prodDB,
		testDB: testDB,
	}
}

func (r *authRepository) GetDB(isSandbox bool) database.DatabaseManager {
	if isSandbox {
		return r.testDB
	}
	return r.prodDB
}

func (r *authRepository) GetAccessTokens(a *entities.AccessToken, isSandbox bool) error {
	db := r.GetDB(isSandbox)
	err := db.SelectFirstFromDb(&a)
	if err != nil {
		return fmt.Errorf("token selection failed: %v", err.Error())
	}
	return nil
}

func (r *authRepository) GetByOwnerID(a *entities.AccessToken, isSandbox bool) (int, error) {
	db := r.GetDB(isSandbox)
	err, nilErr := db.SelectOneFromDb(db, &a, "owner_id = ? ", a.OwnerID)
	if nilErr != nil {
		return http.StatusBadRequest, nilErr
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (r *authRepository) GetByID(ID string, isSandbox bool) (entities.AccessToken, error) {
	db := r.GetDB(isSandbox)
	var accessT entities.AccessToken

	query := db.DB().Where("id = ?", ID)

	if err := query.First(&accessT).Error; err != nil {
		return accessT, err
	}

	return accessT, nil
}

func (r *authRepository) GetByIDBoolean(a *entities.AccessToken, isSandbox bool) (int, error) {
	db := r.GetDB(isSandbox)
	err, nilErr := db.SelectOneFromDb(&a, "id = ? ", a.ID)
	if nilErr != nil {
		return http.StatusBadRequest, nilErr
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (r *authRepository) GetLatestByOwnerIDAndIsLive(a *entities.AccessToken, isSandbox bool) (int, error) {
	db := r.GetDB(isSandbox)
	err, nilErr := db.SelectLatestFromDb(&a, "owner_id = ? and is_live = ? ", a.OwnerID, a.IsLive)
	if nilErr != nil {
		return http.StatusBadRequest, nilErr
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (r *authRepository) CreateAccessToken(a *entities.AccessToken, isSandbox bool, tokenData interface{}) error {
	db := r.GetDB(isSandbox)
	if a.OwnerID == "" {
		return fmt.Errorf("owner id not provided to create access token")
	}

	if a.ID == "" {
		return fmt.Errorf("access id not provided to create access token")
	}

	var (
		access_token = tokenData.(map[string]string)["access_token"]
		exp          = tokenData.(map[string]string)["exp"]
	)

	a.IsLive = true
	a.LoginAccessToken = access_token
	a.LoginAccessTokenExpiresIn = exp
	err := db.CreateOneRecord(&a)
	if err != nil {
		return fmt.Errorf("user creation failed: %v", err.Error())
	}
	return nil
}

func (r *authRepository) RevokeAccessToken(a *entities.AccessToken, isSandbox bool) error {
	db := r.GetDB(isSandbox)
	if a.ID == "" {
		return fmt.Errorf("access token id not provided to revoke access token")
	}
	a.IsLive = false
	_, err := db.SaveAllFields(&a)
	return err
}
