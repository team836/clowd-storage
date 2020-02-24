package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/team836/clowd-storage/internal/model"
	"github.com/team836/clowd-storage/pkg/database"
	"github.com/team836/clowd-storage/pkg/logger"
)

/**
Middelware for node api group.
*/

/**
Middleware for preparing the node model.
*/
func PrepareNodeModel(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		clowderModel := ctx.Get("clowder").(*model.Clowder) // get current clowder model
		machineID := ctx.QueryParam("mid")                  // get clowder machine id
		if machineID == "" {
			return ctx.String(http.StatusBadRequest, "Cannot find the machine id at the query parameters")
		}

		// create node record if not exists
		node := &model.Node{}
		sqlResult := database.Conn().
			Where(&model.Node{MachineID: machineID}).
			Attrs(&model.Node{ClowderGoogleID: clowderModel.GoogleID}).
			FirstOrCreate(node)

		if sqlResult.Error != nil {
			logger.File().Errorf("Error preparing the node model, %s", sqlResult.Error.Error())
			return ctx.NoContent(http.StatusInternalServerError)
		}

		ctx.Set("node", node)

		return next(ctx)
	}
}
