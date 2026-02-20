package config

type FIRS struct {
	FirsApiUrl    string
	FirsApiKey    string
	FirsClientKey string
	FirsPublicKey string
	FirsCertKey   string
}
type ZOHO struct {
	ZohoApiUrl string
}

type S3 struct {
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Region          string
}
