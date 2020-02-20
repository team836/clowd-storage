package auth

import (
	"net/http"

	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
)

// jwtCustomClaims are custom claims extending default ones.
type JwtCustomClaims struct {
	ID   string `json:"sub"`
	Name string `json:"aud"`
	jwt.StandardClaims
}

// JWT authenticate config
func Jwtconfig() *middleware.JWTConfig {
	return &middleware.JWTConfig{
		Claims:     &JwtCustomClaims{},
		SigningKey: []byte(viper.GetString("JWT.SECRET"))}
}

// Document to Chihoon
func CheckUser(c echo.Context) error {
	// Get claims from the header
	user := c.Get("user").(*jwt.Token)

	// Make claims to Custom Claims
	claims := user.Claims.(*JwtCustomClaims)

	// Put value to variable from Custom Claims
	name := claims.Name
	id := claims.ID
	return c.String(http.StatusOK, "Welcome "+name+"!\n ID: "+id)
}
