package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

const (
	defBatchTimeout  = 5 * time.Second
	defExportTimeout = 10 * time.Second
)

// NewHookTracerClient is constructor for mocking otlp traces client
func NewHookTracerClient(cfg *HookClientConfig) otlptrace.Client {
	return newHookClient(cfg)
}

// proxyTracerClient is a custom client to implement traces async uploader via hook client
type proxyTracerClient struct {
	hookClient otlptrace.Client
	otlpClient otlptrace.Client
}

// NewProxyTracerClient is constructor for async traces client uploader via hook client
func NewProxyTracerClient(oc, hc otlptrace.Client) otlptrace.Client {
	ptc := &proxyTracerClient{
		hookClient: hc,
		otlpClient: oc,
	}
	uploadCallback := func(ctx context.Context, traces [][]byte) error {
		protoSpans := make([]*tracepb.ResourceSpans, 0, len(traces))
		for _, trace := range traces {
			resspans := tracepb.ResourceSpans{}
			err := proto.Unmarshal(trace, &resspans)
			if err != nil {
				continue
			}
			protoSpans = append(protoSpans, &resspans)
		}
		if len(protoSpans) != 0 {
			if err := ptc.otlpClient.UploadTraces(ctx, protoSpans); err != nil {
				return fmt.Errorf("failed to upload traces via upload callback: %w", err)
			}
		}
		return nil
	}
	if c, ok := ptc.hookClient.(*hookClient); ok && c.cfg.UploadCallback == nil {
		c.cfg.UploadCallback = uploadCallback
	}
	return ptc
}

// Start is function to run traces clients pair (otlp client is first)
func (ptc *proxyTracerClient) Start(ctx context.Context) error {
	if err := ptc.otlpClient.Start(ctx); err != nil {
		return fmt.Errorf("failed to start otlp client: %w", err)
	}
	if err := ptc.hookClient.Start(ctx); err != nil {
		return fmt.Errorf("failed to start hook client: %w", err)
	}
	return nil
}

// Stop is function to stop traces clients pair (hook client is first)
func (ptc *proxyTracerClient) Stop(ctx context.Context) error {
	if err := ptc.hookClient.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop hook client: %w", err)
	}
	if err := ptc.otlpClient.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop otlp client: %w", err)
	}
	return nil
}

// UploadTraces is function to transfer traces and spans via hook client
func (ptc *proxyTracerClient) UploadTraces(ctx context.Context, protoSpans []*tracepb.ResourceSpans) error {
	if err := ptc.hookClient.UploadTraces(ctx, protoSpans); err != nil {
		return fmt.Errorf("failed to upload traces via proxy client: %w", err)
	}
	return nil
}

// NewOtlpTracerClient is constructor for original otlp traces client
func NewOtlpTracerClient(addr string) otlptrace.Client {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(addr),
		otlptracegrpc.WithReconnectionPeriod(defReconnPeriod),
	}
	return otlptracegrpc.NewClient(opts...)
}

// NewTracerProvider is constructor for otlp sdk traces provider with custom otlp client
func NewTracerProvider(
	ctx context.Context,
	client otlptrace.Client,
	service,
	version string,
	opts ...attribute.KeyValue,
) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptrace.New(ctx, &samplerProxyClient{client})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a new tracer exporter: %w", err)
	}

	res, err := newResource(service, version, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a new application resource for the tracer: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(
			exporter,
			sdktrace.WithBatchTimeout(defBatchTimeout),
			sdktrace.WithExportTimeout(defExportTimeout),
		),
		sdktrace.WithResource(res),
	)

	return tp, err
}

type samplerProxyClient struct {
	otlptrace.Client
}

// Index which contains span ID to object span links (key in the map means current span ID)
type spcIndex map[string]*tracepb.Span

// Index which contains parent to children links (key in the map means parent span ID)
type spcPIndex map[string][]*tracepb.Span

func checkToKeepSpanByParent(span *tracepb.Span, index spcIndex) bool {
	if span.ParentSpanId == nil {
		return false
	}
	parent, ok := index[string(span.ParentSpanId)]
	if !ok {
		return false
	}
	if len(parent.Events) != 0 || len(parent.Links) != 0 {
		return true
	}
	return checkToKeepSpanByParent(parent, index)
}

func checkToKeepSpanByChildren(span *tracepb.Span, pindex spcPIndex) bool {
	if len(span.Events) != 0 || len(span.Links) != 0 {
		return true
	}
	children := pindex[string(span.SpanId)]
	for _, child := range children {
		if checkToKeepSpanByChildren(child, pindex) {
			return true
		}
	}
	return false
}

func buildSpansIndex(spans []*tracepb.Span) (spcIndex, spcPIndex) {
	index, pindex := make(spcIndex), make(spcPIndex)
	for _, span := range spans {
		index[string(span.SpanId)] = span
		pSpanId := string(span.ParentSpanId)
		pindex[pSpanId] = append(pindex[pSpanId], span)
	}
	return index, pindex
}

// UploadTraces is function to cut empty traces when events list is empty for all spans in the branch
func (pc *samplerProxyClient) UploadTraces(ctx context.Context, protoSpans []*tracepb.ResourceSpans) error {
	// variable to store of rebuilded proto spans list result
	rprotoSpans := make([]*tracepb.ResourceSpans, 0, len(protoSpans))

	for _, pspans := range protoSpans {
		if pspans == nil {
			continue
		}
		instLibSpans := pspans.GetInstrumentationLibrarySpans()
		rinstLibSpans := make([]*tracepb.InstrumentationLibrarySpans, 0, len(instLibSpans))

		for _, instLibSpan := range instLibSpans {
			if instLibSpan == nil {
				continue
			}

			spans := instLibSpan.GetSpans()
			index, pindex := buildSpansIndex(spans)
			rspans := make([]*tracepb.Span, 0, len(spans))
			for _, span := range spans {
				if checkToKeepSpanByParent(span, index) {
					rspans = append(rspans, span)
				} else if checkToKeepSpanByChildren(span, pindex) {
					rspans = append(rspans, span)
				}
			}
			instLibSpan.Spans = rspans

			if len(instLibSpan.Spans) == 0 {
				continue
			}
			rinstLibSpans = append(rinstLibSpans, instLibSpan)
		}

		if len(rinstLibSpans) == 0 {
			continue
		}
		pspans.InstrumentationLibrarySpans = rinstLibSpans
		rprotoSpans = append(rprotoSpans, pspans)
	}

	// avoid storing and sending spans empty lists
	if len(rprotoSpans) == 0 {
		return nil
	}
	return pc.Client.UploadTraces(ctx, rprotoSpans)
}
