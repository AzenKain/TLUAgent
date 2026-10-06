package validator

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"

	"tluagent-web/pkg/jsonx"
)

var validate = validator.New()

func init() {
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name == "" {
			name = strings.SplitN(fld.Tag.Get("query"), ",", 2)[0]
		}
		return name
	})

	_ = validate.RegisterValidation("image_url", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		if value == "" {
			return true
		}
		return isImageURL(value)
	})

	_ = validate.RegisterValidation("server_url", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		if value == "" {
			return true
		}
		return isServerURL(value)
	})

	_ = validate.RegisterValidation("password_policy", func(fl validator.FieldLevel) bool {
		return hasLettersAndDigits(fl.Field().String())
	})

	_ = validate.RegisterValidation("base_url", func(fl validator.FieldLevel) bool {
		return isBaseURL(fl.Field().String())
	})

	_ = validate.RegisterValidation("readlist_cursor", func(fl validator.FieldLevel) bool {
		return isReadListCursor(fl.Field().String())
	})
}

func hasLettersAndDigits(value string) bool {
	if utf8.RuneCountInString(value) < 10 {
		return false
	}
	var hasLetter, hasDigit bool
	for _, r := range value {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}

func isBaseURL(value string) bool {
	text := strings.TrimSpace(value)
	if text == "" {
		return true
	}
	if strings.ContainsAny(text, "\r\n") {
		return false
	}
	parsed, err := url.Parse(text)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	if parsed.Host == "" {
		return false
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	return true
}

func isServerURL(value string) bool {
	text := strings.TrimSpace(value)
	if text == "" {
		return true
	}
	if strings.ContainsAny(text, "\r\n") {
		return false
	}
	parsed, err := url.Parse(text)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	if parsed.Host == "" {
		return false
	}
	if strings.Trim(parsed.Path, "/") != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	return true
}

func isImageURL(value string) bool {
	if strings.HasPrefix(value, "/public/") {
		switch strings.ToLower(path.Ext(value)) {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp":
			return true
		default:
			return false
		}
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return false
	}
	if scheme := strings.ToLower(parsed.Scheme); scheme != "http" && scheme != "https" {
		return false
	}
	switch strings.ToLower(path.Ext(parsed.Path)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		return false
	}
}

func isReadListCursor(value string) bool {
	if value == "" {
		return true
	}
	parts := strings.SplitN(value, "|", 2)
	_, err := time.Parse(time.RFC3339Nano, parts[0])
	return err == nil
}

type ErrorResponse struct {
	FailedField string `json:"failed_field,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Value       string `json:"value,omitempty"`
	Message     string `json:"message"`
}

func formatValidationError(err error) []*ErrorResponse {
	var validationErrors validator.ValidationErrors
	errorsList := []*ErrorResponse{}

	if errors.As(err, &validationErrors) {
		for _, fieldError := range validationErrors {
			message := ""
			switch fieldError.Tag() {
			case "required":
				message = fieldError.Field() + " is mandatory"
			case "email":
				message = "The email address is invalid"
			case "min":
				message = fieldError.Field() + " is too short (min " + fieldError.Param() + ")"
			case "max":
				message = fieldError.Field() + " is too long (max " + fieldError.Param() + ")"
			case "url":
				message = fieldError.Field() + " must be a valid URL"
			case "server_url":
				message = fieldError.Field() + " must be a valid http or https URL"
			case "base_url":
				message = fieldError.Field() + " must be a valid http or https base URL without query or fragment"
			case "password_policy":
				message = fieldError.Field() + " must be at least 10 characters and contain both letters and numbers"
			case "image_url":
				message = fieldError.Field() + " must be a link to an image"
			case "readlist_cursor":
				message = fieldError.Field() + " is an invalid cursor format"
			case "oneof":
				message = fieldError.Field() + " must be one of: " + fieldError.Param()
			case "uuid":
				message = fieldError.Field() + " must be a valid UUID"
			case "numeric":
				message = fieldError.Field() + " must be numeric"
			case "len":
				message = fieldError.Field() + " must be exactly " + fieldError.Param() + " characters"
			case "gte":
				message = fieldError.Field() + " must be at least " + fieldError.Param()
			case "lte":
				message = fieldError.Field() + " must be at most " + fieldError.Param()
			default:
				message = "Field " + fieldError.Field() + " failed validation: " + fieldError.Tag()
			}
			errorsList = append(errorsList, &ErrorResponse{
				FailedField: fieldError.Field(),
				Tag:         fieldError.Tag(),
				Value:       fieldError.Param(),
				Message:     message,
			})
		}
	} else {
		errorsList = append(errorsList, &ErrorResponse{
			Message: "Invalid request payload: " + err.Error(),
		})
	}

	return errorsList
}

// FormatValidationError is an exported alias for formatValidationError
func FormatValidationError(err error) []*ErrorResponse {
	return formatValidationError(err)
}

func decodeQueryValue(values url.Values, v reflect.Value) error {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := t.Field(i)

		if fieldType.Anonymous && fieldVal.Kind() == reflect.Struct {
			if err := decodeQueryValue(values, fieldVal); err != nil {
				return err
			}
			continue
		}

		if !fieldVal.CanSet() {
			continue
		}

		tag := fieldType.Tag.Get("query")
		if tag == "" || tag == "-" {
			tag = fieldType.Tag.Get("json")
			tag = strings.SplitN(tag, ",", 2)[0]
		}
		if tag == "" || tag == "-" {
			continue
		}

		paramVal := values.Get(tag)
		if paramVal == "" {
			continue
		}

		switch fieldVal.Kind() {
		case reflect.String:
			fieldVal.SetString(paramVal)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if n, err := strconv.ParseInt(paramVal, 10, 64); err == nil {
				fieldVal.SetInt(n)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if n, err := strconv.ParseUint(paramVal, 10, 64); err == nil {
				fieldVal.SetUint(n)
			}
		case reflect.Bool:
			if b, err := strconv.ParseBool(paramVal); err == nil {
				fieldVal.SetBool(b)
			}
		case reflect.Pointer:
			switch fieldVal.Type().Elem().Kind() {
			case reflect.Bool:
				if b, err := strconv.ParseBool(paramVal); err == nil {
					fieldVal.Set(reflect.ValueOf(&b))
				}
			case reflect.String:
				fieldVal.Set(reflect.ValueOf(&paramVal))
			case reflect.Int:
				if n, err := strconv.Atoi(paramVal); err == nil {
					fieldVal.Set(reflect.ValueOf(&n))
				}
			}
		case reflect.Float32, reflect.Float64:
			if f, err := strconv.ParseFloat(paramVal, 64); err == nil {
				fieldVal.SetFloat(f)
			}
		case reflect.Slice:
			if fieldVal.Type().Elem().Kind() == reflect.String {
				allVals := values[tag]
				fieldVal.Set(reflect.ValueOf(allVals))
			}
		}
	}
	return nil
}

func decodeQuery(values url.Values, target any) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.New("target must be a pointer to a struct")
	}
	return decodeQueryValue(values, v.Elem())
}

func ValidateQueryDto(r *http.Request, dto any) []*ErrorResponse {
	if r != nil && r.URL != nil {
		if err := decodeQuery(r.URL.Query(), dto); err != nil {
			return []*ErrorResponse{{Message: "Invalid query parameters: " + err.Error()}}
		}
	}
	if err := validate.Struct(dto); err != nil {
		return formatValidationError(err)
	}
	return nil
}

func ValidateBodyDto(r *http.Request, dto any) []*ErrorResponse {
	if r.Body == nil {
		return []*ErrorResponse{{Message: "Request body is empty"}}
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return []*ErrorResponse{{Message: "Failed to read request body: " + err.Error()}}
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	if len(body) == 0 {
		return []*ErrorResponse{{Message: "Request body cannot be empty"}}
	}

	if err := jsonx.Unmarshal(body, dto); err != nil {
		return []*ErrorResponse{{Message: "Invalid JSON format: " + err.Error()}}
	}

	if err := validate.Struct(dto); err != nil {
		return formatValidationError(err)
	}

	return nil
}
