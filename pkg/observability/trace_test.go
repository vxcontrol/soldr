package observability_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	oteltrace "go.opentelemetry.io/otel/trace"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"

	obs "soldr/pkg/observability"
)

func checkSpansStructure(t *testing.T, gtraces [][]byte, traceID, pspanID string, nspans int) {
	rtraces := make([]*tracepb.ResourceSpans, len(gtraces))
	for idx, tspans := range gtraces {
		rtraces[idx] = &tracepb.ResourceSpans{}
		if proto.Unmarshal(tspans, rtraces[idx]) != nil {
			t.Fatalf("error on unmarshaling span list for traces")
		}
	}

	if len(gtraces) != 1 || len(rtraces) != 1 {
		t.Fatalf("error on slicing branches spans according root spans")
	}
	instLibSpans := rtraces[0].GetInstrumentationLibrarySpans()
	if len(instLibSpans) != 1 {
		t.Fatalf("error on slicing branches spans according calls")
	}
	if len(instLibSpans[0].GetSpans()) != nspans {
		t.Fatalf("error on storing all spans into the span branch from root entry")
	}
	for _, span := range instLibSpans[0].GetSpans() {
		if string(span.ParentSpanId) == "" && pspanID != "" {
			t.Fatalf("error on getting of stored parent span ID: %v", pspanID)
		}
		if traceID != "" {
			if tid, err := oteltrace.TraceIDFromHex(traceID); err != nil {
				t.Fatalf("error on parsing trace ID: %v", err)
			} else if !bytes.Equal(tid[:], span.TraceId) {
				t.Fatalf("error on getting of stored trace ID: %v | %v", tid, span.TraceId)
			}
		}
	}
}

func putTestSpansBranch(traceID, pspanID string) {
	rootCtx := context.Background()

	span1Ctx, span1 := obs.Observer.NewSpanWithParent(rootCtx, oteltrace.SpanKindInternal, "span1", traceID, pspanID)
	{
		for idx := 0; idx < 5; idx++ {
			logrus.WithContext(span1Ctx).WithField("key", "value").Warnf("test warn log message %d", idx)
		}

		span2Ctx, span2 := obs.Observer.NewSpan(span1Ctx, oteltrace.SpanKindInternal, "span2")
		{
			for idx := 0; idx < 5; idx++ {
				logrus.WithContext(span2Ctx).WithField("key", "value").Infof("test info log message %d", idx)
			}
			logrus.WithContext(span2Ctx).WithError(fmt.Errorf("some error")).Errorf("catch test error")
		}
		span2.End()

		span3Ctx, span3 := obs.Observer.NewSpan(span1Ctx, oteltrace.SpanKindInternal, "span3")
		for idx := 0; idx < 5; idx++ {
			logrus.WithContext(span3Ctx).WithField("key", "value").Debugf("test debug log message %d", idx)
		}
		span3.End()
	}
	span1.End()
}

func putEmptySpansBranch() {
	rootCtx := context.Background()

	span1Ctx, span1 := obs.Observer.NewSpan(rootCtx, oteltrace.SpanKindInternal, "span1")
	{
		span2Ctx, span2 := obs.Observer.NewSpan(span1Ctx, oteltrace.SpanKindInternal, "span2")
		{
			span3Ctx, span3 := obs.Observer.NewSpan(span2Ctx, oteltrace.SpanKindInternal, "span3")
			// must be skipped because current logging level is INFO
			logrus.WithContext(span3Ctx).WithField("key", "value").Debug("test debug log message")
			span3.End()
		}
		span2.End()

		_, span4 := obs.Observer.NewSpan(span1Ctx, oteltrace.SpanKindInternal, "span4")
		span4.End()
	}
	span1.End()
}

func TestObserverTracesAPI(t *testing.T) {
	ctx := context.Background()
	gtraces := make([][]byte, 0)
	upload := func(ctx context.Context, traces [][]byte) error {
		gtraces = append(gtraces, traces...)
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	client := obs.NewHookTracerClient(clientCfg)
	provider, err := obs.NewTracerProvider(ctx, client, service, version)
	if err != nil {
		t.Fatal(err)
	}
	obs.InitObserver(ctx, provider, nil, client, nil, service, version, []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	})

	// start tracing
	putTestSpansBranch("", "")
	// stop tracing

	obs.Observer.Close()
	checkSpansStructure(t, gtraces, "", "", 3)
}

func TestObserverTracesAPIWithFlush(t *testing.T) {
	ctx := context.Background()
	gtraces := make([][]byte, 0)
	upload := func(ctx context.Context, traces [][]byte) error {
		gtraces = append(gtraces, traces...)
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	client := obs.NewHookTracerClient(clientCfg)
	provider, err := obs.NewTracerProvider(ctx, client, service, version)
	if err != nil {
		t.Fatal(err)
	}
	obs.InitObserver(ctx, provider, nil, client, nil, service, version, []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	})

	// start tracing
	putTestSpansBranch("", "")
	// stop tracing

	obs.Observer.Flush(ctx)
	checkSpansStructure(t, gtraces, "", "", 3)

	gtraces = make([][]byte, 0)

	// start new round tracing
	putTestSpansBranch("", "")
	// stop new round tracing

	obs.Observer.Close()
	checkSpansStructure(t, gtraces, "", "", 3)
}

func TestObserverTracesProxyAPI(t *testing.T) {
	ctx := context.Background()
	gtraces := make([][]byte, 0)
	upload := func(ctx context.Context, traces [][]byte) error {
		gtraces = append(gtraces, traces...)
		return nil
	}
	adoptiveClientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}
	adoptiveClient := obs.NewHookTracerClient(adoptiveClientCfg)
	hookClientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
	}
	hookClient := obs.NewHookTracerClient(hookClientCfg)
	proxyClient := obs.NewProxyTracerClient(adoptiveClient, hookClient)

	provider, err := obs.NewTracerProvider(ctx, proxyClient, service, version)
	if err != nil {
		t.Fatal(err)
	}
	obs.InitObserver(ctx, provider, nil, proxyClient, nil, service, version, []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	})

	// start tracing
	putTestSpansBranch("", "")
	// stop tracing

	obs.Observer.Close()
	checkSpansStructure(t, gtraces, "", "", 3)
}

func TestObserverTracesProxyAPIWithFlush(t *testing.T) {
	ctx := context.Background()
	gtraces := make([][]byte, 0)
	upload := func(ctx context.Context, traces [][]byte) error {
		gtraces = append(gtraces, traces...)
		return nil
	}
	adoptiveClientCfg := &obs.HookClientConfig{
		ResendTimeout:   5000000 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}
	adoptiveClient := obs.NewHookTracerClient(adoptiveClientCfg)
	hookClientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
	}
	hookClient := obs.NewHookTracerClient(hookClientCfg)
	proxyClient := obs.NewProxyTracerClient(adoptiveClient, hookClient)

	provider, err := obs.NewTracerProvider(ctx, proxyClient, service, version)
	if err != nil {
		t.Fatal(err)
	}
	obs.InitObserver(ctx, provider, nil, proxyClient, nil, service, version, []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	})

	// start tracing
	putTestSpansBranch("", "")
	// stop tracing

	obs.Observer.Flush(ctx)
	checkSpansStructure(t, gtraces, "", "", 3)

	gtraces = make([][]byte, 0)

	// start new round tracing
	putTestSpansBranch("", "")
	// stop new round tracing

	obs.Observer.Close()
	checkSpansStructure(t, gtraces, "", "", 3)
}

func TestObserverTracesAPIFixedTraceID(t *testing.T) {
	ctx := context.Background()
	gtraces := make([][]byte, 0)
	upload := func(ctx context.Context, traces [][]byte) error {
		gtraces = append(gtraces, traces...)
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	traceID := "d41d8cd98f00b204e9800998ecf8427e"
	pspanID := "8f00b204e9800998"
	client := obs.NewHookTracerClient(clientCfg)
	provider, err := obs.NewTracerProvider(ctx, client, service, version)
	if err != nil {
		t.Fatal(err)
	}
	obs.InitObserver(ctx, provider, nil, client, nil, service, version, []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	})

	// start tracing
	putTestSpansBranch(traceID, pspanID)
	// stop tracing

	obs.Observer.Close()
	checkSpansStructure(t, gtraces, traceID, pspanID, 3)
}

func TestObserverTracesAPIDoubleEndSpan(t *testing.T) {
	ctx := context.Background()
	gtraces := make([][]byte, 0)
	upload := func(ctx context.Context, traces [][]byte) error {
		gtraces = append(gtraces, traces...)
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	client := obs.NewHookTracerClient(clientCfg)
	provider, err := obs.NewTracerProvider(ctx, client, service, version)
	if err != nil {
		t.Fatal(err)
	}
	obs.InitObserver(ctx, provider, nil, client, nil, service, version, []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	})

	// start tracing
	rootCtx := context.Background()
	spanCtx, span := obs.Observer.NewSpan(rootCtx, oteltrace.SpanKindInternal, "span")
	logrus.WithContext(spanCtx).WithField("key", "value").Info("test info log message")
	span.End()
	span.End()
	// stop tracing

	obs.Observer.Close()
	checkSpansStructure(t, gtraces, "", "", 1)
}

func TestObserverTracesAPIEmptyBranch(t *testing.T) {
	ctx := context.Background()
	gtraces := make([][]byte, 0)
	upload := func(ctx context.Context, traces [][]byte) error {
		gtraces = append(gtraces, traces...)
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	client := obs.NewHookTracerClient(clientCfg)
	provider, err := obs.NewTracerProvider(ctx, client, service, version)
	if err != nil {
		t.Fatal(err)
	}
	obs.InitObserver(ctx, provider, nil, client, nil, service, version, []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	})

	// start tracing
	putEmptySpansBranch()
	// stop tracing

	obs.Observer.Close()

	if len(gtraces) != 0 {
		t.Fatalf("error on skipping empty branch spans when each entry hasn't contain events")
	}
}

func TestObserverTracesAPIMixedBranches(t *testing.T) {
	ctx := context.Background()
	gtraces := make([][]byte, 0)
	upload := func(ctx context.Context, traces [][]byte) error {
		gtraces = append(gtraces, traces...)
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	client := obs.NewHookTracerClient(clientCfg)
	provider, err := obs.NewTracerProvider(ctx, client, service, version)
	if err != nil {
		t.Fatal(err)
	}
	obs.InitObserver(ctx, provider, nil, client, nil, service, version, []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	})

	// start tracing
	putTestSpansBranch("", "")
	// empty branch
	putEmptySpansBranch()
	// stop tracing

	obs.Observer.Close()
	checkSpansStructure(t, gtraces, "", "", 3)
}
