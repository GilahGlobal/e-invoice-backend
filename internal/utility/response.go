package utility

import (
	"einvoice-access-point/internal/data/entities"
	"fmt"
	"net/http"
	"reflect"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
)

func BuildSuccessResponse(code int, message string, data interface{}, pagination ...interface{}) entities.Response {
	res := ResponseMessage(code, "success", "", message, nil, data, pagination, nil)
	return res
}

func BuildErrorResponse(code int, status string, message string, err interface{}, data interface{}, logger ...bool) entities.Response {
	res := ResponseMessage(code, status, "", message, err, data, nil, nil)
	return res
}

func ResponseMessage(code int, status string, name string, message string, err interface{}, data interface{}, pagination interface{}, extra interface{}) entities.Response {
	if pagination != nil && reflect.ValueOf(pagination).IsNil() {
		pagination = nil
	}

	if code == http.StatusInternalServerError {
		fmt.Println("internal server error", message, err, data)
		message = "internal server error"
		err = message
	}

	res := entities.Response{
		StatusCode: code,
		Status:     status,
		Name:       name,
		Message:    message,
		Error:      err,
		Data:       data,
		Pagination: pagination,
		Extra:      extra,
	}
	return res
}

func UnauthorisedResponse(code int, status string, name string, message string) entities.Response {
	res := ResponseMessage(http.StatusUnauthorized, status, name, message, nil, nil, nil, nil)
	return res
}

func ValidationResponse(err error, validate *validator.Validate) validator.ValidationErrorsTranslations {
	errs := err.(validator.ValidationErrors)
	english := en.New()
	uni := ut.New(english, english)
	trans, _ := uni.GetTranslator("en")
	_ = enTranslations.RegisterDefaultTranslations(validate, trans)
	return errs.Translate(trans)
}
