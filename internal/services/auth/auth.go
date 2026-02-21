package auth

import (
	"crypto/sha256"
	"einvoice-access-point/internal/dtos"
	authRepo "einvoice-access-point/internal/repository/auth"
	userRepo "einvoice-access-point/internal/repository/business"
	"einvoice-access-point/pkg/common"
	"einvoice-access-point/pkg/config"
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/database/redis"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ValidateCreateUserRequest(req dtos.RegisterDto, db *gorm.DB) (dtos.RegisterDto, error) {

	pdb := inst.InitDB(db, false)
	user := models.Business{}

	if req.Email != "" {
		req.Email = strings.ToLower(req.Email)
		formattedMail, checkBool := utility.EmailValid(req.Email)
		if !checkBool {
			return req, fmt.Errorf("email address is invalid")
		}
		req.Email = formattedMail
		exists := pdb.CheckExists(&user, "email = ?", req.Email)
		if exists {
			return req, errors.New("user already exists with the given email")
		}
	}
	if exists := pdb.CheckExists(&user, "company_name = ?", req.CompanyName); exists {
		return req, errors.New("Business already exists with the given company name")
	}

	return req, nil
}

func CreateUser(req dtos.RegisterDto, db *gorm.DB) (fiber.Map, int, error) {

	pdb := inst.InitDB(db, false)

	config := config.GetConfig()
	serverSecret := config.Server.Secret
	email := strings.ToLower(req.Email)
	name := strings.Title(strings.ToLower(req.Name))

	password, err := utility.HashPassword(req.Password)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to hash password: %w", err)
	}

	apiKey, err := utility.GenerateSecureToken(32, serverSecret)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to generate api key: %w", err)
	}
	encryptedAPIKey, err := common.EncryptAES(apiKey)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to encrypt API key: %w", err)
	}
	apiKeyHash := sha256.Sum256([]byte(apiKey))
	apiKeyHashStr := hex.EncodeToString(apiKeyHash[:])

	platformConfigs := models.PlatformConfigs{}
	for platform, cfg := range req.PlatformConfigs {
		encryptedHMACSecret, err := common.EncryptAES(string(cfg.HMACSecret))
		if err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("failed to encrypt HMAC secret for %s: %w", platform, err)
		}
		encryptedAPIKey, err := common.EncryptAES(string(cfg.APIKey))
		if err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("failed to encrypt API key for %s: %w", platform, err)
		}
		encryptedAPISecret, err := common.EncryptAES(string(cfg.APISecret))
		if err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("failed to encrypt API secret for %s: %w", platform, err)
		}
		encryptedAuthToken, err := common.EncryptAES(string(cfg.AuthToken))
		if err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("failed to encrypt Auth token for %s: %w", platform, err)
		}

		platformConfigs[platform] = models.AccountingPlatformConfig{
			OrgID:      cfg.OrgID,
			HMACSecret: common.EncryptedString(encryptedHMACSecret),
			AuthToken:  common.EncryptedString(encryptedAuthToken),
			APIKey:     common.EncryptedString(encryptedAPIKey),
			APISecret:  common.EncryptedString(encryptedAPISecret),
		}
	}

	user := models.Business{
		ID:              utility.GenerateUUID(),
		Name:            name,
		Email:           email,
		Password:        password,
		ServiceID:       "6A2BC898", //userRepo.GenerateUniqueServiceID(pdb.Db)
		APIKey:          common.EncryptedString(encryptedAPIKey),
		APIKeyHash:      apiKeyHashStr,
		PlatformConfigs: platformConfigs,
		AccStatus:       0,
		TIN:             req.TIN,
		PhoneNumber:     req.PhoneNumber,
		CompanyName:     req.CompanyName,
	}

	err = userRepo.CreateBusiness(&user, pdb)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to create business: %w", err)
	}

	responseData := fiber.Map{
		"id":          user.ID,
		"email":       user.Email,
		"name":        user.Name,
		"business_id": user.BusinessID,
		"service_id":  user.ServiceID,
		"tin":         user.TIN,
		"is_sandbox":  true,
	}

	return responseData, http.StatusCreated, nil
}
func LoginUser(req dtos.LoginRequestDto, db *gorm.DB) (map[string]interface{}, int, error) {

	pdb := inst.InitDB(db, false)
	var (
		user = models.Business{}
	)

	exists := pdb.CheckExists(&user, "email = ?", req.Email)
	if !exists {
		return nil, 400, fmt.Errorf("invalid credentials")
	}

	if !utility.CompareHash(req.Password, user.Password) {
		return nil, 400, fmt.Errorf("invalid credentials")
	}

	userData, err := userRepo.GetUserByEmail(pdb, req.Email)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("unable to fetch user: %w", err)
	}

	tokenData, err := middleware.CreateToken(user, req.IsSandbox)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error saving token: %w", err)
	}
	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	accessToken := models.AccessToken{ID: tokenData.AccessUuid, OwnerID: user.ID}

	err = authRepo.CreateAccessToken(&accessToken, pdb, tokens)

	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error saving token: %w", err)
	}

	responseData := map[string]interface{}{
		"data": dtos.UserResponse{
			ID:         userData.ID,
			Email:      userData.Email,
			Name:       userData.Name,
			BusinessID: userData.BusinessID,
			ServiceID:  userData.ServiceID,
			IsSandbox:  req.IsSandbox,
		},
		"access_token": tokenData.AccessToken,
	}

	return responseData, http.StatusOK, nil
}

func LogoutUser(accessUuid, ownerId string, db *gorm.DB) (fiber.Map, int, error) {

	pdb := inst.InitDB(db, false)
	var (
		responseData fiber.Map
	)

	accessToken := models.AccessToken{ID: accessUuid, OwnerID: ownerId}

	err := authRepo.RevokeAccessToken(&accessToken, pdb)
	if err != nil {
		return responseData, http.StatusInternalServerError, fmt.Errorf("error revoking user session: %w", err)
	}

	responseData = fiber.Map{}

	return responseData, http.StatusOK, nil
}

func InitiateForgotPassword(req dtos.InitiateForgotPasswordDto, db *gorm.DB) error {
	redisClient := redis.NewClient()
	ctx := redisClient.Context()
	pdb := inst.InitDB(db, false)
	var (
		user = models.Business{}
	)
	queryError, err := pdb.SelectOneFromDb(&user, "email = ?", req.Email)
	if err != nil {
		return fmt.Errorf("Account details cannot be retrieved")
	}

	if queryError != nil {
		return queryError
	}

	// otp, err := utility.GenerateOTP(6)
	// if err != nil {
	// 	return fmt.Errorf("failed to generate OTP: %w", err)
	// }

	otp := 123456 // For testing purposes only, replace with generated OTP
	key := "forgot_password_otp_" + user.ID
	duration := 15 * time.Minute // 15 minutes expiration

	redisClient.Set(ctx, key, strconv.Itoa(otp), duration)
	// Send otp to user's email - to be implemented
	return nil
}

func CompleteForgotPassword(req dtos.CompleteForgotPasswordDto, db *gorm.DB) error {
	redisClient := redis.NewClient()
	ctx := redisClient.Context()
	pdb := inst.InitDB(db, false)
	var (
		user = models.Business{}
	)

	queryError, err := pdb.SelectOneFromDb(&user, "email = ?", req.Email)
	if err != nil {
		return fmt.Errorf("Account details cannot be retrieved")
	}

	if queryError != nil {
		return queryError
	}

	key := "forgot_password_otp_" + user.ID

	otp, err := redisClient.Get(ctx, key).Result()

	log.Println(err)
	if err != nil {
		return errors.New("unable to verify token")
	}

	if otp != req.OTP {
		return errors.New("invalid OTP provided")
	}

	password, err := utility.HashPassword(req.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.Password = password
	pdb.UpdateFields(user, user, user.ID)
	redisClient.Del(ctx, key)
	return nil
}

func ToggleApllicationMode(db *gorm.DB, email string, isSandbox bool) (map[string]interface{}, int, error) {
	pdb := inst.InitDB(db, false)

	userData, err := userRepo.GetUserByEmail(pdb, email)
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

	accessToken := models.AccessToken{ID: tokenData.AccessUuid, OwnerID: userData.ID}

	err = authRepo.CreateAccessToken(&accessToken, pdb, tokens)

	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error saving token: %w", err)
	}

	responseData := map[string]interface{}{
		"data": dtos.UserResponse{
			ID:         userData.ID,
			Email:      userData.Email,
			Name:       userData.Name,
			BusinessID: userData.BusinessID,
			ServiceID:  userData.ServiceID,
			IsSandbox:  isSandbox,
		},
		"access_token": tokenData.AccessToken,
	}
	return responseData, http.StatusOK, nil
}

func SynchronizeSandboxToProduction(prodDB, sandboxDB *database.Database, email string) error {
	pDB := inst.InitDB(prodDB.Postgresql.DB(), false)
	sDB := inst.InitDB(sandboxDB.Postgresql.DB(), false)

	exists := pDB.CheckExistsInTable("businesses", "email = ?", email)

	if !exists {
		sandboxExists := sDB.CheckExistsInTable("businesses", "email = ?", email)
		if sandboxExists {
			userData, err := userRepo.GetUserByEmail(sDB, email)
			if err != nil {
				log.Println("unable to fetch user from sandbox: " + err.Error())
				return fmt.Errorf("unable to fetch user from sandbox: %w", err)
			}

			config := config.GetConfig()
			serverSecret := config.Server.Secret

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

			platformConfigs := models.PlatformConfigs{}
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

				platformConfigs[platform] = models.AccountingPlatformConfig{
					OrgID:      cfg.OrgID,
					HMACSecret: common.EncryptedString(encryptedHMACSecret),
					AuthToken:  common.EncryptedString(encryptedAuthToken),
					APIKey:     common.EncryptedString(encryptedAPIKey),
					APISecret:  common.EncryptedString(encryptedAPISecret),
				}
			}

			user := models.Business{
				ID:              utility.GenerateUUID(),
				Name:            userData.Name,
				Email:           userData.Email,
				Password:        userData.Password,
				ServiceID:       "6A2BC898", //userRepo.GenerateUniqueServiceID(pdb.Db)
				APIKey:          common.EncryptedString(encryptedAPIKey),
				APIKeyHash:      apiKeyHashStr,
				PlatformConfigs: platformConfigs,
				AccStatus:       0,
				TIN:             userData.TIN,
				PhoneNumber:     userData.PhoneNumber,
				CompanyName:     userData.CompanyName,
			}

			err = userRepo.CreateBusiness(&user, pDB)
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
