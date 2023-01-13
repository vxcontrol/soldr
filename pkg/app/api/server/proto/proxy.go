package proto

import (
	"errors"
	"sync"

	"github.com/gorilla/websocket"
)

func linkPairSockets(ctxConn *ctxVXConnection, clientConn, serverConn IWSConn) error {
	var wg sync.WaitGroup
	errClient := make(chan error, 10)
	errServer := make(chan error, 10)
	replicateWebsocketConn := func(dst, src IWSConn, errc chan error) {
		defer wg.Done()
		for {
			msg, err := src.Read(ctxConn.ctx)
			if err != nil {
				errc <- err
				break
			}
			err = dst.Write(ctxConn.ctx, msg)
			if err != nil {
				errc <- err
				break
			}
		}
	}

	wg.Add(2)
	go replicateWebsocketConn(clientConn, serverConn, errClient)
	go replicateWebsocketConn(serverConn, clientConn, errServer)
	defer func() {
		clientConn.Close(ctxConn.ctx)
		serverConn.Close(ctxConn.ctx)
		ctxConn.logger.Debug("links proxy reader waits to stop connections")
		wg.Wait()
		ctxConn.logger.Debug("links proxy reader was exited")
	}()

	var (
		err     error
		message string
	)
	ctxConn.logger.Debug("links proxy reader was started")
	select {
	case <-ctxConn.ctx.Done():
		ctxConn.logger.Debug("links proxy reader was requested to stop by context")
		return nil
	case err = <-errClient:
		message = "proxy: error when copying from server to client"
	case err = <-errServer:
		message = "proxy: error when copying from client to server"
	}
	var closeErr *websocket.CloseError
	if !errors.Is(err, websocket.ErrCloseSent) && !(errors.As(err, &closeErr) && closeErr.Code == 1001) {
		ctxConn.logger.WithError(err).Error(message)
		return err
	}
	return nil
}
