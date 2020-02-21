package auth

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
	"github.com/team836/clowd-storage/internal/model"
	"github.com/team836/clowd-storage/pkg/database"
)

// JWTConfig authentication config
var JWTConfig = middleware.JWTConfig{
	Claims:     &JWTCustomClaims{},
	SigningKey: []byte(viper.GetString("JWT.SECRET")),
}

/**
JWTCustomClaims are custom claims extending default ones.
*/
type JWTCustomClaims struct {
	UserID   string `json:"sub"`
	UserName string `json:"aud"`
	jwt.StandardClaims
}

/**
Middleware for authenticating and returning the clowder.
*/
func AuthenticateClowder(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		userID := getUserID(ctx)
		clowder := &model.Clowder{}
		database.Conn().First(clowder, userID)
		ctx.Set("clowder", clowder)

		return next(ctx)
	}
}

/**
Middleware for authenticating and returning the clowdee.
*/
func AuthenticateClowdee(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		userID := getUserID(ctx)
		clowdee := &model.Clowdee{}
		database.Conn().First(clowdee, userID)
		ctx.Set("clowdee", clowdee)

		return next(ctx)
	}
}

/**
Get user id from the context.
*/
func getUserID(ctx echo.Context) string {
	// get claims from the header
	token := ctx.Get("user").(*jwt.Token)
	claims := token.Claims.(*JWTCustomClaims)

	return claims.UserID
}
