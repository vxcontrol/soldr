package observability_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	"google.golang.org/protobuf/proto"

	obs "soldr/internal/observability"
)

type dumper struct{}

func (d *dumper) DumpStats() (map[string]float64, error) {
	return map[string]float64{
		"test_metric": 100,
	}, nil
}

func putTestMetricsSet(t *testing.T, meter obs.IMeter) {
	rootCtx := context.Background()

	{
		for met := int64(1); met < 10; met++ {
			counter, err := meter.NewInt64Counter("test-int64-counter")
			if err != nil {
				t.Errorf("error on creating int64 counter")
			}
			counter.Add(rootCtx, met)
		}
	}

	{
		direction := int64(1)
		for met := int64(1); met < 20; met++ {
			counter, err := meter.NewInt64UpDownCounter("test-int64-ud-counter")
			if err != nil {
				t.Errorf("error on creating int64 counter")
			}
			counter.Add(rootCtx, met*direction)
			direction *= -1
		}
	}

	{
		for met := int64(1); met < 30; met++ {
			histogram, err := meter.NewInt64Histogram("test-int64-histogram")
			if err != nil {
				t.Errorf("error on creating int64 histogram")
			}
			histogram.Record(rootCtx, met)
		}
	}

	{
		for met := float64(1); met < 40; met++ {
			counter, err := meter.NewFloat64Counter("test-float64-counter")
			if err != nil {
				t.Errorf("error on creating float64 counter")
			}
			counter.Add(rootCtx, met)
		}
	}

	{
		direction := float64(1)
		for met := float64(1); met < 50; met++ {
			counter, err := meter.NewFloat64UpDownCounter("test-float64-ud-counter")
			if err != nil {
				t.Errorf("error on creating float64 counter")
			}
			counter.Add(rootCtx, met*direction)
			direction *= -1
		}
	}

	{
		for met := float64(1); met < 60; met++ {
			histogram, err := meter.NewFloat64Histogram("test-float64-histogram")
			if err != nil {
				t.Errorf("error on creating float64 histogram")
			}
			histogram.Record(rootCtx, met)
		}
	}

	// TODO: it needs to take more guaranty write metrics
	time.Sleep(100 * time.Microsecond)
}

func checkCollectedMetrics(t *testing.T, mmetrics map[string][]*metricspb.Metric) {
	{
		sum := int64(0)
		array := getMetricsIntCounter(mmetrics["test-int64-counter"])
		for _, met := range array {
			sum += met
		}
		if sum != 45 {
			t.Fatalf("error on storing all metrics int64 counter")
		}
	}

	{
		sum := int64(0)
		array := getMetricsIntCounter(mmetrics["test-int64-ud-counter"])
		for _, met := range array {
			sum += met
		}
		if sum != 10 {
			t.Fatalf("error on storing all metrics int64 counter")
		}
	}

	{
		sum := uint64(0)
		array := getMetricsHistogram(mmetrics["test-int64-histogram"])
		for _, met := range array {
			sum += met
		}
		if sum != 29 {
			t.Fatalf("error on storing all metrics int64 histogram")
		}
	}

	{
		sum := float64(0)
		array := getMetricsFloatCounter(mmetrics["test-float64-counter"])
		for _, met := range array {
			sum += met
		}
		if sum != 780 {
			t.Fatalf("error on storing all metrics float64 counter")
		}
	}

	{
		sum := float64(0)
		array := getMetricsFloatCounter(mmetrics["test-float64-ud-counter"])
		for _, met := range array {
			sum += met
		}
		if sum != 25 {
			t.Fatalf("error on storing all metrics float64 counter")
		}
	}

	{
		sum := uint64(0)
		array := getMetricsHistogram(mmetrics["test-float64-histogram"])
		for _, met := range array {
			sum += met
		}
		if sum != 59 {
			t.Fatalf("error on storing all metrics float64 histogram")
		}
	}
}

func getMetrics(t *testing.T, metricsBuffers [][]byte) []*metricspb.Metric {
	metrics := make([]*metricspb.Metric, 0)
	metricsPackets := make([]*metricspb.ResourceMetrics, len(metricsBuffers))
	for idx, buffer := range metricsBuffers {
		metricPacket := &metricspb.ResourceMetrics{}
		if err := proto.Unmarshal(buffer, metricPacket); err != nil {
			t.Fatalf("failed to unmarshal metrics buffer")
		}
		metricsPackets[idx] = metricPacket
	}
	for _, metricPacket := range metricsPackets {
		instLibMetrics := metricPacket.GetInstrumentationLibraryMetrics()
		for _, instLibMetric := range instLibMetrics {
			metrics = append(metrics, instLibMetric.Metrics...)
		}
	}
	return metrics
}

func mapMetricsByName(metrics []*metricspb.Metric) map[string][]*metricspb.Metric {
	mapMetrics := make(map[string][]*metricspb.Metric)
	for _, metric := range metrics {
		mapMetrics[metric.GetName()] = append(mapMetrics[metric.GetName()], metric)
	}
	return mapMetrics
}

func getMetricsIntCounter(metrics []*metricspb.Metric) []int64 {
	array := make([]int64, 0)
	for _, metric := range metrics {
		if sum := metric.GetSum(); sum != nil {
			for _, point := range sum.GetDataPoints() {
				array = append(array, point.GetAsInt())
			}
		}
	}
	return array
}

func getMetricsFloatCounter(metrics []*metricspb.Metric) []float64 {
	array := make([]float64, 0)
	for _, metric := range metrics {
		if sum := metric.GetSum(); sum != nil {
			for _, point := range sum.GetDataPoints() {
				array = append(array, point.GetAsDouble())
			}
		}
	}
	return array
}

func getMetricsHistogram(metrics []*metricspb.Metric) []uint64 {
	array := make([]uint64, 0)
	for _, metric := range metrics {
		if hist := metric.GetHistogram(); hist != nil {
			for _, point := range hist.GetDataPoints() {
				array = append(array, point.Count)
			}
		}
	}
	return array
}

func TestObserverMetricsAPI(t *testing.T) {
	ctx := context.Background()
	gmetrics := make([][]byte, 0)
	upload := func(ctx context.Context, metrics [][]byte) error {
		gmetrics = append(gmetrics, metrics...)
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	client := obs.NewHookMeterClient(clientCfg)
	controller, err := obs.NewMeterProvider(ctx, client, "vxcommon", "v1.0.0-develop")
	if err != nil {
		t.Fatal(err)
	}
	obs.InitObserver(ctx, nil, controller, nil, client, "vxcommon", nil)

	// start metrics collecting
	putTestMetricsSet(t, obs.Observer)
	// stop metrics collecting

	obs.Observer.Close()

	if len(gmetrics) == 0 {
		t.Fatalf("error on catching metrics from observer API")
	}
	umetrics := getMetrics(t, gmetrics)
	if len(umetrics) != 6 {
		t.Fatalf("error on parsing metrics from uploaded data")
	}
	checkCollectedMetrics(t, mapMetricsByName(umetrics))
}

func TestObserverMetricsAPIGaugeCounter(t *testing.T) {
	ctx := context.Background()
	gmetrics := make([][]byte, 0)
	upload := func(ctx context.Context, metrics [][]byte) error {
		gmetrics = append(gmetrics, metrics...)
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	client := obs.NewHookMeterClient(clientCfg)
	controller, err := obs.NewMeterProvider(ctx, client, "vxcommon", "v1.0.0-develop")
	if err != nil {
		t.Fatal(err)
	}
	obs.InitObserver(ctx, nil, controller, nil, client, "vxcommon", nil)

	// start metrics collecting
	gaugeInt64Counter, err := obs.Observer.NewInt64GaugeCounter("test-int64-gauge-counter")
	if err != nil {
		t.Errorf("error on creating int64 gauge counter")
	}
	gaugeInt64Counter.Record(context.Background(), 123)
	gaugeInt64Counter.Record(context.Background(), 234)
	gaugeInt64Counter.Record(context.Background(), 345)

	gaugeFloat64Counter, err := obs.Observer.NewFloat64GaugeCounter("test-float64-gauge-counter")
	if err != nil {
		t.Errorf("error on creating float64 gauge counter")
	}
	gaugeFloat64Counter.Record(context.Background(), 12.3)
	gaugeFloat64Counter.Record(context.Background(), 23.4)
	gaugeFloat64Counter.Record(context.Background(), 34.5)
	// stop metrics collecting

	obs.Observer.Close()

	if len(gmetrics) == 0 {
		t.Fatalf("error on catching metrics from observer API")
	}
	umetrics := getMetrics(t, gmetrics)
	if len(umetrics) != 2 {
		t.Fatalf("error on parsing metrics from uploaded data")
	}
	for i := 0; i < len(umetrics); i++ {
		point := umetrics[i].GetGauge().DataPoints[0]
		if point.GetAsInt() != 345 && point.GetAsDouble() != 34.5 {
			t.Fatalf("error on getting last gauge counter value from datapoints")
		}
	}
}

func TestObserverMetricsAPIWithFlush(t *testing.T) {
	ctx := context.Background()
	gmetrics := make([][]byte, 0)
	upload := func(ctx context.Context, metrics [][]byte) error {
		gmetrics = append(gmetrics, metrics...)
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	client := obs.NewHookMeterClient(clientCfg)
	controller, err := obs.NewMeterProvider(ctx, client, "vxcommon", "v1.0.0-develop")
	if err != nil {
		t.Fatal(err)
	}
	obs.InitObserver(ctx, nil, controller, nil, client, "vxcommon", nil)

	// start metrics collecting
	putTestMetricsSet(t, obs.Observer)
	// stop metrics collecting

	if len(gmetrics) != 0 {
		t.Fatalf("error on waiting timeout to collect metrics")
	}

	obs.Observer.Flush(ctx)

	if len(gmetrics) == 0 {
		t.Fatalf("error on catching metrics from observer API")
	}
	umetrics := getMetrics(t, gmetrics)
	if len(umetrics) != 6 {
		t.Fatalf("error on parsing metrics from uploaded data")
	}
	checkCollectedMetrics(t, mapMetricsByName(umetrics))

	gmetrics = make([][]byte, 0)

	// start metrics collecting
	putTestMetricsSet(t, obs.Observer)
	// stop metrics collecting

	obs.Observer.Close()

	if len(gmetrics) == 0 {
		t.Fatalf("error on catching metrics from observer API")
	}
	if len(getMetrics(t, gmetrics)) != 6 {
		t.Fatalf("error on parsing metrics from uploaded data")
	}
}

func TestObserverMetricsAPIWithTimeout(t *testing.T) {
	ctx := context.Background()
	gmetrics := make([][]byte, 0)
	upload := func(ctx context.Context, metrics [][]byte) error {
		gmetrics = append(gmetrics, metrics...)
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	client := obs.NewHookMeterClient(clientCfg)
	controller, err := obs.NewMeterProvider(ctx, client, "vxcommon", "v1.0.0-develop")
	if err != nil {
		t.Fatal(err)
	}
	obs.InitObserver(ctx, nil, controller, nil, client, "vxcommon", nil)

	// start metrics collecting
	putTestMetricsSet(t, obs.Observer)
	// stop metrics collecting

	if len(gmetrics) != 0 {
		t.Fatalf("error on waiting timeout to collect metrics")
	}

	time.Sleep(11 * time.Second)

	if len(gmetrics) == 0 {
		t.Fatalf("error on catching metrics from observer API")
	}
	umetrics := getMetrics(t, gmetrics)
	if len(umetrics) != 6 {
		t.Fatalf("error on parsing metrics from uploaded data")
	}
	checkCollectedMetrics(t, mapMetricsByName(umetrics))

	obs.Observer.Close()
}

func TestObserverMetricsAPIRegistry(t *testing.T) {
	ctx := context.Background()
	gmetrics := make([][]byte, 0)
	upload := func(ctx context.Context, metrics [][]byte) error {
		gmetrics = append(gmetrics, metrics...)
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	client := obs.NewHookMeterClient(clientCfg)
	controller, err := obs.NewMeterProvider(ctx, client, "vxcommon", "v1.0.0-develop")
	if err != nil {
		t.Fatal(err)
	}
	obs.InitObserver(ctx, nil, controller, nil, client, "vxcommon", nil)

	registry, err := obs.NewMetricRegistry()
	if err != nil {
		t.Fatalf("error on making cache metrics registry")
	}

	// start metrics collecting
	putTestMetricsSet(t, registry)
	// stop metrics collecting

	obs.Observer.Close()

	if len(gmetrics) == 0 {
		t.Fatalf("error on catching metrics from observer API")
	}
	umetrics := getMetrics(t, gmetrics)
	if len(umetrics) != 6 {
		t.Fatalf("error on parsing metrics from uploaded data")
	}
	checkCollectedMetrics(t, mapMetricsByName(umetrics))
}

func TestObserverMetricsCollector(t *testing.T) {
	var d dumper
	ctx := context.Background()
	gmetrics := make([][]byte, 0)
	upload := func(ctx context.Context, metrics [][]byte) error {
		gmetrics = append(gmetrics, metrics...)
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	client := obs.NewHookMeterClient(clientCfg)
	controller, err := obs.NewMeterProvider(ctx, client, "vxcommon", "v1.0.0-develop")
	if err != nil {
		t.Fatal(err)
	}
	obs.InitObserver(ctx, nil, controller, nil, client, "vxcommon", nil)

	// start metrics collecting
	obs.Observer.StartDumperMetricCollect(&d, "vxcommon", "v1.0.0-develop")
	// stop metrics collecting

	obs.Observer.Close()

	if len(gmetrics) == 0 {
		t.Fatalf("error on catching metrics from observer API")
	}
	umetrics := getMetrics(t, gmetrics)
	if len(umetrics) != 1 {
		t.Fatalf("error on parsing metrics from uploaded data")
	}
	sum := float64(0)
	mmetrics := mapMetricsByName(umetrics)
	array := getMetricsFloatCounter(mmetrics["test_metric"])
	for _, met := range array {
		sum += met
	}
	if sum != 100 {
		t.Fatalf("error on storing test metric counter")
	}
}

func TestObserverMetricsProxyCollector(t *testing.T) {
	var d dumper
	ctx := context.Background()
	gmetrics := make([][]byte, 0)
	upload := func(ctx context.Context, metrics [][]byte) error {
		gmetrics = append(gmetrics, metrics...)
		return nil
	}
	adoptiveClientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}
	adoptiveClient := obs.NewHookMeterClient(adoptiveClientCfg)
	hookClientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
	}
	hookClient := obs.NewHookMeterClient(hookClientCfg)
	proxyClient := obs.NewProxyMeterClient(adoptiveClient, hookClient)
	controller, err := obs.NewMeterProvider(ctx, proxyClient, "vxcommon", "v1.0.0-develop")
	if err != nil {
		t.Fatal(err)
	}
	obs.InitObserver(ctx, nil, controller, nil, proxyClient, "vxcommon", nil)

	// start metrics collecting
	obs.Observer.StartDumperMetricCollect(&d, "vxcommon", "v1.0.0-develop")
	// stop metrics collecting

	obs.Observer.Close()

	if len(gmetrics) == 0 {
		t.Fatalf("error on catching metrics from observer API")
	}
	umetrics := getMetrics(t, gmetrics)
	if len(umetrics) != 1 {
		t.Fatalf("error on parsing metrics from uploaded data")
	}
	sum := float64(0)
	mmetrics := mapMetricsByName(umetrics)
	array := getMetricsFloatCounter(mmetrics["test_metric"])
	for _, met := range array {
		sum += met
	}
	if sum != 100 {
		t.Fatalf("error on storing test metric counter")
	}
}

func BenchmarkObserverMetricsAPI(b *testing.B) {
	ctx := context.Background()
	ncalls := 0
	nbytes := 0
	upload := func(ctx context.Context, metrics [][]byte) error {
		ncalls++
		for _, buf := range metrics {
			nbytes += len(buf)
		}
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	client := obs.NewHookMeterClient(clientCfg)
	controller, err := obs.NewMeterProvider(ctx, client, "vxcommon", "v1.0.0-develop")
	if err != nil {
		b.Fatal(err)
	}
	obs.InitObserver(ctx, nil, controller, nil, client, "vxcommon", nil)

	meter := obs.Observer
	direction := int64(1)
	measure := func(met int) error {
		if counter, err := meter.NewInt64UpDownCounter("test-int64-ud-counter"); err != nil {
			return fmt.Errorf("error on creating int64 counter")
		} else {
			counter.Add(ctx, int64(met)*direction)
		}

		if hist, err := meter.NewInt64Histogram("test-int64-histogram"); err != nil {
			return fmt.Errorf("error on creating int64 histogram")
		} else {
			hist.Record(ctx, int64(met)*direction)
		}

		direction *= -1
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := measure(i); err != nil {
			b.Fatal(err)
		}
	}

	obs.Observer.Close()

	b.Logf("bench result: %d calls upload func; %.0f avg bytes to call", ncalls, float64(nbytes)/float64(ncalls))
}

func BenchmarkObserverMetricsAPIRegistry(b *testing.B) {
	ctx := context.Background()
	ncalls := 0
	nbytes := 0
	upload := func(ctx context.Context, metrics [][]byte) error {
		ncalls++
		for _, buf := range metrics {
			nbytes += len(buf)
		}
		return nil
	}
	clientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}

	client := obs.NewHookMeterClient(clientCfg)
	controller, err := obs.NewMeterProvider(ctx, client, "vxcommon", "v1.0.0-develop")
	if err != nil {
		b.Fatal(err)
	}
	obs.InitObserver(ctx, nil, controller, nil, client, "vxcommon", nil)

	meter, err := obs.NewMetricRegistry()
	if err != nil {
		b.Fatalf("error on making cache metrics registry")
	}

	direction := int64(1)
	measure := func(met int) error {
		if counter, err := meter.NewInt64UpDownCounter("test-int64-ud-counter"); err != nil {
			return fmt.Errorf("error on creating int64 counter")
		} else {
			counter.Add(ctx, int64(met)*direction)
		}

		if hist, err := meter.NewInt64Histogram("test-int64-histogram"); err != nil {
			return fmt.Errorf("error on creating int64 histogram")
		} else {
			hist.Record(ctx, int64(met)*direction)
		}

		direction *= -1
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := measure(i); err != nil {
			b.Fatal(err)
		}
	}

	obs.Observer.Close()

	b.Logf("bench result: %d calls upload func; %.0f avg bytes to call", ncalls, float64(nbytes)/float64(ncalls))
}

func BenchmarkObserverMetricsProxyAPI(b *testing.B) {
	ctx := context.Background()
	ncalls := 0
	nbytes := 0
	upload := func(ctx context.Context, metrics [][]byte) error {
		ncalls++
		for _, buf := range metrics {
			nbytes += len(buf)
		}
		return nil
	}
	adoptiveClientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  upload,
	}
	adoptiveClient := obs.NewHookMeterClient(adoptiveClientCfg)
	hookClientCfg := &obs.HookClientConfig{
		ResendTimeout:   500 * time.Millisecond,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
	}
	hookClient := obs.NewHookMeterClient(hookClientCfg)
	proxyClient := obs.NewProxyMeterClient(adoptiveClient, hookClient)

	controller, err := obs.NewMeterProvider(ctx, proxyClient, "vxcommon", "v1.0.0-develop")
	if err != nil {
		b.Fatal(err)
	}
	obs.InitObserver(ctx, nil, controller, nil, proxyClient, "vxcommon", nil)

	meter := obs.Observer
	direction := int64(1)
	measure := func(met int) error {
		if counter, err := meter.NewInt64UpDownCounter("test-int64-ud-counter"); err != nil {
			return fmt.Errorf("error on creating int64 counter")
		} else {
			counter.Add(ctx, int64(met)*direction)
		}

		if hist, err := meter.NewInt64Histogram("test-int64-histogram"); err != nil {
			return fmt.Errorf("error on creating int64 histogram")
		} else {
			hist.Record(ctx, int64(met)*direction)
		}

		direction *= -1
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := measure(i); err != nil {
			b.Fatal(err)
		}
	}

	obs.Observer.Close()

	b.Logf("bench result: %d calls upload func; %.0f avg bytes to call", ncalls, float64(nbytes)/float64(ncalls))
}
