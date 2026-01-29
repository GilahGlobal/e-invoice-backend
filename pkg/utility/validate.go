package utility

import (
	"io"
	"net/mail"
	"os"
	"reflect"
	"regexp"
	"time"

	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/go-playground/validator/v10"
	"github.com/microcosm-cc/bluemonday"
	"github.com/nyaruka/phonenumbers"
)

func EmailValid(email string) (string, bool) {
	// made some change to parse the formated email
	e, err := mail.ParseAddress(email)
	if err != nil {
		return "", false
	}
	return e.Address, err == nil
}

func PhoneValid(phone string) (string, bool) {
	parsed, err := phonenumbers.Parse(phone, "")
	if err != nil {
		return phone, false
	}

	if !phonenumbers.IsValidNumber(parsed) {
		return phone, false
	}

	formattedNum := phonenumbers.Format(parsed, phonenumbers.NATIONAL)
	return formattedNum, true
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func CleanStringInput(input string) string {
	policy := bluemonday.UGCPolicy()
	cleanedInput := policy.Sanitize(input)
	re := regexp.MustCompile(`[^\w\s]`)
	cleanedInput = re.ReplaceAllString(cleanedInput, "")

	return cleanedInput
}

// VerifyWebhookSignature verifies the Zoho webhook signature
func VerifyWebhookSignature(body []byte, secret, signature string) bool {
	if secret == "" || signature == "" {
		return false
	}
	hash := hmac.New(sha256.New, []byte(secret))
	hash.Write(body)
	expected := hex.EncodeToString(hash.Sum(nil))
	return signature == expected
}

func DecodeJSONWithDefaults(r io.Reader, v interface{}) error {
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return err
	}
	cleanEmptyValues(reflect.ValueOf(v))
	return nil
}

func cleanEmptyValues(v reflect.Value) bool {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return true // nil pointer stays nil
		}
		if cleanEmptyValues(v.Elem()) {
			// If child is empty → set parent pointer to nil
			v.Set(reflect.Zero(v.Type()))
			return true
		}
		return false
	}

	switch v.Kind() {
	case reflect.Struct:
		allEmpty := true
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if !cleanEmptyValues(field) {
				allEmpty = false
			}
		}
		return allEmpty

	case reflect.Slice:
		if v.IsNil() || v.Len() == 0 {
			return true
		}
		allEmpty := true
		for i := 0; i < v.Len(); i++ {
			if !cleanEmptyValues(v.Index(i)) {
				allEmpty = false
			}
		}
		return allEmpty

	case reflect.Map:
		if v.IsNil() || len(v.MapKeys()) == 0 {
			return true
		}
		allEmpty := true
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			if !cleanEmptyValues(val) {
				allEmpty = false
			}
		}
		return allEmpty

	case reflect.String:
		return v.Len() == 0

	case reflect.Interface:
		if v.IsNil() {
			return true
		}
		return cleanEmptyValues(v.Elem())

	default:
		// Numbers, bools, etc.: 0 = empty, else not
		z := reflect.Zero(v.Type())
		return reflect.DeepEqual(v.Interface(), z.Interface())
	}
}

func IsValidNRSDate(fl validator.FieldLevel) bool {
	_, err := time.Parse("2006-01-02", fl.Field().String())
	return err == nil
}
