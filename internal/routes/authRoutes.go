package routes

import (
	"net/http"
	"time"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
)

func RegisterAuthRoutes(mux *http.ServeMux, authCtrl *controllers.AuthController, userRepo repositories.UserRepository) {
	jwtAccess := middlewares.JWTAccess(userRepo)
	jwtRefresh := middlewares.JWTRefresh(userRepo)

	// Auth rate limiter to protect against brute-force attacks (30 attempts / minute per IP)
	authRateLimiter := middlewares.NewRateLimiter(30, time.Minute)

	mux.HandleFunc("GET /api/setup/status", authCtrl.SetupStatus)
	mux.Handle("POST /api/setup", authRateLimiter.Middleware()(http.HandlerFunc(authCtrl.SubmitSetup)))
	mux.Handle("POST /api/auth/login", authRateLimiter.Middleware()(http.HandlerFunc(authCtrl.Login)))
	mux.Handle("POST /api/auth/refresh", jwtRefresh(http.HandlerFunc(authCtrl.Refresh)))
	mux.Handle("POST /api/auth/logout", jwtAccess(http.HandlerFunc(authCtrl.Logout)))
	mux.Handle("GET /api/auth/me", jwtAccess(http.HandlerFunc(authCtrl.Me)))
}

