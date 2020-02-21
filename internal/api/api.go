package api

import (
	"github.com/labstack/echo/v4"
	"github.com/team836/clowd-storage/internal/api/client"
	"github.com/team836/clowd-storage/internal/api/node"
)

func RegisterHandlers(group *echo.Group) {
	nodeGroup := group.Group("/node")
	node.RegisterHandlers(nodeGroup)

	clientGroup := group.Group("/client")
	client.RegisterHandlers(clientGroup)
}
