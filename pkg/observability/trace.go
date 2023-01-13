package observability

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/model/otlpgrpc"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

const (
	defBatchTimeout  = 5 * time.Second
	defExportTimeout = 10 * time.Second
)

var (
	pAnyValueTypeString = reflect.TypeOf((*commonpb.AnyValue_StringValue)(nil))
	pAnyValueTypeBool   = reflect.TypeOf((*commonpb.AnyValue_BoolValue)(nil))
	pAnyValueTypeInt    = reflect.TypeOf((*commonpb.AnyValue_IntValue)(nil))
	pAnyValueTypeDouble = reflect.TypeOf((*commonpb.AnyValue_DoubleValue)(nil))
	pAnyValueTypeArray  = reflect.TypeOf((*commonpb.AnyValue_ArrayValue)(nil))
	pAnyValueTypeKvlist = reflect.TypeOf((*commonpb.AnyValue_KvlistValue)(nil))
	pAnyValueTypeBytes  = reflect.TypeOf((*commonpb.AnyValue_BytesValue)(nil))
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

// logsTracerClient is a custom client to implement traces async uploader via hook client
type logsTracerClient struct {
	mx         *sync.Mutex
	addr       string
	grpcClient *grpc.ClientConn
	logsClient otlpgrpc.LogsClient
	otlpClient otlptrace.Client
}

func (ltc *logsTracerClient) newLogsClient(ctx context.Context) (*grpc.ClientConn, otlpgrpc.LogsClient, error) {
	grpcClient, err := grpc.Dial(ltc.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithWriteBufferSize(512*1024),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to otel gRPC service: %w", err)
	}
	return grpcClient, otlpgrpc.NewLogsClient(grpcClient), nil
}

func (ltc *logsTracerClient) copyToAttribute(src *commonpb.AnyValue, dst pdata.AttributeValue) {
	switch reflect.TypeOf(src.GetValue()) {
	case pAnyValueTypeString:
		dst.SetStringVal(src.GetStringValue())
	case pAnyValueTypeBool:
		dst.SetBoolVal(src.GetBoolValue())
	case pAnyValueTypeInt:
		dst.SetIntVal(src.GetIntValue())
	case pAnyValueTypeDouble:
		dst.SetDoubleVal(src.GetDoubleValue())
	case pAnyValueTypeBytes:
		dst.SetBytesVal(src.GetBytesValue())
	case pAnyValueTypeArray:
		slv := dst.SliceVal()
		for _, v := range src.GetArrayValue().GetValues() {
			ltc.copyToAttribute(v, slv.AppendEmpty())
		}
	case pAnyValueTypeKvlist:
		slv := dst.MapVal()
		for _, kv := range src.GetKvlistValue().GetValues() {
			a := pdata.NewAttributeValueEmpty()
			k, v := kv.GetKey(), kv.GetValue()
			ltc.copyToAttribute(v, a)
			slv.Upsert(k, a)
		}
	default:
	}
}

func (ltc *logsTracerClient) copyToResAttributes(src []*commonpb.KeyValue, dst pdata.AttributeMap) {
	for _, kv := range src {
		k, v := strings.ReplaceAll(kv.GetKey(), ".", "_"), kv.GetValue()
		if strings.HasPrefix(k, "telemetry") || k == "environment" {
			continue
		} else if k == "os_type" {
			k = "service_os"
		}
		a := pdata.NewAttributeValueEmpty()
		ltc.copyToAttribute(v, a)
		dst.Upsert(k, a)
	}
}

func (ltc *logsTracerClient) copyToLogAttributes(src []*commonpb.KeyValue, dst pdata.AttributeMap, trimPrefix string) {
	for _, kv := range src {
		k, v := strings.ReplaceAll(strings.TrimPrefix(kv.GetKey(), trimPrefix), ".", "_"), kv.GetValue()
		a := pdata.NewAttributeValueEmpty()
		ltc.copyToAttribute(v, a)
		dst.Upsert(k, a)
	}
}

func (ltc *logsTracerClient) getSeverityNumber(severity string) pdata.SeverityNumber {
	switch severity {
	case "TRACE", "trace":
		return pdata.SeverityNumberTRACE
	case "DEBUG", "debug":
		return pdata.SeverityNumberDEBUG
	case "INFO", "info":
		return pdata.SeverityNumberINFO
	case "WARN", "warn":
		return pdata.SeverityNumberWARN
	case "ERROR", "error":
		return pdata.SeverityNumberERROR
	case "FATAL", "fatal":
		return pdata.SeverityNumberFATAL
	default:
		return pdata.SeverityNumberUNDEFINED
	}
}

func (ltc *logsTracerClient) getLogs(protoSpans []*tracepb.ResourceSpans) pdata.Logs {
	pLog := pdata.NewLogs()

	for _, pspans := range protoSpans {
		if pspans == nil {
			continue
		}

		pmm := pLog.ResourceLogs().AppendEmpty()
		ltc.copyToResAttributes(pspans.Resource.GetAttributes(), pmm.Resource().Attributes())
		pmm.SetSchemaUrl(pspans.GetSchemaUrl())

		instLibSpans := pspans.GetInstrumentationLibrarySpans()
		for _, instLibSpan := range instLibSpans {
			if instLibSpan == nil {
				continue
			}

			il := instLibSpan.GetInstrumentationLibrary()
			lsl := pmm.InstrumentationLibraryLogs().AppendEmpty()
			lsl.InstrumentationLibrary().SetName(il.GetName())
			lsl.InstrumentationLibrary().SetVersion(il.GetVersion())

			spans := instLibSpan.GetSpans()
			for _, span := range spans {
				for _, event := range span.GetEvents() {
					lr := lsl.LogRecords().AppendEmpty()
					lr.SetName(span.GetName())

					logTimeUnix := event.GetTimeUnixNano()
					logTime := time.Unix(int64(logTimeUnix/1e9), int64(logTimeUnix%1e9))
					lr.SetTimestamp(pdata.NewTimestampFromTime(logTime))

					var spanID [8]byte
					copy(spanID[:], span.GetSpanId()[:8])
					lr.SetSpanID(pdata.NewSpanID(spanID))
					var traceID [16]byte
					copy(traceID[:], span.GetTraceId()[:16])
					lr.SetTraceID(pdata.NewTraceID(traceID))

					lra := lr.Attributes()
					ltc.copyToLogAttributes(span.GetAttributes(), lra, "span.")
					ltc.copyToLogAttributes(event.GetAttributes(), lra, event.GetName()+".")

					if event.GetName() == "exception" {
						lra.UpsertString("severity", "ERROR")
					}
					lra.UpsertString("time", logTime.Format("2006-01-02 15:04:05.000000000"))

					var severity string
					if a, ok := lra.Get("severity"); ok && a.Type() == pdata.AttributeValueTypeString {
						severity = a.AsString()
					}

					lr.SetSeverityNumber(ltc.getSeverityNumber(severity))
					lr.SetSeverityText(severity)
					lr.SetFlags(31)
				}
			}
		}
	}

	return pLog
}

// Start is function to run otlp and grpc clients pair (grpc client is first)
func (ltc *logsTracerClient) Start(ctx context.Context) error {
	ltc.mx.Lock()
	defer ltc.mx.Unlock()

	var err error
	if ltc.logsClient == nil || ltc.grpcClient == nil {
		if ltc.grpcClient, ltc.logsClient, err = ltc.newLogsClient(ctx); err != nil {
			return fmt.Errorf("failed to start grpc client: %w", err)
		}
	}
	if err = ltc.otlpClient.Start(ctx); err != nil {
		return fmt.Errorf("failed to start otlp client: %w", err)
	}

	return nil
}

// Stop is function to stop otlp and grpc clients pair (otlp client is first)
func (ltc *logsTracerClient) Stop(ctx context.Context) error {
	ltc.mx.Lock()
	defer ltc.mx.Unlock()

	if err := ltc.otlpClient.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop otlp client: %w", err)
	}
	ltc.logsClient = nil
	if ltc.grpcClient == nil {
		if err := ltc.grpcClient.Close(); err != nil {
			return fmt.Errorf("failed to stop grpc client: %w", err)
		}
		ltc.grpcClient = nil
	}

	return nil
}

// UploadTraces is function to transfer traces, spans and logs via otlp and grpc clients
func (ltc *logsTracerClient) UploadTraces(ctx context.Context, protoSpans []*tracepb.ResourceSpans) error {
	ltc.mx.Lock()
	defer ltc.mx.Unlock()

	if err := ltc.otlpClient.UploadTraces(ctx, protoSpans); err != nil {
		return fmt.Errorf("failed to upload traces via otlp client: %w", err)
	}
	if ltc.grpcClient == nil || ltc.logsClient == nil {
		return nil
	}
	request := otlpgrpc.NewLogsRequest()
	request.SetLogs(ltc.getLogs(protoSpans))
	if _, err := ltc.logsClient.Export(ctx, request); err != nil {
		return fmt.Errorf("failed to upload logs via grpc client: %w", err)
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

// NewOtlpTracerAndLoggerClient is constructor for otlp traces and logs client
func NewOtlpTracerAndLoggerClient(addr string) otlptrace.Client {
	var err error
	ltc := logsTracerClient{
		mx:   &sync.Mutex{},
		addr: addr,
	}
	ltc.grpcClient, err = grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithWriteBufferSize(512*1024),
	)
	if err != nil {
		fmt.Printf("failed to connect to otel gRPC service: %v\n", err)
	} else {
		ltc.logsClient = otlpgrpc.NewLogsClient(ltc.grpcClient)
	}
	ltc.otlpClient = NewOtlpTracerClient(addr)
	return &ltc
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
