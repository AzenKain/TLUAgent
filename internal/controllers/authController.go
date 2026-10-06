package controllers

import (
	"crypto/subtle"
	"net/http"
	"time"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/config"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/crypto"
	"tluagent-web/pkg/validator"
)

type AuthController struct {
	authService services.AuthService
}

func NewAuthController(authService services.AuthService) *AuthController {
	return &AuthController{authService: authService}
}

func (c *AuthController) setAuthCookies(w http.ResponseWriter, res *response.AuthResponse) {
	secure := config.CookieSecureMode()

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    res.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(constants.AccessTokenDuration / time.Second),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    res.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(constants.RefreshTokenDuration / time.Second),
	})

	csrfToken, err := crypto.GenerateCSRFToken()
	if err == nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "csrf_token",
			Value:    csrfToken,
			Path:     "/",
			HttpOnly: false,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(constants.RefreshTokenDuration / time.Second),
		})
	}
}

func (c *AuthController) clearAuthCookies(w http.ResponseWriter) {
	secure := config.CookieSecureMode()

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		MaxAge:   -1,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		MaxAge:   -1,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		MaxAge:   -1,
	})
}

func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.SignInDto
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	res, err := c.authService.Signin(ctx, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	c.setAuthCookies(w, res)

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Signed in successfully",
		Data:    res,
	})
}

func (c *AuthController) SetupStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	required := c.authService.SetupRequired(ctx)
	if !required {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrNotFound, "Not found"))
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data: map[string]bool{
			"required": true,
		},
	})
}

func (c *AuthController) SubmitSetup(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	if requiredToken := config.GetConfigWithDefault("SETUP_TOKEN", ""); requiredToken != "" {
		provided := r.Header.Get("X-Setup-Token")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(requiredToken)) != 1 {
			apperrors.HandleError(w, apperrors.New(apperrors.ErrForbidden, "Invalid or missing setup token"))
			return
		}
	}

	var dto request.SetupDto
	if err := validator.ValidateBodyDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	res, err := c.authService.SubmitSetup(ctx, &dto)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	c.setAuthCookies(w, res)

	writeJSONResponse(w, http.StatusCreated, response.CommonResponse{
		Status:  true,
		Message: "Setup completed successfully",
		Data:    res,
	})
}

func (c *AuthController) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	uid := getUserID(r)
	if uid == "" || uid == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	refreshToken := middlewares.GetRawRefreshToken(r.Context())
	if refreshToken == "" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Missing refresh token"))
		return
	}

	res, err := c.authService.RefreshToken(ctx, uid, refreshToken)
	if err != nil {
		c.clearAuthCookies(w)
		apperrors.HandleError(w, err)
		return
	}

	c.setAuthCookies(w, res)

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Token refreshed successfully",
		Data:    res,
	})
}

func (c *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	uid := getUserID(r)
	if uid == "" || uid == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	_ = c.authService.Logout(ctx, uid)

	c.clearAuthCookies(w)

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Logged out successfully",
	})
}

func (c *AuthController) Me(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	uid := getUserID(r)
	if uid == "" || uid == "0" {
		apperrors.HandleError(w, apperrors.New(apperrors.ErrUnauthorized, "Unauthorized"))
		return
	}

	user, err := c.authService.GetMe(ctx, uid)
	if err != nil {
		apperrors.HandleError(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "User profile fetched",
		Data:    user,
	})
}
