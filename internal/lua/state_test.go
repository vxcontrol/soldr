package lua_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"runtime"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"

	"soldr/internal/lua"
	obs "soldr/internal/observability"
)

func init() {
	logrus.SetOutput(ioutil.Discard)
}

func initObserver(uploadTraces, uploadMetrics func(context.Context, [][]byte) error) error {
	ctx := context.Background()
	clientTracesCfg := &obs.HookClientConfig{
		ResendTimeout:   obs.DefaultResendTimeout,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  uploadTraces,
	}
	clientMetricsCfg := &obs.HookClientConfig{
		ResendTimeout:   obs.DefaultResendTimeout,
		QueueSizeLimit:  obs.DefaultQueueSizeLimit,
		PacketSizeLimit: obs.DefaultPacketSizeLimit,
		UploadCallback:  uploadMetrics,
	}

	clientTraces := obs.NewHookTracerClient(clientTracesCfg)
	provider, err := obs.NewTracerProvider(ctx, clientTraces, "vxcommon", "v1.0.0-develop")
	if err != nil {
		return err
	}
	clientMetric := obs.NewHookMeterClient(clientMetricsCfg)
	controller, err := obs.NewMeterProvider(ctx, clientMetric, "vxcommon", "v1.0.0-develop")
	if err != nil {
		return err
	}

	obs.InitObserver(ctx, provider, controller, clientTraces, clientMetric, "vxcommon", []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
		logrus.TraceLevel,
	})
	logrus.SetLevel(logrus.DebugLevel)
	return nil
}

func checkEventAttributes(event *tracepb.Span_Event) error {
	for _, attr := range event.GetAttributes() {
		switch attr.GetKey() {
		case "log.severity":
			severity := attr.GetValue().GetStringValue()
			switch severity {
			case "ERROR", "WARN", "INFO", "DEBUG":
			default:
				return fmt.Errorf("mismatch event severity: '%s'", severity)
			}
		case "log.tmpdir":
		case "log.module":
		case "log.group_id":
		case "log.component":
			component := attr.GetValue().GetStringValue()
			if component != "lua_state" && !strings.HasPrefix(component, "lua_module_") {
				return fmt.Errorf("mismatch event component: '%s'", component)
			}
		case "log.message":
			msg := attr.GetValue().GetStringValue()
			switch msg {
			case "the state was created", "the state was started", "the state was stopped":
			case "the state loaded clibs", "the state loaded data", "the state was destroyed":
			case "[some debug]", "[some info]", "[some warn]", "[some error]",
				"[some debug argument]", "[some info argument]",
				"[some warn argument]", "[some error argument]",
				"[some debug: message]", "[some info: message]",
				"[some warn: message]", "[some error: message]":
			default:
				return fmt.Errorf("mismatch event message: '%s'", msg)
			}
		default:
			return fmt.Errorf("mismatch event key: '%s'", attr.GetKey())
		}
	}
	return nil
}

func checkSpans(spans []*tracepb.Span) error {
	for _, span := range spans {
		for _, event := range span.GetEvents() {
			if event.GetName() != "log" {
				return fmt.Errorf("expected event name 'log', got %s", event.GetName())
			}
			if err := checkEventAttributes(event); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkTraces(gtraces [][]byte) error {
	rtraces := make([]*tracepb.ResourceSpans, len(gtraces))
	for idx, tspans := range gtraces {
		rtraces[idx] = &tracepb.ResourceSpans{}
		if proto.Unmarshal(tspans, rtraces[idx]) != nil {
			return fmt.Errorf("error on unmarshaling span list for traces")
		}
	}

	if len(gtraces) != 1 || len(rtraces) != 1 {
		return fmt.Errorf("error on slicing branches spans according root spans")
	}
	instLibSpans := rtraces[0].GetInstrumentationLibrarySpans()
	if len(instLibSpans) != 1 {
		return fmt.Errorf("error on slicing branches spans according calls")
	}
	spans := instLibSpans[0].GetSpans()
	if len(spans) == 0 {
		return fmt.Errorf("error on storing all spans into the span branch (empty span list)")
	}
	if err := checkSpans(spans); err != nil {
		return err
	}
	return nil
}

func checkMetrics(metricsBuffers [][]byte) error {
	metrics := make([]*metricspb.Metric, 0)
	metricsPackets := make([]*metricspb.ResourceMetrics, len(metricsBuffers))
	for idx, buffer := range metricsBuffers {
		metricPacket := &metricspb.ResourceMetrics{}
		if err := proto.Unmarshal(buffer, metricPacket); err != nil {
			return fmt.Errorf("failed to unmarshal metrics buffer")
		}
		metricsPackets[idx] = metricPacket
	}
	for _, metricPacket := range metricsPackets {
		instLibMetrics := metricPacket.GetInstrumentationLibraryMetrics()
		for _, instLibMetric := range instLibMetrics {
			metrics = append(metrics, instLibMetric.Metrics...)
		}
	}
	mapMetrics := make(map[string][]*metricspb.Metric)
	for _, metric := range metrics {
		mapMetrics[metric.GetName()] = append(mapMetrics[metric.GetName()], metric)
	}

	{
		sum := int64(0)
		array := getMetricsIntCounter(mapMetrics["test-int64-counter"])
		for _, met := range array {
			sum += met
		}
		if sum != 100 {
			return fmt.Errorf("error on storing all metrics int64 counter")
		}
	}

	{
		sum := int64(0)
		array := getMetricsIntCounter(mapMetrics["test-int64-ud-counter"])
		for _, met := range array {
			sum += met
		}
		if sum != 50 {
			return fmt.Errorf("error on storing all metrics int64 updown counter")
		}
	}

	{
		array := getMetricsIntGauge(mapMetrics["test-int64-gg-counter"])
		// here using last logged value for the last 5 seconds
		if len(array) != 1 {
			return fmt.Errorf("error on getting last value for int64 gauge counter")
		}
		if array[0] != -40 {
			return fmt.Errorf("error on storing all metrics int64 gauge counter")
		}
	}

	{
		sum := uint64(0)
		array := getMetricsHistogram(mapMetrics["test-int64-histogram"])
		for _, met := range array {
			sum += met
		}
		if sum != 2 {
			return fmt.Errorf("error on storing all metrics int64 histogram")
		}
	}

	{
		sum := float64(0)
		array := getMetricsFloatCounter(mapMetrics["test-float64-counter"])
		for _, met := range array {
			sum += met
		}
		if sum != 100.5 {
			return fmt.Errorf("error on storing all metrics float64 counter")
		}
	}

	{
		sum := float64(0)
		array := getMetricsFloatCounter(mapMetrics["test-float64-ud-counter"])
		for _, met := range array {
			sum += met
		}
		if sum != 50.4 {
			return fmt.Errorf("error on storing all metrics float64 updown counter")
		}
	}

	{
		array := getMetricsFloatGauge(mapMetrics["test-float64-gg-counter"])
		// here using last logged value for the last 5 seconds
		if len(array) != 1 {
			return fmt.Errorf("error on getting last value for float64 gauge counter")
		}
		if array[0] != -50.3 {
			return fmt.Errorf("error on storing all metrics float64 gauge counter")
		}
	}

	{
		sum := uint64(0)
		array := getMetricsHistogram(mapMetrics["test-float64-histogram"])
		for _, met := range array {
			sum += met
		}
		if sum != 2 {
			return fmt.Errorf("error on storing all metrics float64 histogram")
		}
	}

	return nil
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

func getMetricsIntGauge(metrics []*metricspb.Metric) []int64 {
	array := make([]int64, 0)
	for _, metric := range metrics {
		if gauge := metric.GetGauge(); gauge != nil {
			for _, point := range gauge.GetDataPoints() {
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

func getMetricsFloatGauge(metrics []*metricspb.Metric) []float64 {
	array := make([]float64, 0)
	for _, metric := range metrics {
		if gauge := metric.GetGauge(); gauge != nil {
			for _, point := range gauge.GetDataPoints() {
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

// Run simple test with loading state
func TestLoadStateWithMainModule(t *testing.T) {
	files := map[string][]byte{
		"main.lua": []byte(`return 'success'`),
	}

	state, err := lua.NewState(files)
	if err != nil {
		t.Error("Error with creating new state: ", err)
		t.Fail()
	}

	result, err := state.Exec()
	if err != nil {
		t.Error("Error with executing state: ", err)
		t.Fail()
	}
	if result != "success" {
		t.Error("Error with getting result: ", result)
		t.Fail()
	}
}

// Run simple test with loading other module
func TestLoadStateWithOtherModule(t *testing.T) {
	files := map[string][]byte{
		"main.lua":     []byte(`return require('mymodule')`),
		"mymodule.lua": []byte(`return 'success'`),
	}

	state, err := lua.NewState(files)
	if err != nil {
		t.Error("Error with creating new state: ", err)
		t.Fail()
	}

	result, err := state.Exec()
	if err != nil {
		t.Error("Error with executing state: ", err)
		t.Fail()
	}
	if result != "success" {
		t.Error("Error with getting result: ", result)
		t.Fail()
	}
}

// Run test with logging
func TestLoadStateWithLogging(t *testing.T) {
	gtraces := make([][]byte, 0)
	uploadTraces := func(ctx context.Context, traces [][]byte) error {
		gtraces = append(gtraces, traces...)
		return nil
	}
	gmetrics := make([][]byte, 0)
	uploadMetrics := func(ctx context.Context, metrics [][]byte) error {
		gmetrics = append(gmetrics, metrics...)
		return nil
	}
	files := map[string][]byte{
		"main.lua": []byte(`
			__log.debug("some debug")
			__log.info("some info")
			__log.warn("some warn")
			__log.error("some error")

			__log.debug("some debug", "argument")
			__log.info("some info", "argument")
			__log.warn("some warn", "argument")
			__log.error("some error", "argument")

			__log.debugf("some debug: %s", "message")
			__log.infof("some info: %s", "message")
			__log.warnf("some warn: %s", "message")
			__log.errorf("some error: %s", "message")

			__metric.add_int_counter("test-int64-counter", 100)
			__metric.add_int_gauge_counter("test-int64-gg-counter", 100)
			__metric.add_int_gauge_counter("test-int64-gg-counter", -40)
			__metric.add_int_updown_counter("test-int64-ud-counter", 100)
			__metric.add_int_updown_counter("test-int64-ud-counter", -50)
			__metric.add_int_histogram("test-int64-histogram", 100)
			__metric.add_int_histogram("test-int64-histogram", 50)

			__metric.add_float_counter("test-float64-counter", 100.5)
			__metric.add_float_gauge_counter("test-float64-gg-counter", 100.5)
			__metric.add_float_gauge_counter("test-float64-gg-counter", -50.3)
			__metric.add_float_updown_counter("test-float64-ud-counter", 100.5)
			__metric.add_float_updown_counter("test-float64-ud-counter", -50.1)
			__metric.add_float_histogram("test-float64-histogram", 100.5)
			__metric.add_float_histogram("test-float64-histogram", 50.2)

			return 'success'
		`),
	}

	if err := initObserver(uploadTraces, uploadMetrics); err != nil {
		t.Fatal(err)
	}

	state, err := lua.NewState(files)
	if err != nil {
		t.Error("Error with creating new state: ", err)
		t.Fail()
	}

	state.RegisterLogger(logrus.TraceLevel, logrus.Fields{})
	state.RegisterMeter(logrus.Fields{})

	result, err := state.Exec()
	if err != nil {
		t.Error("Error with executing state: ", err)
		t.Fail()
	}
	if result != "success" {
		t.Error("Error with getting result: ", result)
		t.Fail()
	}

	obs.Observer.SpanFromContext(state.Context()).End()

	obs.Observer.Close()

	if err = checkTraces(gtraces); err != nil {
		t.Error("Error with checking traces: ", err)
		t.Fail()
	}
	if err = checkMetrics(gmetrics); err != nil {
		t.Error("Error with checking metrics: ", err)
		t.Fail()
	}
}

func BenchmarkLuaLoadStateWithMainModule(b *testing.B) {
	for i := 0; i < b.N; i++ {
		files := map[string][]byte{
			"main.lua": []byte(`return 'success'`),
		}

		state, err := lua.NewState(files)
		if err != nil {
			b.Fatal("Error with creating new state: ", err)
		}

		result, err := state.Exec()
		if err != nil {
			b.Fatal("Error with executing state: ", err)
		}
		if result != "success" {
			b.Fatal("Error with getting result: ", result)
		}

		// force using GC
		runtime.GC()
	}
}

func BenchmarkLuaLoadStateWithLargeFile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		files := map[string][]byte{
			"main.lua":     []byte(`return 'success'`),
			"big_file.dat": make([]byte, 10*1024*1024),
		}

		state, err := lua.NewState(files)
		if err != nil {
			b.Fatal("Error with creating new state: ", err)
		}

		result, err := state.Exec()
		if err != nil {
			b.Fatal("Error with executing state: ", err)
		}
		if result != "success" {
			b.Fatal("Error with getting result: ", result)
		}

		// force using GC
		runtime.GC()
	}
}
