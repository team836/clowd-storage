package auth

import (
	"net/http"

	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
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

// Document to Chihoon
func CheckUser(c echo.Context) error {
	// Get claims from the header
	user := c.Get("user").(*jwt.Token)

	// Make claims to Custom Claims
	claims := user.Claims.(*JWTCustomClaims)

	// Put value to variable from Custom Claims
	name := claims.Name
	id := claims.ID
	return c.String(http.StatusOK, "Welcome "+name+"!\n ID: "+id)
}
