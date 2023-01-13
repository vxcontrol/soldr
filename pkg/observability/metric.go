package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	otelmetric "go.opentelemetry.io/otel/metric"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	"google.golang.org/protobuf/proto"
)

const (
	defCollectPeriod = 10 * time.Second
	defPushTimeout   = 10 * time.Second
)

// NewHookMeterClient is constructor for mocking otlp metrics client
func NewHookMeterClient(cfg *HookClientConfig) otlpmetric.Client {
	return newHookClient(cfg)
}

// proxyMeterClient is a custom client to implement metrics async uploader via hook client
type proxyMeterClient struct {
	hookClient otlpmetric.Client
	otlpClient otlpmetric.Client
}

// NewProxyMeterClient is constructor for async metrics client uploader via hook client
func NewProxyMeterClient(oc, hc otlpmetric.Client) otlpmetric.Client {
	pmc := &proxyMeterClient{
		hookClient: hc,
		otlpClient: oc,
	}
	uploadCallback := func(ctx context.Context, metrics [][]byte) error {
		protoMetrics := make([]*metricspb.ResourceMetrics, 0, len(metrics))
		for _, metric := range metrics {
			resmetrics := metricspb.ResourceMetrics{}
			err := proto.Unmarshal(metric, &resmetrics)
			if err != nil {
				continue
			}
			protoMetrics = append(protoMetrics, &resmetrics)
		}
		if len(protoMetrics) != 0 {
			if err := pmc.otlpClient.UploadMetrics(ctx, protoMetrics); err != nil {
				return fmt.Errorf("failed to upload metrics via upload callback: %w", err)
			}
		}
		return nil
	}
	if c, ok := pmc.hookClient.(*hookClient); ok && c.cfg.UploadCallback == nil {
		c.cfg.UploadCallback = uploadCallback
	}
	return pmc
}

// Start is function to run metrics clients pair (otlp client is first)
func (pmc *proxyMeterClient) Start(ctx context.Context) error {
	if err := pmc.otlpClient.Start(ctx); err != nil {
		return fmt.Errorf("failed to start otlp client: %w", err)
	}
	if err := pmc.hookClient.Start(ctx); err != nil {
		return fmt.Errorf("failed to start hook client: %w", err)
	}
	return nil
}

// Stop is function to stop metrics clients pair (hook client is first)
func (pmc *proxyMeterClient) Stop(ctx context.Context) error {
	if err := pmc.hookClient.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop hook client: %w", err)
	}
	if err := pmc.otlpClient.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop otlp client: %w", err)
	}
	return nil
}

// UploadMetrics is function to transfer metrics via hook client
func (pmc *proxyMeterClient) UploadMetrics(ctx context.Context, protoMetrics []*metricspb.ResourceMetrics) error {
	if err := pmc.hookClient.UploadMetrics(ctx, protoMetrics); err != nil {
		return fmt.Errorf("failed to upload metrics via proxy client: %w", err)
	}
	return nil
}

// NewOtlpMeterClient is constructor for original otlp metrics client
func NewOtlpMeterClient(addr string) otlpmetric.Client {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(addr),
		otlpmetricgrpc.WithReconnectionPeriod(defReconnPeriod),
	}
	return otlpmetricgrpc.NewClient(opts...)
}

// NewMeterProvider is constructor for otlp sdk metrics provider with custom otlp client
func NewMeterProvider(
	ctx context.Context,
	client otlpmetric.Client,
	service,
	version string,
	opts ...attribute.KeyValue,
) (*controller.Controller, error) {
	exporter, err := otlpmetric.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a new metrics exporter: %w", err)
	}

	res, err := newResource(service, version, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a new application resource for the metrics provider: %w", err)
	}

	ct := controller.New(
		processor.NewFactory(
			simple.NewWithHistogramDistribution(),
			exporter,
		),
		controller.WithExporter(exporter),
		controller.WithCollectPeriod(defCollectPeriod),
		controller.WithPushTimeout(defPushTimeout),
		controller.WithResource(res),
	)

	return ct, err
}

type metricRegistry struct {
	cache   map[string]interface{}
	options []otelmetric.InstrumentOption
	mx      *sync.Mutex
	IMeter
}

func NewMetricRegistry(options ...otelmetric.InstrumentOption) (IMeter, error) {
	if Observer == nil {
		return nil, fmt.Errorf("observer object is not initialized")
	}
	return &metricRegistry{
		cache:   make(map[string]interface{}),
		options: append([]otelmetric.InstrumentOption{}, options...),
		mx:      &sync.Mutex{},
		IMeter:  Observer,
	}, nil
}

func (mr *metricRegistry) NewFloat64Counter(name string,
	options ...otelmetric.InstrumentOption) (otelmetric.Float64Counter, error) {
	mr.mx.Lock()
	defer mr.mx.Unlock()

	key := "Float64Counter|" + name
	if metric, ok := mr.cache[key]; ok {
		return metric.(otelmetric.Float64Counter), nil
	}
	res, err := mr.IMeter.NewFloat64Counter(name, append(mr.options, options...)...)
	if err != nil {
		return res, err
	}
	mr.cache[key] = res
	return res, nil
}

func (mr *metricRegistry) NewFloat64GaugeCounter(name string,
	options ...otelmetric.InstrumentOption) (Float64GaugeCounter, error) {
	mr.mx.Lock()
	defer mr.mx.Unlock()

	key := "Float64GaugeCounter|" + name
	if metric, ok := mr.cache[key]; ok {
		return metric.(Float64GaugeCounter), nil
	}
	res, err := mr.IMeter.NewFloat64GaugeCounter(name, append(mr.options, options...)...)
	if err != nil {
		return res, err
	}
	mr.cache[key] = res
	return res, nil
}

func (mr *metricRegistry) NewFloat64Histogram(name string,
	options ...otelmetric.InstrumentOption) (otelmetric.Float64Histogram, error) {
	mr.mx.Lock()
	defer mr.mx.Unlock()

	key := "Float64Histogram|" + name
	if metric, ok := mr.cache[key]; ok {
		return metric.(otelmetric.Float64Histogram), nil
	}
	res, err := mr.IMeter.NewFloat64Histogram(name, append(mr.options, options...)...)
	if err != nil {
		return res, err
	}
	mr.cache[key] = res
	return res, nil
}

func (mr *metricRegistry) NewFloat64UpDownCounter(name string,
	options ...otelmetric.InstrumentOption) (otelmetric.Float64UpDownCounter, error) {
	mr.mx.Lock()
	defer mr.mx.Unlock()

	key := "Float64UpDownCounter|" + name
	if metric, ok := mr.cache[key]; ok {
		return metric.(otelmetric.Float64UpDownCounter), nil
	}
	res, err := mr.IMeter.NewFloat64UpDownCounter(name, append(mr.options, options...)...)
	if err != nil {
		return res, err
	}
	mr.cache[key] = res
	return res, nil
}

func (mr *metricRegistry) NewInt64Counter(name string,
	options ...otelmetric.InstrumentOption) (otelmetric.Int64Counter, error) {
	mr.mx.Lock()
	defer mr.mx.Unlock()

	key := "Int64Counter|" + name
	if metric, ok := mr.cache[key]; ok {
		return metric.(otelmetric.Int64Counter), nil
	}
	res, err := mr.IMeter.NewInt64Counter(name, append(mr.options, options...)...)
	if err != nil {
		return res, err
	}
	mr.cache[key] = res
	return res, nil
}

func (mr *metricRegistry) NewInt64GaugeCounter(name string,
	options ...otelmetric.InstrumentOption) (Int64GaugeCounter, error) {
	mr.mx.Lock()
	defer mr.mx.Unlock()

	key := "Int64GaugeCounter|" + name
	if metric, ok := mr.cache[key]; ok {
		return metric.(Int64GaugeCounter), nil
	}
	res, err := mr.IMeter.NewInt64GaugeCounter(name, append(mr.options, options...)...)
	if err != nil {
		return res, err
	}
	mr.cache[key] = res
	return res, nil
}

func (mr *metricRegistry) NewInt64Histogram(name string,
	options ...otelmetric.InstrumentOption) (otelmetric.Int64Histogram, error) {
	mr.mx.Lock()
	defer mr.mx.Unlock()

	key := "Int64Histogram|" + name
	if metric, ok := mr.cache[key]; ok {
		return metric.(otelmetric.Int64Histogram), nil
	}
	res, err := mr.IMeter.NewInt64Histogram(name, append(mr.options, options...)...)
	if err != nil {
		return res, err
	}
	mr.cache[key] = res
	return res, nil
}

func (mr *metricRegistry) NewInt64UpDownCounter(name string,
	options ...otelmetric.InstrumentOption) (otelmetric.Int64UpDownCounter, error) {
	mr.mx.Lock()
	defer mr.mx.Unlock()

	key := "Int64UpDownCounter|" + name
	if metric, ok := mr.cache[key]; ok {
		return metric.(otelmetric.Int64UpDownCounter), nil
	}
	res, err := mr.IMeter.NewInt64UpDownCounter(name, append(mr.options, options...)...)
	if err != nil {
		return res, err
	}
	mr.cache[key] = res
	return res, nil
}
