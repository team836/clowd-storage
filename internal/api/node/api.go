package node

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/team836/clowd-storage/internal/model"
	"github.com/team836/clowd-storage/pkg/logger"
)

var (
	// TODO: CheckOrigin option must be changed because too open
	// See this link(https://godoc.org/github.com/gorilla/websocket#hdr-Origin_Considerations)
	upgrader = websocket.Upgrader{
		ReadBufferSize:  512,
		WriteBufferSize: 512,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func RegisterHandlers(group *echo.Group) {
	group.GET("", openWebsocket)
}

/**
Handle the websocket requests from the clowder.
*/
func openWebsocket(ctx echo.Context) error {
	clowderModel := ctx.Get("clowder").(*model.Clowder) // get current clowder model

	machineID := ctx.QueryParam("mid") // get clowder machine id
	if machineID == "" {
		return ctx.String(http.StatusBadRequest, "Cannot find the machine id at the query parameters")
	}

	// find duplicate clowder connection and unregistered old one
	for clowder := range Pool().clowders {
		if clowder.machineID == machineID {
			Pool().unregister <- clowder
			break
		}
	}

	// upgrade to websocket protocol
	conn, err := upgrader.Upgrade(ctx.Response(), ctx.Request(), nil)
	if err != nil {
		logger.File().Infof("Error upgrading to websocket, %s", err)
		return err
	}

	// create new clowder
	clowder := NewClowder(machineID, conn, clowderModel)

	go clowder.run()           // run the websocket operations
	Pool().register <- clowder // register this clowder to pool

	return nil
}
