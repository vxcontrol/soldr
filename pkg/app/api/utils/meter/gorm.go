package meter

import (
	"context"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/unit"
)

var (
	createHist metric.Float64Histogram
	queryHist  metric.Float64Histogram
	updateHist metric.Float64Histogram
	deleteHist metric.Float64Histogram
)

func InitGormMetrics(meter metric.Meter) error {
	var err error
	createHist, err = meter.NewFloat64Histogram(
		"vxapi_gorm_create_queries_hist",
		metric.WithUnit(unit.Milliseconds),
		metric.WithDescription("create queries execution histogram"),
	)
	if err != nil {
		return err
	}
	updateHist, err = meter.NewFloat64Histogram(
		"vxapi_gorm_update_queries_hist",
		metric.WithUnit(unit.Milliseconds),
		metric.WithDescription("update queries execution histogram"),
	)
	if err != nil {
		return err
	}
	deleteHist, err = meter.NewFloat64Histogram(
		"vxapi_gorm_delete_queries_hist",
		metric.WithUnit(unit.Milliseconds),
		metric.WithDescription("delete queries execution histogram"),
	)
	if err != nil {
		return err
	}
	queryHist, err = meter.NewFloat64Histogram(
		"vxapi_gorm_select_queries_hist",
		metric.WithUnit(unit.Milliseconds),
		metric.WithDescription("select queries execution histogram"),
	)
	if err != nil {
		return err
	}
	return nil
}

func makeQueryLabel(scope *gorm.Scope) attribute.KeyValue {
	return attribute.KeyValue{
		Key:   "query",
		Value: attribute.StringValue(scope.SQL),
	}
}

var startTimes = map[string]time.Time{}
var mu = sync.Mutex{}

func recordStartTime(scope *gorm.Scope) {
	mu.Lock()
	defer mu.Unlock()

	startTimes[scope.InstanceID()] = time.Now()
}

func recordHistogram(scope *gorm.Scope, hist metric.Float64Histogram, labels ...attribute.KeyValue) {
	mu.Lock()
	defer mu.Unlock()

	if startQueryTs, exists := startTimes[scope.InstanceID()]; exists {
		ms := float64(time.Since(startQueryTs).Nanoseconds()) / 1e6
		hist.Record(context.Background(), ms, labels...)
		delete(startTimes, scope.InstanceID())
	}
}

func ApplyGorm(db *gorm.DB) {
	labels := []attribute.KeyValue{
		{Key: "database_name", Value: attribute.StringValue(db.Dialect().CurrentDatabase())},
		{Key: "dialect", Value: attribute.StringValue(db.Dialect().GetName())},
	}

	db.Callback().Create().Before("gorm:create").Register("metric:create-before", func(scope *gorm.Scope) {
		recordStartTime(scope)
	})
	db.Callback().Create().After("gorm:create").Register("metric:create-after", func(scope *gorm.Scope) {
		labels := append(labels, makeQueryLabel(scope))
		recordHistogram(scope, createHist, labels...)
	})

	db.Callback().Update().Before("gorm:update").Register("metric:update-before", func(scope *gorm.Scope) {
		recordStartTime(scope)
	})
	db.Callback().Update().After("gorm:update").Register("metric:update-after", func(scope *gorm.Scope) {
		labels := append(labels, makeQueryLabel(scope))
		recordHistogram(scope, updateHist, labels...)
	})

	db.Callback().Query().Before("gorm:query").Register("metric:query-before", func(scope *gorm.Scope) {
		recordStartTime(scope)
	})
	db.Callback().Query().After("gorm:query").Register("metric:query-after", func(scope *gorm.Scope) {
		labels := append(labels, makeQueryLabel(scope))
		recordHistogram(scope, queryHist, labels...)
	})

	db.Callback().Delete().Before("gorm:delete").Register("metric:delete-before", func(scope *gorm.Scope) {
		recordStartTime(scope)
	})
	db.Callback().Delete().After("gorm:delete").Register("metric:delete-after", func(scope *gorm.Scope) {
		labels := append(labels, makeQueryLabel(scope))
		recordHistogram(scope, deleteHist, labels...)
	})
}
