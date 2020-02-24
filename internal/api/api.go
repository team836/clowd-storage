package api

import (
	"github.com/labstack/echo/v4"
	"github.com/team836/clowd-storage/internal/api/client"
	"github.com/team836/clowd-storage/internal/api/node"
	"github.com/team836/clowd-storage/internal/middleware"
	"github.com/team836/clowd-storage/internal/middleware/auth"
)

func RegisterHandlers(group *echo.Group) {
	nodeGroup := group.Group("/node", auth.AuthenticateClowder, middleware.PrepareNodeModel)
	node.RegisterHandlers(nodeGroup)

	clientGroup := group.Group("/client", auth.AuthenticateClowdee)
	client.RegisterHandlers(clientGroup)
}
