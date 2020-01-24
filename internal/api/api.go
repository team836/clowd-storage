package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func RegisterHandlers(group *echo.Group) {
	group.GET("/test", test)
}

func test(ctx echo.Context) error {
	return ctx.String(http.StatusOK, "Hello clowd")
}
