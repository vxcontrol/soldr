package protected

import (
	"encoding/json"
	"errors"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/sirupsen/logrus"

	logger2 "soldr/pkg/app/api/logger"
	"soldr/pkg/app/api/models"
	"soldr/pkg/app/api/server/response"
	srvevents "soldr/pkg/app/api/worker/events"
)

type PermissionsFilter func(*gin.Context, srvevents.EventChannelName) bool

// SubscribeHandler is a function to subscribe on notifications via WS connection
// @Summary Retrieve events via websocket connections on changing or creating or deleting instance entities by filter
// @Tags Notifications
// @Produce json
// @Param list query string true "list of events type to get from notification service (support of multiple choices)" default(all) Enums(all, create-agent, update-agent, delete-agent, create-group, update-group, delete-group, create-policy, update-policy, delete-policy, create-module, update-module, delete-module, create-group-to-policy, delete-group-to-policy)
// @Success 200 {object} response.successResp{} "fake response because here will be upgrade to websocket"
// @Failure 500 {object} response.errorResp "internal error on upgraging to websocket"
// @Router /notifications/subscribe/ [get]
func SubscribeHandler(exchanger *srvevents.Exchanger, permsFilter PermissionsFilter) func(c *gin.Context) {
	return func(c *gin.Context) {
		subscribeString := c.Query("list")
		w, r := c.Writer, c.Request
		logger := logger2.FromContext(c).WithField("component", "notifier").WithContext(r.Context())

		var subscribes []srvevents.EventChannelName
		for _, name := range strings.Split(subscribeString, ",") {
			name := srvevents.EventChannelName(strings.TrimSpace(name))
			if srvevents.ValidChannelName(name) && permsFilter(c, name) {
				subscribes = append(subscribes, name)
			}
		}

		if len(subscribes) == 0 {
			logger.Errorf("subscribes empty list after filtering: %s", subscribeString)
			response.Error(c, response.ErrNotificationsSubscribesEmptyList, nil)
			return
		}

		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			logger.WithError(err).Errorf("failed to updgrade ws conn")
			response.Error(c, response.ErrNotificationsUpgradeWSConnFail, err)
			return
		}

		svc, exists := c.Get("SV")
		if !exists {
			logger.Errorf("service not found in session: %#v", c.Keys)
			response.Error(c, response.ErrNotificationsServiceNotFound, nil)
			return
		}

		service, ok := svc.(*models.Service)
		if !ok {
			logger.Errorf("unexpected service value: %#v", svc)
			response.Error(c, response.ErrNotificationsInvalidServiceValue, nil)
			return
		}

		go handleWSConn(conn, service, logger, subscribes, exchanger)
	}
}

func handleWSConn(conn net.Conn, service *models.Service, logger *logrus.Entry,
	subscribes []srvevents.EventChannelName, exchanger *srvevents.Exchanger) {
	defer conn.Close()

	sub := exchanger.Subscribe(service.ID, subscribes...)
	defer exchanger.UnSubscribe(sub)

	connCloseChan := make(chan error)
	defer close(connCloseChan)

	//keep alive loop
	go func() {
		for {
			_, _, err := wsutil.ReadClientData(conn)
			if err != nil {
				connCloseChan <- err
				return
			}
		}
	}()

	for {
		select {
		case event := <-sub.C:
			msg, err := json.Marshal(event)
			if err != nil {
				logger.WithError(err).Error("failed to marshal event")
				continue
			}

			err = wsutil.WriteServerMessage(conn, ws.OpText, msg)
			if err != nil {
				var cErr wsutil.ClosedError
				if !errors.As(err, &cErr) {
					logger.WithError(err).Error("ws conn close with error")
				}

				return
			}

		case err := <-connCloseChan:
			var cErr wsutil.ClosedError
			if !errors.As(err, &cErr) {
				logger.Errorf("ws conn close with error: %v", err)
			}

			return
		}
	}
}
