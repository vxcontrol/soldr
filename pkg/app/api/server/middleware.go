package server

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"soldr/pkg/app/api/models"
	"soldr/pkg/app/api/server/response"
	"soldr/pkg/app/api/utils/dbencryptor"
	obs "soldr/pkg/observability"
)

func localUserRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		roleID := c.GetUint64("rid")
		if roleID == models.RoleExternal {
			response.Error(c, response.ErrLocalUserRequired, nil)
			return
		}

		c.Next()
	}
}

func inconcurrentRequest() gin.HandlerFunc {
	var reqLock sync.Mutex
	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		reqLock.Lock()
		defer reqLock.Unlock()

		c.Next()
	}
}

func setGlobalDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		c.Set("gDB", db)
		c.Next()
	}
}

func setSecureConfigEncryptor() gin.HandlerFunc {
	encryptor := dbencryptor.NewSecureConfigEncryptor(dbencryptor.GetKey)

	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		c.Set("crp", encryptor)
		c.Next()
	}
}

// Deprecated
func setServiceInfo(db *gorm.DB) gin.HandlerFunc {
	var mu sync.Mutex
	serviceCache := make(map[uint64]*models.Service)

	getService := func(c *gin.Context) (*models.Service, error) {
		mu.Lock()
		defer mu.Unlock()

		sid := c.GetUint64("sid")
		if sid == 0 {
			return nil, errors.New("sid cannot be 0")
		}

		service, ok := serviceCache[sid]
		if !ok {
			var s models.Service
			if err := db.Take(&s, "id = ?", sid).Error; err != nil {
				return nil, fmt.Errorf("could not fetch service: %w", err)
			}
			serviceCache[sid] = &s
		}
		return service, nil
	}

	loadServices := func() {
		var svs []models.Service
		if err := db.Find(&svs).Error; err == nil {
			for idx := range svs {
				s := svs[idx]
				serviceCache[s.ID] = &s
			}
		}
	}
	loadServices()

	return func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		service, err := getService(c)
		if err != nil {
			response.Error(c, response.ErrInternalServiceNotFound, nil)
			return
		}

		c.Set("SV", service)
		c.Next()
	}
}

func WithLogger(service string) gin.HandlerFunc {
	propagators := otel.GetTextMapPropagator()
	return func(c *gin.Context) {
		start := time.Now()
		uri := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if raw != "" {
			uri = uri + "?" + raw
		}

		savedCtx := c.Request.Context()
		defer func() {
			c.Request = c.Request.WithContext(savedCtx)
		}()

		ctx := propagators.Extract(savedCtx, propagation.HeaderCarrier(c.Request.Header))
		opts := []oteltrace.SpanStartOption{
			oteltrace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", c.Request)...),
			oteltrace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(c.Request)...),
			oteltrace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(service, c.FullPath(), c.Request)...),
		}

		entry := logrus.WithFields(logrus.Fields{
			"component":      "api",
			"net_peer_ip":    c.ClientIP(),
			"http_uri":       uri,
			"http_path":      c.Request.URL.Path,
			"http_host_name": c.Request.Host,
			"http_method":    c.Request.Method,
		})
		spanName := c.FullPath()
		if spanName == "" {
			spanName = fmt.Sprintf("proxy request with method %s", c.Request.Method)
			entry = entry.WithField("request", "proxy handled")
		} else {
			entry = entry.WithField("request", "api handled")
		}

		ctx, span := obs.Observer.NewSpan(ctx, obs.SpanKindServer, spanName, opts...)
		defer span.End()

		// pass the span through the request context
		c.Request = c.Request.WithContext(ctx)

		// serve the request to the next middleware
		c.Next()

		status := c.Writer.Status()
		attrs := semconv.HTTPAttributesFromHTTPStatusCode(status)
		spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(status)
		span.SetAttributes(attrs...)
		span.SetStatus(spanStatus, spanMessage)
		if len(c.Errors) > 0 {
			span.SetAttributes(attribute.String("gin.errors", c.Errors.String()))
		}

		entry = entry.WithFields(logrus.Fields{
			"duration":         time.Since(start),
			"http_status_code": c.Writer.Status(),
			"http_resp_size":   c.Writer.Size(),
		}).WithContext(ctx)
		if spanStatus == codes.Error {
			entry.Error("http request handled error")
		} else {
			entry.Info("http request handled success")
		}
	}
}
