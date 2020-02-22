package auth

import (
	"net/http"

	"github.com/team836/clowd-storage/pkg/logger"

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

		// find the clowder
		clowder := &model.Clowder{}
		sqlResult := database.Conn().Where(&model.Clowder{GoogleID: userID}).First(&clowder)

		if sqlResult.Error != nil {
			// if current user is not clowder
			if sqlResult.RecordNotFound() {
				return ctx.String(http.StatusUnauthorized, "Cannot authorize as clowder")
			}

			// other sql error
			logger.File().Errorf("Error finding the clowder in database, %s", sqlResult.Error.Error())
			return ctx.NoContent(http.StatusInternalServerError)
		}

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

		// find the clowdee
		clowdee := &model.Clowdee{}
		sqlResult := database.Conn().Where(&model.Clowdee{GoogleID: userID}).First(&clowdee)

		if sqlResult.Error != nil {
			// if current user is not clowdee
			if sqlResult.RecordNotFound() {
				return ctx.String(http.StatusUnauthorized, "Cannot authorize as clowdee")
			}

			// other sql error
			logger.File().Errorf("Error finding the clowdee in database, %s", sqlResult.Error.Error())
			return ctx.NoContent(http.StatusInternalServerError)
		}

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
