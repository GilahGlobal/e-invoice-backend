package auth

import (
	"crypto/sha256"
	"einvoice-access-point/internal/common"
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/database/redis"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/data/repositories"
	"einvoice-access-point/internal/middleware"
	"einvoice-access-point/internal/pkg/resend_email"
	"einvoice-access-point/internal/utility"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	mainRedis "github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type Service interface {
	ResendVerificationOTP(email string, isSandbox bool) error
	ValidateCreateUserRequest(req RegisterDto) (RegisterDto, error)
	CreateUser(req RegisterDto, isSandbox bool) (int, error)
	LoginUser(req LoginRequestDto, isSandbox bool) (map[string]interface{}, int, error)
	LogoutUser(accessUuid, ownerId string, isSandbox bool) (map[string]interface{}, int, error)
	InitiateForgotPassword(req InitiateForgotPasswordDto, isSandbox bool) error
	InitiateForgotPasswordAcrossEnvironments(req InitiateForgotPasswordDto) error
	ChangePassword(userID string, req ChangePasswordDto, isSandbox bool) error
	CompleteForgotPasswordAcrossEnvironments(req CompleteForgotPasswordDto) error
	ToggleApplicationMode(email string, isSandbox bool) (map[string]interface{}, int, error)
	SynchronizeSandboxToProduction(email string) error
	VerifyBusinessAccount(req VerifyEmailDto, isSandbox bool) (map[string]interface{}, error)
	VerifyProdBuisnessAccount(req VerifyEmailDto) error
	SendOtp(email, key string)
}

type service struct {
	repo         repositories.AuthRepository
	businessRepo repositories.BusinessRepository
	cfg          *config.Configuration
}

func NewService(repo repositories.AuthRepository, businessRepo repositories.BusinessRepository, cfg *config.Configuration) Service {
	return &service{repo: repo, businessRepo: businessRepo, cfg: cfg}
}

func NewServiceWithDB(prodDB, testDB database.DatabaseManager, cfg *config.Configuration) Service {
	repo := repositories.NewAuthRepository(prodDB, testDB)
	businessRepo := repositories.NewBusinessRepository(prodDB, testDB)
	return NewService(repo, businessRepo, cfg)
}

func forgotPasswordKey(email string) string {
	return "forgot_password_otp_" + strings.ToLower(strings.TrimSpace(email))
}

func VerifyEmailKey(email string) string {
	return "verify_email_otp_" + email
}

func (s *service) ResendVerificationOTP(email string, isSandbox bool) error {
	db := s.repo.GetDB(isSandbox)
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	user := entities.Business{}
	queryError, err := db.SelectOneFromDb(&user, "email = ?", normalizedEmail)
	if err != nil {
		return fmt.Errorf("account details cannot be retrieved")
	}
	if queryError != nil {
		return queryError
	}
	log.Println(user.EmailVerified, user.Email)
	if user.EmailVerified {
		return errors.New("email already verified")
	}
	key := VerifyEmailKey(email)
	s.SendOtp(user.Email, key)

	return nil
}

func (s *service) ValidateCreateUserRequest(req RegisterDto) (RegisterDto, error) {
	devDb := s.repo.GetDB(true)
	prodDb := s.repo.GetDB(false)
	business := entities.Business{}

	if req.Email != "" {
		req.Email = strings.ToLower(req.Email)
		formattedMail, checkBool := utility.EmailValid(req.Email)
		if !checkBool {
			return req, fmt.Errorf("email address is invalid")
		}
		req.Email = formattedMail
		exists := devDb.CheckExists(&business, "email = ?", req.Email)
		if exists {
			return req, errors.New("user already exists with the given email")
		}
		exists = prodDb.CheckExists(&business, "email = ?", req.Email)
		if exists {
			return req, errors.New("user already exists with the given email in production")
		}
	}

	if exists := devDb.CheckExists(&business, "LOWER(company_name) = LOWER(?)", req.CompanyName); exists {
		return req, errors.New("Business already exists with the given company name")
	}
	exists := prodDb.CheckExists(&business, "LOWER(company_name) = LOWER(?)", req.CompanyName)
	if exists {
		return req, errors.New("Business already exists with the given company name")
	}

	return req, nil
}

func (s *service) CreateUser(req RegisterDto, isSandbox bool) (int, error) {
	db := s.repo.GetDB(isSandbox)
	serverSecret := s.cfg.Server.Secret
	email := strings.ToLower(req.Email)
	name := strings.Title(strings.ToLower(req.Name))

	password, err := utility.HashPassword(req.Password)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to hash password: %w", err)
	}

	apiKey, err := utility.GenerateSecureToken(32, serverSecret)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to generate api key: %w", err)
	}
	encryptedAPIKey, err := common.EncryptAES(apiKey)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to encrypt API key: %w", err)
	}
	apiKeyHash := sha256.Sum256([]byte(apiKey))
	apiKeyHashStr := hex.EncodeToString(apiKeyHash[:])

	platformConfigs := entities.PlatformConfigs{}
	for platform, cfg := range req.PlatformConfigs {
		encryptedHMACSecret, err := common.EncryptAES(string(cfg.HMACSecret))
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("failed to encrypt HMAC secret for %s: %w", platform, err)
		}
		encryptedAPIKey, err := common.EncryptAES(string(cfg.APIKey))
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("failed to encrypt API key for %s: %w", platform, err)
		}
		encryptedAPISecret, err := common.EncryptAES(string(cfg.APISecret))
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("failed to encrypt API secret for %s: %w", platform, err)
		}
		encryptedAuthToken, err := common.EncryptAES(string(cfg.AuthToken))
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("failed to encrypt Auth token for %s: %w", platform, err)
		}

		platformConfigs[platform] = entities.AccountingPlatformConfig{
			OrgID:      cfg.OrgID,
			HMACSecret: common.EncryptedString(encryptedHMACSecret),
			AuthToken:  common.EncryptedString(encryptedAuthToken),
			APIKey:     common.EncryptedString(encryptedAPIKey),
			APISecret:  common.EncryptedString(encryptedAPISecret),
		}
	}

	user := entities.Business{
		ID:              utility.GenerateUUID(),
		Name:            name,
		Email:           email,
		Password:        password,
		APIKey:          common.EncryptedString(encryptedAPIKey),
		APIKeyHash:      apiKeyHashStr,
		PlatformConfigs: platformConfigs,
		AccStatus:       0,
		TIN:             req.TIN,
		PhoneNumber:     req.PhoneNumber,
		CompanyName:     req.CompanyName,
		IsAggregator:    *req.IsAggregator,
	}

	err = s.businessRepo.CreateBusiness(&user, db)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to create business: %w", err)
	}

	return http.StatusCreated, nil
}

func (s *service) LoginUser(req LoginRequestDto, isSandbox bool) (map[string]interface{}, int, error) {
	redisClient := redis.NewClient()
	ctx := redisClient.Context()
	db := s.repo.GetDB(isSandbox)
	var user = entities.Business{}

	exists := db.CheckExists(&user, "email = ?", req.Email)
	if !exists {
		return nil, 400, fmt.Errorf("invalid credentials")
	}

	if !utility.CompareHash(req.Password, user.Password) {
		return nil, 400, fmt.Errorf("invalid credentials")
	}

	userData, err := s.businessRepo.GetUserByEmail(db, req.Email)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("unable to fetch user: %w", err)
	}

	if !userData.EmailVerified {
		otp := 123456 // For testing purposes only, replace with generated OTP
		key := VerifyEmailKey(userData.Email)
		duration := 15 * time.Minute // 15 minutes expiration

		redisClient.Set(ctx, key, strconv.Itoa(otp), duration)
		return nil, http.StatusExpectationFailed, fmt.Errorf("Email has not been verified, an otp has been sent to your mail, use it to verify your email")
	}
	tokenData, err := middleware.CreateToken(user, isSandbox)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error saving token: %w", err)
	}
	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	accessToken := entities.AccessToken{ID: tokenData.AccessUuid, OwnerID: user.ID}

	err = s.repo.CreateAccessToken(&accessToken, isSandbox, tokens)

	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error saving token: %w", err)
	}

	responseData := map[string]interface{}{
		"data": UserResponse{
			ID:           userData.ID,
			Email:        userData.Email,
			Name:         userData.Name,
			BusinessID:   userData.BusinessID,
			ServiceID:    userData.ServiceID,
			IsSandbox:    isSandbox,
			IsAggregator: userData.IsAggregator,
			KeysSet:      userData.KeysSet,
		},
		"access_token": tokenData.AccessToken,
	}

	return responseData, http.StatusOK, nil
}

func (s *service) LogoutUser(accessUuid, ownerId string, isSandbox bool) (map[string]interface{}, int, error) {
	var responseData map[string]interface{}
	accessToken := entities.AccessToken{ID: accessUuid, OwnerID: ownerId}

	err := s.repo.RevokeAccessToken(&accessToken, isSandbox)
	if err != nil {
		return responseData, http.StatusInternalServerError, fmt.Errorf("error revoking user session: %w", err)
	}

	responseData = map[string]interface{}{}
	return responseData, http.StatusOK, nil
}

func (s *service) InitiateForgotPassword(req InitiateForgotPasswordDto, isSandbox bool) error {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	db := s.repo.GetDB(isSandbox)
	user := entities.Business{}
	queryError, err := db.SelectOneFromDb(&user, "email = ?", email)
	if err != nil {
		return fmt.Errorf("account details cannot be retrieved")
	}

	if queryError != nil {
		return queryError
	}

	key := forgotPasswordKey(email)
	s.SendOtp(user.Email, key)
	return nil
}

func (s *service) InitiateForgotPasswordAcrossEnvironments(req InitiateForgotPasswordDto) error {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	for _, isSandbox := range []bool{false, true} {
		db := s.repo.GetDB(isSandbox)
		user := entities.Business{}
		queryError, err := db.SelectOneFromDb(&user, "email = ?", email)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return fmt.Errorf("account details cannot be retrieved")
		}

		if queryError != nil {
			if errors.Is(queryError, gorm.ErrRecordNotFound) {
				continue
			}
			return queryError
		}

		key := forgotPasswordKey(email)
		s.SendOtp(user.Email, key)
		return nil
	}

	return fmt.Errorf("account details cannot be retrieved")
}

func (s *service) ChangePassword(userID string, req ChangePasswordDto, isSandbox bool) error {
	db := s.repo.GetDB(isSandbox)
	user, err := s.businessRepo.FindUserByID(db, userID)
	if err != nil {
		return fmt.Errorf("account details cannot be retrieved")
	}

	if !utility.CompareHash(req.OldPassword, user.Password) {
		return errors.New("old password is incorrect")
	}

	if req.OldPassword == req.NewPassword {
		return errors.New("new password must be different from old password")
	}

	password, err := utility.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.Password = password
	if _, err := db.UpdateFields(*user, *user, user.ID); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

func (s *service) CompleteForgotPasswordAcrossEnvironments(req CompleteForgotPasswordDto) error {
	redisClient := redis.NewClient()
	ctx := redisClient.Context()
	email := strings.ToLower(strings.TrimSpace(req.Email))
	key := forgotPasswordKey(email)

	otp, err := redisClient.Get(ctx, key).Result()

	if err == mainRedis.Nil {
		log.Println("otp not found or has expired")
		return errors.New("otp has expired")
	}

	if err != nil {
		log.Println("unable to verify otp", err)
		return errors.New("unable to verify otp")
	}

	if otp != req.OTP {
		log.Println("invalid OTP provided")
		return errors.New("invalid OTP provided")
	}

	password, err := utility.HashPassword(req.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	updated := false
	for _, isSandbox := range []bool{false, true} {
		db := s.repo.GetDB(isSandbox)
		user := entities.Business{}
		queryError, err := db.SelectOneFromDb(&user, "email = ?", email)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return fmt.Errorf("account details cannot be retrieved: %w", err)
		}

		if queryError != nil {
			if errors.Is(queryError, gorm.ErrRecordNotFound) {
				continue
			}
			return queryError
		}

		user.Password = password
		if _, err := db.UpdateFields(user, user, user.ID); err != nil {
			return fmt.Errorf("failed to update password: %w", err)
		}
		updated = true
	}

	if !updated {
		return fmt.Errorf("account details cannot be retrieved")
	}

	redisClient.Del(ctx, key)
	return nil
}

func (s *service) ToggleApplicationMode(email string, isSandbox bool) (map[string]interface{}, int, error) {
	db := s.repo.GetDB(isSandbox)
	userData, err := s.businessRepo.GetUserByEmail(db, email)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("unable to fetch user: %w", err)
	}

	tokenData, err := middleware.CreateToken(userData, isSandbox)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error saving token: %w", err)
	}
	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	accessToken := entities.AccessToken{ID: tokenData.AccessUuid, OwnerID: userData.ID}

	err = s.repo.CreateAccessToken(&accessToken, isSandbox, tokens)

	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error saving token: %w", err)
	}

	responseData := map[string]interface{}{
		"data": UserResponse{
			ID:           userData.ID,
			Email:        userData.Email,
			Name:         userData.Name,
			BusinessID:   userData.BusinessID,
			ServiceID:    userData.ServiceID,
			IsSandbox:    isSandbox,
			IsAggregator: userData.IsAggregator,
			KeysSet:      userData.KeysSet,
		},
		"access_token": tokenData.AccessToken,
	}
	return responseData, http.StatusOK, nil
}

func (s *service) SynchronizeSandboxToProduction(email string) error {
	pDB := s.repo.GetDB(false)
	sDB := s.repo.GetDB(true)

	exists := pDB.CheckExistsInTable("businesses", "email = ?", email)

	if !exists {
		sandboxExists := sDB.CheckExistsInTable("businesses", "email = ?", email)
		if sandboxExists {
			userData, err := s.businessRepo.GetUserByEmail(sDB, email)
			if err != nil {
				log.Println("unable to fetch user from sandbox: " + err.Error())
				return fmt.Errorf("unable to fetch user from sandbox: %w", err)
			}

			serverSecret := s.cfg.Server.Secret

			apiKey, err := utility.GenerateSecureToken(32, serverSecret)
			if err != nil {
				log.Println("failed to generate api key: " + err.Error())
				return fmt.Errorf("failed to generate api key: %w", err)
			}
			encryptedAPIKey, err := common.EncryptAES(apiKey)
			if err != nil {
				log.Println("failed to encrypt API key: " + err.Error())
				return fmt.Errorf("failed to encrypt API key: %w", err)
			}
			apiKeyHash := sha256.Sum256([]byte(apiKey))
			apiKeyHashStr := hex.EncodeToString(apiKeyHash[:])

			platformConfigs := entities.PlatformConfigs{}
			for platform, cfg := range userData.PlatformConfigs {
				encryptedHMACSecret, err := common.EncryptAES(string(cfg.HMACSecret))
				if err != nil {
					log.Printf("failed to encrypt HMAC secret for %s: %v", platform, err)
					return fmt.Errorf("failed to encrypt HMAC secret for %s: %w", platform, err)
				}
				encryptedAPIKey, err := common.EncryptAES(string(cfg.APIKey))
				if err != nil {
					log.Printf("failed to encrypt API key for %s: %v", platform, err)
					return fmt.Errorf("failed to encrypt API key for %s: %w", platform, err)
				}
				encryptedAPISecret, err := common.EncryptAES(string(cfg.APISecret))
				if err != nil {
					log.Printf("failed to encrypt API secret for %s: %v", platform, err)
					return fmt.Errorf("failed to encrypt API secret for %s: %w", platform, err)
				}
				encryptedAuthToken, err := common.EncryptAES(string(cfg.AuthToken))
				if err != nil {
					log.Printf("failed to encrypt Auth token for %s: %v", platform, err)
					return fmt.Errorf("failed to encrypt Auth token for %s: %w", platform, err)
				}

				platformConfigs[platform] = entities.AccountingPlatformConfig{
					OrgID:      cfg.OrgID,
					HMACSecret: common.EncryptedString(encryptedHMACSecret),
					AuthToken:  common.EncryptedString(encryptedAuthToken),
					APIKey:     common.EncryptedString(encryptedAPIKey),
					APISecret:  common.EncryptedString(encryptedAPISecret),
				}
			}

			user := entities.Business{
				ID:              utility.GenerateUUID(),
				Name:            userData.Name,
				Email:           userData.Email,
				Password:        userData.Password,
				APIKey:          common.EncryptedString(encryptedAPIKey),
				APIKeyHash:      apiKeyHashStr,
				PlatformConfigs: platformConfigs,
				AccStatus:       0,
				TIN:             userData.TIN,
				PhoneNumber:     userData.PhoneNumber,
				CompanyName:     userData.CompanyName,
				EmailVerified:   userData.EmailVerified,
				IsAggregator:    userData.IsAggregator,
			}

			err = s.businessRepo.CreateBusiness(&user, pDB)
			if err != nil {
				log.Println(err)
			}
		} else {
			return nil
		}
	} else {
		return nil
	}

	return nil
}

func (s *service) VerifyBusinessAccount(req VerifyEmailDto, isSandbox bool) (map[string]interface{}, error) {
	redisClient := redis.NewClient()
	ctx := redisClient.Context()
	db := s.repo.GetDB(isSandbox)

	email := strings.ToLower(strings.TrimSpace(req.Email))
	user := entities.Business{}
	queryError, err := db.SelectOneFromDb(&user, "email = ?", email)
	if err != nil {
		return nil, fmt.Errorf("account details cannot be retrieved")
	}

	if queryError != nil {
		return nil, fmt.Errorf("account details cannot be retrieved")
	}

	key := VerifyEmailKey(email)

	otp, err := redisClient.Get(ctx, key).Result()

	if err == mainRedis.Nil {
		return nil, errors.New("otp not found or has expired")
	}

	if err != nil {
		return nil, errors.New("unable to verify otp")
	}

	if otp != req.OTP {
		return nil, errors.New("invalid OTP provided")
	}

	user.EmailVerified = true
	if _, err := db.UpdateFields(user, user, user.ID); err != nil {
		return nil, fmt.Errorf("failed to update password: %w", err)
	}
	redisClient.Del(ctx, key)

	tokenData, err := middleware.CreateToken(user, isSandbox)
	if err != nil {
		return nil, fmt.Errorf("error saving token: %w", err)
	}
	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	accessToken := entities.AccessToken{ID: tokenData.AccessUuid, OwnerID: user.ID}

	err = s.repo.CreateAccessToken(&accessToken, isSandbox, tokens)

	if err != nil {
		return nil, fmt.Errorf("error saving token: %w", err)
	}

	responseData := map[string]interface{}{
		"data": UserResponse{
			ID:           user.ID,
			Email:        user.Email,
			Name:         user.Name,
			BusinessID:   user.BusinessID,
			ServiceID:    user.ServiceID,
			IsSandbox:    isSandbox,
			IsAggregator: user.IsAggregator,
			KeysSet:      user.KeysSet,
		},
		"access_token": tokenData.AccessToken,
	}

	return responseData, nil
}

func (s *service) VerifyProdBuisnessAccount(req VerifyEmailDto) error {
	db := s.repo.GetDB(false)

	email := strings.ToLower(strings.TrimSpace(req.Email))
	user := entities.Business{}
	queryError, err := db.SelectOneFromDb(&user, "email = ?", email)
	if err != nil {
		return fmt.Errorf("account details cannot be retrieved")
	}

	if queryError != nil {
		return fmt.Errorf("account details cannot be retrieved")
	}

	user.EmailVerified = true
	if _, err := db.UpdateFields(user, user, user.ID); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

func (s *service) SendOtp(email, key string) {
	redisClient := redis.NewClient()
	ctx := redisClient.Context()
	otp, _ := utility.GenerateOTP(6)

	// otp := 123456 // For testing purposes only, replace with generated OTP
	duration := 15 * time.Minute // 15 minutes expiration

	err := redisClient.Set(ctx, key, strconv.Itoa(otp), duration)
	log.Println("possible error: ", err)
	resend_email.Send(email, strconv.Itoa(otp))
}
