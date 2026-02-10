package config

import (
	"encoding/json"
	"os"
)

type Configuration struct {
	Server       ServerConfiguration
	Database     Database
	TestDatabase Database
	App          App
	Firs         FIRS
	FirsSandbox  FIRS
	Zoho         ZOHO
	Mail         MAIL
	Redis        Redis
	S3           S3
}

type BaseConfig struct {
	SERVER_PORT                      string  `mapstructure:"SERVER_PORT"`
	SERVER_SECRET                    string  `mapstructure:"SERVER_SECRET"`
	MASTER_SECRET                    string  `mapstructure:"MASTER_ENCRYPTION_KEY"`
	DEFAULT_PASSWORD                 string  `mapstructure:"DEFAULT_PASSWORD"`
	SERVER_ACCESSTOKENEXPIREDURATION int     `mapstructure:"SERVER_ACCESSTOKENEXPIREDURATION"`
	REQUEST_PER_SECOND               float64 `mapstructure:"REQUEST_PER_SECOND"`
	TRUSTED_PROXIES                  string  `mapstructure:"TRUSTED_PROXIES"`
	EXEMPT_FROM_THROTTLE             string  `mapstructure:"EXEMPT_FROM_THROTTLE"`

	APP_NAME                string `mapstructure:"APP_NAME"`
	APP_MODE                string `mapstructure:"APP_MODE"`
	APP_URL                 string `mapstructure:"APP_URL"`
	MAGIC_LINK_DURATION     int    `mapstructure:"MAGIC_LINK_DURATION"`
	RESET_PASSWORD_DURATION int    `mapstructure:"RESET_PASSWORD_DURATION"`

	DB_HOST       string `mapstructure:"DB_HOST"`
	DB_PORT       string `mapstructure:"DB_PORT"`
	DB_CONNECTION string `mapstructure:"DB_CONNECTION"`
	TIMEZONE      string `mapstructure:"TIMEZONE"`
	SSLMODE       string `mapstructure:"SSLMODE"`
	USERNAME      string `mapstructure:"USERNAME"`
	PASSWORD      string `mapstructure:"PASSWORD"`
	DB_NAME       string `mapstructure:"DB_NAME"`
	MIGRATE       bool   `mapstructure:"MIGRATE"`

	TEST_DB_HOST       string `mapstructure:"TEST_DB_HOST"`
	TEST_DB_PORT       string `mapstructure:"TEST_DB_PORT"`
	TEST_DB_CONNECTION string `mapstructure:"TEST_DB_CONNECTION"`
	TEST_TIMEZONE      string `mapstructure:"TEST_TIMEZONE"`
	TEST_SSLMODE       string `mapstructure:"TEST_SSLMODE"`
	TEST_USERNAME      string `mapstructure:"TEST_USERNAME"`
	TEST_PASSWORD      string `mapstructure:"TEST_PASSWORD"`
	TEST_DB_NAME       string `mapstructure:"TEST_DB_NAME"`
	TEST_MIGRATE       bool   `mapstructure:"TEST_MIGRATE"`

	FIRS_API_URL    string `mapstructure:"FIRS_API_URL"`
	FIRS_API_KEY    string `mapstructure:"FIRS_API_KEY"`
	FIRS_CLIENT_KEY string `mapstructure:"FIRS_CLIENT_KEY"`
	FIRS_PUBLIC_KEY string `mapstructure:"FIRS_PUBLIC_KEY"`
	FIRS_CERT_KEY   string `mapstructure:"FIRS_CERT_KEY"`

	SANDBOX_FIRS_API_URL    string `mapstructure:"SANDBOX_FIRS_API_URL"`
	SANDBOX_FIRS_API_KEY    string `mapstructure:"SANDBOX_FIRS_API_KEY"`
	SANDBOX_FIRS_CLIENT_KEY string `mapstructure:"SANDBOX_FIRS_CLIENT_KEY"`
	SANDBOX_FIRS_PUBLIC_KEY string `mapstructure:"SANDBOX_FIRS_PUBLIC_KEY"`
	SANDBOX_FIRS_CERT_KEY   string `mapstructure:"SANDBOX_FIRS_CERT_KEY"`

	MAIL_SERVER   string `mapstructure:"MAIL_SERVER"`
	MAIL_PASSWORD string `mapstructure:"MAIL_PASSWORD"`
	MAIL_USERNAME string `mapstructure:"MAIL_USERNAME"`
	MAIL_PORT     string `mapstructure:"MAIL_PORT"`

	REDIS_PORT string `mapstructure:"REDIS_PORT"`
	REDIS_HOST string `mapstructure:"REDIS_HOST"`
	REDIS_DB   string `mapstructure:"REDIS_DB"`
	REDIS_URL  string `mapstructure:"REDIS_URL"`

	ZOHO_API_URL string `mapstructure:"ZOHO_API_URL"`

	S3_ACCESS_KEY_ID     string `mapstructure:"S3_ACCESS_KEY_ID"`
	S3_SECRET_ACCESS_KEY string `mapstructure:"S3_SECRET_ACCESS_KEY"`
	S3_BUCKET            string `mapstructure:"S3_BUCKET"`
	S3_REGION            string `mapstructure:"S3_REGION"`
}

func (config *BaseConfig) SetupConfigurationn() *Configuration {
	trustedProxies := []string{}
	exemptFromThrottle := []string{}
	json.Unmarshal([]byte(config.TRUSTED_PROXIES), &trustedProxies)
	json.Unmarshal([]byte(config.EXEMPT_FROM_THROTTLE), &exemptFromThrottle)

	if config.SERVER_PORT == "" {
		config.SERVER_PORT = os.Getenv("PORT")
	}
	return &Configuration{
		Server: ServerConfiguration{
			Port:                      config.SERVER_PORT,
			Secret:                    config.SERVER_SECRET,
			MasterSecret:              config.MASTER_SECRET,
			DefaultPassword:           config.DEFAULT_PASSWORD,
			AccessTokenExpireDuration: config.SERVER_ACCESSTOKENEXPIREDURATION,
			RequestPerSecond:          config.REQUEST_PER_SECOND,
			TrustedProxies:            trustedProxies,
			ExemptFromThrottle:        exemptFromThrottle,
		},
		App: App{
			Name:                  config.APP_NAME,
			Mode:                  config.APP_MODE,
			Url:                   config.APP_URL,
			MagicLinkDuration:     config.MAGIC_LINK_DURATION,
			ResetPasswordDuration: config.RESET_PASSWORD_DURATION,
		},
		Database: Database{
			DB_HOST:       config.DB_HOST,
			DB_PORT:       config.DB_PORT,
			DB_CONNECTION: config.DB_CONNECTION,
			USERNAME:      config.USERNAME,
			PASSWORD:      config.PASSWORD,
			TIMEZONE:      config.TIMEZONE,
			SSLMODE:       config.SSLMODE,
			DB_NAME:       config.DB_NAME,
			Migrate:       config.MIGRATE,
		},
		TestDatabase: Database{
			DB_HOST:       config.TEST_DB_HOST,
			DB_PORT:       config.TEST_DB_PORT,
			DB_CONNECTION: config.TEST_DB_CONNECTION,
			USERNAME:      config.TEST_USERNAME,
			PASSWORD:      config.TEST_PASSWORD,
			TIMEZONE:      config.TEST_TIMEZONE,
			SSLMODE:       config.TEST_SSLMODE,
			DB_NAME:       config.TEST_DB_NAME,
			Migrate:       config.TEST_MIGRATE,
		},

		Firs: FIRS{
			FirsApiUrl:    config.FIRS_API_URL,
			FirsApiKey:    config.FIRS_API_KEY,
			FirsClientKey: config.FIRS_CLIENT_KEY,
			FirsPublicKey: config.FIRS_PUBLIC_KEY,
			FirsCertKey:   config.FIRS_CERT_KEY,
		},

		FirsSandbox: FIRS{
			FirsApiUrl:    config.SANDBOX_FIRS_API_URL,
			FirsApiKey:    config.SANDBOX_FIRS_API_KEY,
			FirsClientKey: config.SANDBOX_FIRS_CLIENT_KEY,
			FirsPublicKey: config.SANDBOX_FIRS_PUBLIC_KEY,
			FirsCertKey:   config.SANDBOX_FIRS_CERT_KEY,
		},

		Zoho: ZOHO{
			ZohoApiUrl: config.ZOHO_API_URL,
		},

		Mail: MAIL{
			Server:   config.MAIL_SERVER,
			Password: config.MAIL_PASSWORD,
			Port:     config.MAIL_PORT,
			Username: config.MAIL_USERNAME,
		},

		Redis: Redis{
			REDIS_PORT: config.REDIS_PORT,
			REDIS_HOST: config.REDIS_HOST,
			REDIS_DB:   config.REDIS_DB,
			REDIS_URL:  config.REDIS_URL,
		},
		S3: S3{
			AccessKeyID:     config.S3_ACCESS_KEY_ID,
			SecretAccessKey: config.S3_SECRET_ACCESS_KEY,
			Bucket:          config.S3_BUCKET,
			Region:          config.S3_REGION,
		},
	}
}
