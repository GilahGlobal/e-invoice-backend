package admin

import (
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/data/repositories"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/utility"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Service interface {
	SetupInitialSuperAdmin(req AdminRegisterDto, isSandbox bool) (int, error)
	RegisterAdmin(req AdminRegisterDto, isSandbox bool) (int, error)
	LoginAdmin(req AdminLoginRequestDto, isSandbox bool) (map[string]interface{}, int, error)
}

type service struct {
	adminRepo repositories.AdminRepository
	authRepo  repositories.AuthRepository
	cfg       *config.Configuration
}

func NewServiceWithDB(prodDB, testDB database.DatabaseManager, cfg *config.Configuration) Service {
	adminRepo := repositories.NewAdminRepository()
	authRepo := repositories.NewAuthRepository(prodDB, testDB)
	return &service{adminRepo: adminRepo, authRepo: authRepo, cfg: cfg}
}

func (s *service) SetupInitialSuperAdmin(req AdminRegisterDto, isSandbox bool) (int, error) {
	db := s.authRepo.GetDB(isSandbox)
	count, err := s.adminRepo.CountAdmins(db)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to check existing admins: %w", err)
	}
	if count > 0 {
		return http.StatusForbidden, errors.New("initial setup is already complete")
	}

	req.Role = entities.RoleSuperAdmin
	return s.RegisterAdmin(req, isSandbox)
}

func (s *service) RegisterAdmin(req AdminRegisterDto, isSandbox bool) (int, error) {
	db := s.authRepo.GetDB(isSandbox)

	email := strings.ToLower(req.Email)
	existingAdmin, _ := s.adminRepo.GetAdminByEmail(db, email)
	if existingAdmin != nil {
		return http.StatusBadRequest, errors.New("admin with this email already exists")
	}

	passwordHash, err := utility.HashPassword(req.Password)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to hash password: %w", err)
	}

	admin := entities.Admin{
		ID:       utility.GenerateUUID(),
		Name:     req.Name,
		Email:    email,
		Password: passwordHash,
		Role:     req.Role,
	}

	err = s.adminRepo.CreateAdmin(&admin, db)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to create admin: %w", err)
	}

	return http.StatusCreated, nil
}

func (s *service) LoginAdmin(req AdminLoginRequestDto, isSandbox bool) (map[string]interface{}, int, error) {
	db := s.authRepo.GetDB(isSandbox)

	email := strings.ToLower(req.Email)
	admin, err := s.adminRepo.GetAdminByEmail(db, email)
	if err != nil || admin == nil {
		return nil, http.StatusBadRequest, errors.New("invalid credentials")
	}

	if !utility.CompareHash(req.Password, admin.Password) {
		return nil, http.StatusBadRequest, errors.New("invalid credentials")
	}

	tokenData, err := middleware.CreateAdminToken(*admin, isSandbox)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create token: %w", err)
	}

	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	accessToken := entities.AccessToken{ID: tokenData.AccessUuid, OwnerID: admin.ID}
	err = s.authRepo.CreateAccessToken(&accessToken, isSandbox, tokens)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to save token session: %w", err)
	}

	responseData := map[string]interface{}{
		"data": AdminResponse{
			ID:    admin.ID,
			Name:  admin.Name,
			Email: admin.Email,
			Role:  admin.Role,
		},
		"access_token": tokenData.AccessToken,
	}

	return responseData, http.StatusOK, nil
}
