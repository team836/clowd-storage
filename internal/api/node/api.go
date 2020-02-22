package node

import (
	"net/http"

	"github.com/team836/clowd-storage/internal/module/cwdr"

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
Handle the websocket requests from the node.
*/
func openWebsocket(ctx echo.Context) error {
	nodeModel := ctx.Get("node").(*model.Node) // get current node model

	// find duplicate node connection and unregistered old one
	for node := range cwdr.Pool().Nodes {
		if node.Model.MachineID == nodeModel.MachineID {
			cwdr.Pool().Unregister <- node
			break
		}
	}

	// upgrade to websocket protocol
	conn, err := upgrader.Upgrade(ctx.Response(), ctx.Request(), nil)
	if err != nil {
		logger.File().Infof("Error upgrading to websocket, %s", err)
		return err
	}

	// create new node
	node := cwdr.NewNode(conn, nodeModel)

	go node.Run()                // run the websocket operations
	cwdr.Pool().Register <- node // register this node to pool

	return nil
}
