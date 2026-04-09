package aggregator

import (
	"einvoice-access-point/internal/dtos"
	aggregatorRepo "einvoice-access-point/internal/repository/aggregator"
	authRepo "einvoice-access-point/internal/repository/auth"
	"einvoice-access-point/pkg/database/redis"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/middleware"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/ses"
	"einvoice-access-point/pkg/utility"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

func verifyEmailKey(email string) string {
	return "verify_aggregator_email_otp_" + strings.ToLower(strings.TrimSpace(email))
}

func ValidateRegisterRequest(req dtos.AggregatorRegisterDto, testDb, prodDb *gorm.DB) (dtos.AggregatorRegisterDto, error) {
	if req.Email != "" {
		req.Email = strings.ToLower(req.Email)
		formattedMail, checkBool := utility.EmailValid(req.Email)
		if !checkBool {
			return req, fmt.Errorf("email address is invalid")
		}
		req.Email = formattedMail

		if aggregatorRepo.CheckAggregatorEmailExists(testDb, req.Email) {
			return req, errors.New("user already exists with the given email")
		}
		if aggregatorRepo.CheckAggregatorEmailExists(prodDb, req.Email) {
			return req, errors.New("user already exists with the given email")
		}
	}

	if aggregatorRepo.CheckAggregatorCompanyExists(testDb, req.CompanyName) {
		return req, errors.New("aggregator already exists with the given company name")
	}
	if aggregatorRepo.CheckAggregatorCompanyExists(prodDb, req.CompanyName) {
		return req, errors.New("aggregator already exists with the given company name")
	}

	return req, nil
}

func RegisterAggregator(req dtos.AggregatorRegisterDto, db *gorm.DB) (int, error) {
	email := strings.ToLower(req.Email)
	name := strings.Title(strings.ToLower(req.Name))

	password, err := utility.HashPassword(req.Password)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to hash password: %w", err)
	}

	aggregator := models.Aggregator{
		ID:          utility.GenerateUUID(),
		Name:        name,
		Email:       email,
		Password:    password,
		CompanyName: req.CompanyName,
		PhoneNumber: req.PhoneNumber,
	}

	err = aggregatorRepo.CreateAggregator(&aggregator, db)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to create aggregator: %w", err)
	}

	return http.StatusCreated, nil
}

func LoginAggregator(req dtos.AggregatorLoginDto, db *gorm.DB) (map[string]interface{}, int, error) {
	redisClient := redis.NewClient()
	ctx := redisClient.Context()

	aggregator, err := aggregatorRepo.GetAggregatorByEmail(db, req.Email)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("unable to fetch aggregator: %w", err)
	}
	if aggregator == nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid credentials")
	}

	if !utility.CompareHash(req.Password, aggregator.Password) {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid credentials")
	}

	if !aggregator.EmailVerified {
		otp := 123456 // For testing purposes only, replace with generated OTP
		key := verifyEmailKey(aggregator.Email)
		duration := 15 * time.Minute

		redisClient.Set(ctx, key, strconv.Itoa(otp), duration)
		return nil, http.StatusExpectationFailed, fmt.Errorf("email has not been verified, an otp has been sent to your mail, use it to verify your email")
	}

	if !aggregator.IsActive {
		return nil, http.StatusForbidden, fmt.Errorf("aggregator account is deactivated")
	}

	pdb := inst.InitDB(db, false)

	tokenData, err := middleware.CreateAggregatorToken(*aggregator, req.IsSandbox)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error creating token: %w", err)
	}
	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	accessToken := models.AccessToken{ID: tokenData.AccessUuid, OwnerID: aggregator.ID}
	err = authRepo.CreateAccessToken(&accessToken, pdb, tokens)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error saving token: %w", err)
	}

	responseData := map[string]interface{}{
		"data": dtos.AggregatorUserResponse{
			ID:          aggregator.ID,
			Email:       aggregator.Email,
			Name:        aggregator.Name,
			CompanyName: aggregator.CompanyName,
			IsSandbox:   req.IsSandbox,
		},
		"access_token": tokenData.AccessToken,
	}

	return responseData, http.StatusOK, nil
}

func LogoutAggregator(accessUuid, ownerID string, db *gorm.DB) (int, error) {
	pdb := inst.InitDB(db, false)

	accessToken := models.AccessToken{ID: accessUuid, OwnerID: ownerID}
	err := authRepo.RevokeAccessToken(&accessToken, pdb)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error revoking session: %w", err)
	}

	return http.StatusOK, nil
}

func VerifyAggregatorEmail(db *gorm.DB, req dtos.AggregatorVerifyEmailDto, isSandbox bool) (map[string]interface{}, error) {
	redisClient := redis.NewClient()
	ctx := redisClient.Context()

	email := strings.ToLower(strings.TrimSpace(req.Email))

	aggregator, err := aggregatorRepo.GetAggregatorByEmail(db, email)
	if err != nil || aggregator == nil {
		return nil, fmt.Errorf("account details cannot be retrieved")
	}

	key := verifyEmailKey(email)
	otp, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, errors.New("unable to verify token, error from redis: " + err.Error())
	}
	if otp != req.OTP {
		return nil, errors.New("invalid OTP provided")
	}

	aggregator.EmailVerified = true
	if err := aggregatorRepo.UpdateAggregator(aggregator, db); err != nil {
		return nil, fmt.Errorf("failed to verify email: %w", err)
	}
	redisClient.Del(ctx, key)

	pdb := inst.InitDB(db, false)

	tokenData, err := middleware.CreateAggregatorToken(*aggregator, isSandbox)
	if err != nil {
		return nil, fmt.Errorf("error creating token: %w", err)
	}
	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	accessToken := models.AccessToken{ID: tokenData.AccessUuid, OwnerID: aggregator.ID}
	err = authRepo.CreateAccessToken(&accessToken, pdb, tokens)
	if err != nil {
		return nil, fmt.Errorf("error saving token: %w", err)
	}

	responseData := map[string]interface{}{
		"data": dtos.AggregatorUserResponse{
			ID:          aggregator.ID,
			Email:       aggregator.Email,
			Name:        aggregator.Name,
			CompanyName: aggregator.CompanyName,
			IsSandbox:   isSandbox,
		},
		"access_token": tokenData.AccessToken,
	}

	return responseData, nil
}

func ResendVerificationOTP(db *gorm.DB, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))

	aggregator, err := aggregatorRepo.GetAggregatorByEmail(db, email)
	if err != nil || aggregator == nil {
		return fmt.Errorf("account details cannot be retrieved")
	}
	if aggregator.EmailVerified {
		return errors.New("email already verified")
	}

	SendAggregatorOtp(aggregator.Email)
	return nil
}

func SendAggregatorOtp(email string) {
	redisClient := redis.NewClient()
	ctx := redisClient.Context()

	otp := 123456 // For testing purposes only, replace with generated OTP
	key := verifyEmailKey(email)
	duration := 15 * time.Minute

	redisClient.Set(ctx, key, strconv.Itoa(otp), duration)
	ses.SendEmail(email, strconv.Itoa(otp))
}
