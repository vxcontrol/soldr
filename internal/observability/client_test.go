package observability_test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	respb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	obs "soldr/internal/observability"
)

const (
	defUploadChan = 256
	service       = "vxcommon"
	version       = "v1.0.0-develop"
)

func makeResourceSpans(num int) []*tracepb.ResourceSpans {
	spans := make([]*tracepb.ResourceSpans, 0)
	for idx := 0; idx < num; idx++ {
		spans = append(spans, &tracepb.ResourceSpans{
			Resource: &respb.Resource{},
			InstrumentationLibrarySpans: []*tracepb.InstrumentationLibrarySpans{
				{},
			},
		})
	}
	return spans
}

func makeResourceMetrics(num int) []*metricspb.ResourceMetrics {
	metrics := make([]*metricspb.ResourceMetrics, 0)
	for idx := 0; idx < num; idx++ {
		metrics = append(metrics, &metricspb.ResourceMetrics{
			Resource: &respb.Resource{},
			InstrumentationLibraryMetrics: []*metricspb.InstrumentationLibraryMetrics{
				{},
			},
		})
	}
	return metrics
}

func getResourceSize(res protoreflect.ProtoMessage) int {
	pkg, err := proto.Marshal(res)
	if err != nil {
		return 0
	}
	return len(pkg)
}

func Example_newHookTracesClient() {
	ctx := context.Background()
	gtraces := make([][]byte, 0)
	upload := func(ctx context.Context, traces [][]byte) error {
		gtraces = append(gtraces, traces...)
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   obs.DefaultResendTimeout,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	client := obs.NewHookTracerClient(clientCfg)

	if err := client.Start(ctx); err != nil {
		log.Fatalf("failed to start client")
	}

	if err := client.UploadTraces(ctx, makeResourceSpans(defUploadChan)); err != nil {
		log.Fatalf("failed to upload traces")
	}

	if err := client.Stop(ctx); err != nil {
		log.Fatalf("failed to stop client")
	}

	if len(gtraces) == defUploadChan {
		fmt.Println("success")
	} else {
		log.Fatalf("failed to read all traces")
	}
	// Output:
	//success
}

func Example_newHookMetricsClient() {
	ctx := context.Background()
	gmetrics := make([][]byte, 0)
	upload := func(ctx context.Context, metrics [][]byte) error {
		gmetrics = append(gmetrics, metrics...)
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   obs.DefaultResendTimeout,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	client := obs.NewHookMeterClient(clientCfg)

	if err := client.Start(ctx); err != nil {
		log.Fatalf("failed to start client")
	}

	if err := client.UploadMetrics(ctx, makeResourceMetrics(defUploadChan)); err != nil {
		log.Fatalf("failed to upload metrics")
	}

	if err := client.Stop(ctx); err != nil {
		log.Fatalf("failed to stop client")
	}

	if len(gmetrics) == defUploadChan {
		fmt.Println("success")
	} else {
		log.Fatalf("failed to read all metrics")
	}
	// Output:
	//success
}

func TestReopenHookClient(t *testing.T) {
	ctx := context.Background()
	gtraces := make([][]byte, 0)
	upload := func(ctx context.Context, traces [][]byte) error {
		gtraces = append(gtraces, traces...)
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   obs.DefaultResendTimeout,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	client := obs.NewHookTracerClient(clientCfg)

	nreopen := 5
	for idx := 0; idx < nreopen; idx++ {
		if err := client.Start(ctx); err != nil {
			t.Fatalf("failed to start client")
		}

		if err := client.UploadTraces(ctx, makeResourceSpans(defUploadChan)); err != nil {
			t.Fatalf("failed to upload traces")
		}

		if err := client.Stop(ctx); err != nil {
			t.Fatalf("failed to stop client")
		}
	}

	if len(gtraces) != defUploadChan*nreopen {
		t.Fatalf("failed to read all traces")
	}
}

func TestSendTimeout(t *testing.T) {
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

	if err := client.Start(ctx); err != nil {
		t.Fatalf("failed to start client")
	}

	if err := client.UploadTraces(ctx, makeResourceSpans(defUploadChan)); err != nil {
		t.Fatalf("failed to upload traces")
	}

	if len(gtraces) != 0 {
		t.Fatalf("failed to match uploaded traces before using upload api call")
	}

	time.Sleep(time.Second)

	if len(gtraces) != defUploadChan {
		t.Fatalf("failed to read all traces after exceeded upload timeout")
	}

	if err := client.Stop(ctx); err != nil {
		t.Fatalf("failed to stop client")
	}
}

func TestResendTracesAfterCatchError(t *testing.T) {
	ctx := context.Background()
	gtraces := make([][]byte, 0)
	needRaiseError := true
	upload := func(ctx context.Context, traces [][]byte) error {
		if needRaiseError {
			needRaiseError = false
			return fmt.Errorf("some sending traces error")
		}
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

	if err := client.Start(ctx); err != nil {
		t.Fatalf("failed to start client")
	}

	if err := client.UploadTraces(ctx, makeResourceSpans(defUploadChan)); err != nil {
		t.Fatalf("failed to upload traces")
	}

	time.Sleep(800 * time.Millisecond)

	if len(gtraces) != 0 {
		t.Fatalf("failed to match uploaded traces when raised error")
	}

	time.Sleep(500 * time.Millisecond)

	if len(gtraces) != defUploadChan {
		t.Fatalf("failed to read all traces after correct response from upload callback")
	}

	if err := client.Stop(ctx); err != nil {
		t.Fatalf("failed to stop client")
	}
}

func TestPacketSizeLimit(t *testing.T) {
	ctx := context.Background()
	gtraces := make([][]byte, 0)
	ncalls := 0
	upload := func(ctx context.Context, traces [][]byte) error {
		gtraces = append(gtraces, traces...)
		ncalls++
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   obs.DefaultResendTimeout,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: getResourceSize(makeResourceSpans(1)[0]) * 16,
		UploadCallback:  upload,
	}

	client := obs.NewHookTracerClient(clientCfg)

	if err := client.Start(ctx); err != nil {
		t.Fatalf("failed to start client")
	}

	if err := client.UploadTraces(ctx, makeResourceSpans(defUploadChan)); err != nil {
		t.Fatalf("failed to upload traces")
	}

	if err := client.Stop(ctx); err != nil {
		t.Fatalf("failed to stop client")
	}

	if len(gtraces) != defUploadChan {
		t.Fatalf("failed to read all traces")
	}

	if ncalls != defUploadChan/16 {
		t.Fatalf("failed to catch all upload calls")
	}
}

func TestQueueSizeLimit(t *testing.T) {
	ctx := context.Background()
	gtraces := make([][]byte, 0)
	needRaiseError := true
	upload := func(ctx context.Context, traces [][]byte) error {
		if needRaiseError {
			needRaiseError = false
			return fmt.Errorf("some sending traces error")
		}
		gtraces = append(gtraces, traces...)
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  getResourceSize(makeResourceSpans(1)[0]) * 16,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	client := obs.NewHookTracerClient(clientCfg)

	if err := client.Start(ctx); err != nil {
		t.Fatalf("failed to start client")
	}

	if err := client.UploadTraces(ctx, makeResourceSpans(defUploadChan)); err != nil {
		t.Fatalf("failed to upload traces")
	}

	time.Sleep(800 * time.Millisecond)

	if len(gtraces) != 0 {
		t.Fatalf("failed to match uploaded traces when raised error")
	}

	time.Sleep(500 * time.Millisecond)

	if len(gtraces) != 16 {
		t.Fatalf("failed to read all traces after shrink queue size")
	}

	if err := client.Stop(ctx); err != nil {
		t.Fatalf("failed to stop client")
	}
}
