package observability

import (
	"context"
	"crypto/rand"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/uptrace/opentelemetry-go-extra/otelutil"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	otelmetric "go.opentelemetry.io/otel/metric"
	metricglobal "go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/number"
	"go.opentelemetry.io/otel/metric/sdkapi"
	"go.opentelemetry.io/otel/propagation"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type SpanContextKey int

// context functors keys to close blocking spans
const (
	VXProtoAgentConnect SpanContextKey = iota + 1
)

var (
	logSeverityKey = attribute.Key("log.severity")
	logMessageKey  = attribute.Key("log.message")
)

const (
	// SpanKindUnspecified is an unspecified SpanKind and is not a valid
	// SpanKind. SpanKindUnspecified should be replaced with SpanKindInternal
	// if it is received.
	SpanKindUnspecified oteltrace.SpanKind = 0
	// SpanKindInternal is a SpanKind for a Span that represents an internal
	// operation within an application.
	SpanKindInternal oteltrace.SpanKind = 1
	// SpanKindServer is a SpanKind for a Span that represents the operation
	// of handling a request from a client.
	SpanKindServer oteltrace.SpanKind = 2
	// SpanKindClient is a SpanKind for a Span that represents the operation
	// of client making a request to a server.
	SpanKindClient oteltrace.SpanKind = 3
	// SpanKindProducer is a SpanKind for a Span that represents the operation
	// of a producer sending a message to a message broker. Unlike
	// SpanKindClient and SpanKindServer, there is often no direct
	// relationship between this kind of Span and a SpanKindConsumer kind. A
	// SpanKindProducer Span will end once the message is accepted by the
	// message broker which might not overlap with the processing of that
	// message.
	SpanKindProducer oteltrace.SpanKind = 4
	// SpanKindConsumer is a SpanKind for a Span that represents the operation
	// of a consumer receiving a message from a message broker. Like
	// SpanKindProducer Spans, there is often no direct relationship between
	// this Span and the Span that produced the message.
	SpanKindConsumer oteltrace.SpanKind = 5
)

var Observer IObserver

type IObserver interface {
	Flush(ctx context.Context)
	Close()
	IMeter
	ITracer
	ICollector
}

type ITracer interface {
	NewSpan(context.Context, oteltrace.SpanKind, string, ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span)
	NewSpanWithParent(context.Context, oteltrace.SpanKind, string, string, string, ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span)
	SpanFromContext(ctx context.Context) oteltrace.Span
	SpanContextFromContext(ctx context.Context) oteltrace.SpanContext
}

type IMeter interface {
	NewFloat64Counter(name string, options ...otelmetric.InstrumentOption) (otelmetric.Float64Counter, error)
	NewFloat64Histogram(name string, options ...otelmetric.InstrumentOption) (otelmetric.Float64Histogram, error)
	NewFloat64UpDownCounter(name string, options ...otelmetric.InstrumentOption) (otelmetric.Float64UpDownCounter, error)
	NewInt64Counter(name string, options ...otelmetric.InstrumentOption) (otelmetric.Int64Counter, error)
	NewInt64Histogram(name string, options ...otelmetric.InstrumentOption) (otelmetric.Int64Histogram, error)
	NewInt64UpDownCounter(name string, options ...otelmetric.InstrumentOption) (otelmetric.Int64UpDownCounter, error)
	NewInt64GaugeCounter(name string, options ...otelmetric.InstrumentOption) (Int64GaugeCounter, error)
	NewFloat64GaugeCounter(name string, options ...otelmetric.InstrumentOption) (Float64GaugeCounter, error)
}

type ICollector interface {
	StartProcessMetricCollect(service, version string, attrs ...attribute.KeyValue) error
	StartGoRuntimeMetricCollect(service, version string, attrs ...attribute.KeyValue) error
	StartDumperMetricCollect(stats IDumper, service, version string, attrs ...attribute.KeyValue) error
}

type IDumper interface {
	DumpStats() (map[string]float64, error)
}

func init() {
	InitObserver(context.Background(), nil, nil, nil, nil, "noop", "v0.0.0", []logrus.Level{})
}

func InitObserver(
	ctx context.Context,
	tprovider *sdktrace.TracerProvider,
	mprovider *controller.Controller,
	tclient otlptrace.Client,
	mclient otlpmetric.Client,
	tname string,
	tversion string,
	levels []logrus.Level,
) {
	if Observer != nil {
		Observer.Close()
	}

	ctx, cancelCtx := context.WithCancel(ctx)
	obs := &observer{
		cancelCtx: cancelCtx,
		levels:    levels,
		tprovider: tprovider,
		mprovider: mprovider,
		tclient:   tclient,
		mclient:   mclient,
	}

	tverRev := strings.Split(tversion, "-")
	tversion = strings.TrimPrefix(tverRev[0], "v")

	if tprovider != nil {
		otel.SetTracerProvider(tprovider)
		otel.SetTextMapPropagator(
			propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			),
		)
		obs.tracer = tprovider.Tracer(tname, oteltrace.WithInstrumentationVersion(tversion))
		logrus.AddHook(obs)
	}

	if mprovider != nil {
		metricglobal.SetMeterProvider(mprovider)
		obs.meter = mprovider.Meter(tname, otelmetric.WithInstrumentationVersion(tversion))
		mprovider.Start(ctx)
	}

	Observer = obs
}

// Int64GaugeCounter is a metric that records int64 values.
type Int64GaugeCounter struct {
	sdkapi.SyncImpl
}

// Measurement creates a Measurement object to use with batch recording.
func (c Int64GaugeCounter) Measurement(value int64) otelmetric.Measurement {
	return sdkapi.NewMeasurement(c, number.NewInt64Number(value))
}

// Record adds a new value to the Histogram's distribution. The
// labels should contain the keys and values to be associated with
// this value.
func (c Int64GaugeCounter) Record(ctx context.Context, value int64, labels ...attribute.KeyValue) {
	c.RecordOne(ctx, number.NewInt64Number(value), labels)
}

// Float64GaugeCounter is a metric that records float64 values.
type Float64GaugeCounter struct {
	sdkapi.SyncImpl
}

// Measurement creates a Measurement object to use with batch recording.
func (c Float64GaugeCounter) Measurement(value float64) otelmetric.Measurement {
	return sdkapi.NewMeasurement(c, number.NewFloat64Number(value))
}

// Record adds a new value to the list of Histogram's records. The
// labels should contain the keys and values to be associated with
// this value.
func (c Float64GaugeCounter) Record(ctx context.Context, value float64, labels ...attribute.KeyValue) {
	c.RecordOne(ctx, number.NewFloat64Number(value), labels)
}

// newResource returns a resource describing the application
func newResource(service, version string, opts ...attribute.KeyValue) (*resource.Resource, error) {
	var env = "production"
	verRev := strings.Split(version, "-")
	version = strings.TrimPrefix(verRev[0], "v")
	if len(verRev) == 2 {
		env = "development"
	}

	opts = append(opts,
		semconv.OSTypeKey.String(runtime.GOOS),
		attribute.String("service.arch", runtime.GOARCH),
		semconv.ServiceNameKey.String(service),
		semconv.ServiceVersionKey.String(version),
		attribute.String("environment", env),
	)

	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			opts...,
		),
	)
	return r, err
}

type observer struct {
	cancelCtx context.CancelFunc
	levels    []logrus.Level
	meter     otelmetric.Meter
	tracer    oteltrace.Tracer
	tprovider *sdktrace.TracerProvider
	mprovider *controller.Controller
	tclient   otlptrace.Client
	mclient   otlpmetric.Client
}

func (obs *observer) flushMClient(ctx context.Context) {
	if obs == nil {
		return
	}
	if obs.mclient == nil {
		return
	}

	if client, ok := obs.mclient.(*hookClient); ok {
		client.Flush(ctx)
	}
	if pclient, ok := obs.mclient.(*proxyMeterClient); ok {
		if client, ok := pclient.hookClient.(*hookClient); ok {
			client.Flush(ctx)
		}
		if client, ok := pclient.otlpClient.(*hookClient); ok {
			client.Flush(ctx)
		}
	}
}

func (obs *observer) flushTClient(ctx context.Context) {
	if obs == nil {
		return
	}
	if obs.tclient == nil {
		return
	}

	if client, ok := obs.tclient.(*hookClient); ok {
		client.Flush(ctx)
	}
	if pclient, ok := obs.tclient.(*proxyTracerClient); ok {
		if client, ok := pclient.hookClient.(*hookClient); ok {
			client.Flush(ctx)
		}
		if client, ok := pclient.otlpClient.(*hookClient); ok {
			client.Flush(ctx)
		}
	}
}

func (obs *observer) Flush(ctx context.Context) {
	if obs == nil {
		return
	}

	if obs.mprovider != nil {
		// TODO: it's dirty hack because otel sdk can't use Collect method when ticker was running
		if obs.mprovider.IsRunning() {
			obs.mprovider.Stop(ctx)
			obs.mprovider.Start(ctx)
		} else {
			obs.mprovider.Collect(ctx)
		}
		obs.flushMClient(ctx)
	}
	if obs.tprovider != nil {
		obs.tprovider.ForceFlush(ctx)
		obs.flushTClient(ctx)
	}
}

func (obs *observer) Close() {
	if obs == nil {
		return
	}

	for lvl, hooks := range logrus.StandardLogger().Hooks {
		for idx, hook := range hooks {
			if hook == obs {
				logrus.StandardLogger().Hooks[lvl] = append(hooks[:idx], hooks[idx+1:]...)
				break
			}
		}
	}
	ctx := context.Background()
	if obs.mprovider != nil {
		if obs.mprovider.IsRunning() {
			obs.mprovider.Stop(ctx)
		}
		obs.mprovider.Collect(ctx)
		obs.flushMClient(ctx)
	}
	if obs.tprovider != nil {
		obs.tprovider.ForceFlush(ctx)
		obs.flushTClient(ctx)
		obs.tprovider.Shutdown(ctx)
	}
	obs.cancelCtx()
}

func (obs *observer) StartProcessMetricCollect(service, version string, attrs ...attribute.KeyValue) error {
	attrs = append(attrs,
		semconv.ServiceNameKey.String(service),
		semconv.ServiceVersionKey.String(version),
	)
	return startProcessMetricCollect(obs.meter, attrs)
}

func (obs *observer) StartGoRuntimeMetricCollect(service, version string, attrs ...attribute.KeyValue) error {
	attrs = append(attrs,
		semconv.ServiceNameKey.String(service),
		semconv.ServiceVersionKey.String(version),
	)
	return startGoRuntimeMetricCollect(obs.meter, attrs)
}

func (obs *observer) StartDumperMetricCollect(stats IDumper,
	service, version string, attrs ...attribute.KeyValue) error {
	attrs = append(attrs,
		semconv.ServiceNameKey.String(service),
		semconv.ServiceVersionKey.String(version),
	)
	return startDumperMetricCollect(stats, obs.meter, attrs)
}

func (obs *observer) NewFloat64Counter(name string,
	options ...otelmetric.InstrumentOption) (otelmetric.Float64Counter, error) {
	if obs == nil {
		return otelmetric.Float64Counter{}, fmt.Errorf("observer object is not initialized")
	}
	if obs.meter.MeterImpl() != nil {
		return obs.meter.NewFloat64Counter(name, options...)
	}
	return otelmetric.Float64Counter{}, fmt.Errorf("meter object is not initialized")
}

func (obs *observer) NewFloat64GaugeCounter(name string,
	options ...otelmetric.InstrumentOption) (Float64GaugeCounter, error) {
	impl := obs.meter.MeterImpl()
	if impl == nil {
		return Float64GaugeCounter{SyncImpl: sdkapi.NewNoopSyncInstrument()}, nil
	}

	cfg := otelmetric.NewInstrumentConfig(options...)
	desc := sdkapi.NewDescriptor(name, sdkapi.GaugeObserverInstrumentKind,
		number.Float64Kind, cfg.Description(), cfg.Unit())
	inst, err := impl.NewSyncInstrument(desc)
	if err != nil {
		return Float64GaugeCounter{SyncImpl: sdkapi.NewNoopSyncInstrument()}, err
	}

	return Float64GaugeCounter{SyncImpl: inst}, nil
}

func (obs *observer) NewFloat64Histogram(name string,
	options ...otelmetric.InstrumentOption) (otelmetric.Float64Histogram, error) {
	if obs == nil {
		return otelmetric.Float64Histogram{}, fmt.Errorf("observer object is not initialized")
	}
	if obs.meter.MeterImpl() != nil {
		return obs.meter.NewFloat64Histogram(name, options...)
	}
	return otelmetric.Float64Histogram{}, fmt.Errorf("meter object is not initialized")
}

func (obs *observer) NewFloat64UpDownCounter(name string,
	options ...otelmetric.InstrumentOption) (otelmetric.Float64UpDownCounter, error) {
	if obs == nil {
		return otelmetric.Float64UpDownCounter{}, fmt.Errorf("observer object is not initialized")
	}
	if obs.meter.MeterImpl() != nil {
		return obs.meter.NewFloat64UpDownCounter(name, options...)
	}
	return otelmetric.Float64UpDownCounter{}, fmt.Errorf("meter object is not initialized")
}

func (obs *observer) NewInt64Counter(name string,
	options ...otelmetric.InstrumentOption) (otelmetric.Int64Counter, error) {
	if obs == nil {
		return otelmetric.Int64Counter{}, fmt.Errorf("observer object is not initialized")
	}
	if obs.meter.MeterImpl() != nil {
		return obs.meter.NewInt64Counter(name, options...)
	}
	return otelmetric.Int64Counter{}, fmt.Errorf("meter object is not initialized")
}

func (obs *observer) NewInt64GaugeCounter(name string,
	options ...otelmetric.InstrumentOption) (Int64GaugeCounter, error) {
	impl := obs.meter.MeterImpl()
	if impl == nil {
		return Int64GaugeCounter{SyncImpl: sdkapi.NewNoopSyncInstrument()}, nil
	}

	cfg := otelmetric.NewInstrumentConfig(options...)
	desc := sdkapi.NewDescriptor(name, sdkapi.GaugeObserverInstrumentKind,
		number.Int64Kind, cfg.Description(), cfg.Unit())
	inst, err := impl.NewSyncInstrument(desc)
	if err != nil {
		return Int64GaugeCounter{SyncImpl: sdkapi.NewNoopSyncInstrument()}, err
	}

	return Int64GaugeCounter{SyncImpl: inst}, nil
}

func (obs *observer) NewInt64Histogram(name string,
	options ...otelmetric.InstrumentOption) (otelmetric.Int64Histogram, error) {
	if obs == nil {
		return otelmetric.Int64Histogram{}, fmt.Errorf("observer object is not initialized")
	}
	if obs.meter.MeterImpl() != nil {
		return obs.meter.NewInt64Histogram(name, options...)
	}
	return otelmetric.Int64Histogram{}, fmt.Errorf("meter object is not initialized")
}

func (obs *observer) NewInt64UpDownCounter(name string,
	options ...otelmetric.InstrumentOption) (otelmetric.Int64UpDownCounter, error) {
	if obs == nil {
		return otelmetric.Int64UpDownCounter{}, fmt.Errorf("observer object is not initialized")
	}
	if obs.meter.MeterImpl() != nil {
		return obs.meter.NewInt64UpDownCounter(name, options...)
	}
	return otelmetric.Int64UpDownCounter{}, fmt.Errorf("meter object is not initialized")
}

func (obs *observer) NewSpan(ctx context.Context, kind oteltrace.SpanKind,
	component string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	if ctx == nil {
		// TODO: here should use default context
		ctx = context.TODO()
	}
	if obs == nil || obs.tracer == nil {
		return oteltrace.NewNoopTracerProvider().Tracer("noop").Start(
			ctx,
			component,
			oteltrace.WithSpanKind(kind),
			oteltrace.WithAttributes(attribute.Key("span.component").String(component)),
		)
	}

	opts = append(opts,
		oteltrace.WithSpanKind(kind),
		oteltrace.WithAttributes(attribute.Key("span.component").String(component)),
	)
	return obs.tracer.Start(ctx, component, opts...)
}

func (obs *observer) NewSpanWithParent(ctx context.Context, kind oteltrace.SpanKind,
	component, traceID, pspanID string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	if ctx == nil {
		// TODO: here should use default context
		ctx = context.TODO()
	}
	if obs == nil || obs.tracer == nil {
		opts = append(opts,
			oteltrace.WithSpanKind(kind),
			oteltrace.WithAttributes(attribute.Key("span.component").String(component)),
		)
		return oteltrace.NewNoopTracerProvider().Tracer("noop").Start(ctx, component, opts...)
	}

	var (
		err error
		tid oteltrace.TraceID
		sid oteltrace.SpanID
	)
	tid, err = oteltrace.TraceIDFromHex(traceID)
	if err != nil {
		rand.Read(tid[:])
	}
	sid, err = oteltrace.SpanIDFromHex(pspanID)
	if err != nil {
		sid = oteltrace.SpanID{}
	}
	ctx = oteltrace.ContextWithRemoteSpanContext(ctx,
		oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
			TraceID: tid,
			SpanID:  sid,
		}),
	)

	return obs.tracer.Start(
		ctx,
		component,
		oteltrace.WithSpanKind(kind),
		oteltrace.WithAttributes(attribute.Key("span.component").String(component)),
	)
}

func (obs *observer) SpanFromContext(ctx context.Context) oteltrace.Span {
	return oteltrace.SpanFromContext(ctx)
}

func (obs *observer) SpanContextFromContext(ctx context.Context) oteltrace.SpanContext {
	return oteltrace.SpanContextFromContext(ctx)
}

func (obs *observer) makeAttrs(entry *logrus.Entry, span oteltrace.Span) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, len(entry.Data)+2+3)

	attrs = append(attrs, logSeverityKey.String(levelString(entry.Level)))
	attrs = append(attrs, logMessageKey.String(entry.Message))

	if entry.Caller != nil {
		if entry.Caller.Function != "" {
			attrs = append(attrs, semconv.CodeFunctionKey.String(entry.Caller.Function))
		}
		if entry.Caller.File != "" {
			attrs = append(attrs, semconv.CodeFilepathKey.String(entry.Caller.File))
			attrs = append(attrs, semconv.CodeLineNumberKey.Int(entry.Caller.Line))
		}
	}

	opts := []oteltrace.EventOption{}
	if entry.Level <= logrus.ErrorLevel {
		span.SetStatus(codes.Error, entry.Message)
		opts = append(opts, oteltrace.WithStackTrace(true))
	}

	for k, v := range entry.Data {
		if k == "error" {
			switch val := v.(type) {
			case error:
				span.RecordError(val, opts...)
			case fmt.Stringer:
				attrs = append(attrs, semconv.ExceptionTypeKey.String(reflect.TypeOf(val).String()))
				span.RecordError(fmt.Errorf(val.String()), opts...)
			case nil:
				span.RecordError(fmt.Errorf("unknown or empty error type: nil"), opts...)
			default:
				attrs = append(attrs, semconv.ExceptionTypeKey.String(reflect.TypeOf(val).String()))
				span.RecordError(fmt.Errorf("unknown exception: %v", v), opts...)
			}
			continue
		}

		attrs = append(attrs, otelutil.Attribute("log."+k, v))
	}

	return attrs
}

// Fire is a logrus hook that is fired on a new log entry.
func (obs *observer) Fire(entry *logrus.Entry) error {
	if obs == nil {
		return nil
	}

	ctx := entry.Context
	if ctx == nil {
		ctx = context.Background()
	}

	span := oteltrace.SpanFromContext(ctx)
	if !span.IsRecording() {
		component := "internal"
		if op, ok := entry.Data["component"]; ok {
			component = op.(string)
		}
		if obs.tracer == nil {
			return nil
		}
		_, span = obs.NewSpan(ctx, oteltrace.SpanKindInternal, component)
		defer span.End()
	}

	span.AddEvent("log", oteltrace.WithAttributes(obs.makeAttrs(entry, span)...))

	return nil
}

func (obs *observer) Levels() []logrus.Level {
	if obs == nil {
		return []logrus.Level{}
	}

	return obs.levels
}

func levelString(lvl logrus.Level) string {
	s := lvl.String()
	if s == "warning" {
		s = "warn"
	}
	return strings.ToUpper(s)
}
