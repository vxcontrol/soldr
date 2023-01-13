package observability

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func startProcessMetricCollect(meter metric.Meter, attrs []attribute.KeyValue) error {
	proc := process.Process{
		Pid: int32(os.Getpid()),
	}

	collectRssMem := func(ctx context.Context, m metric.Int64ObserverResult) {
		procMemInfo, err := proc.MemoryInfoWithContext(ctx)
		if err != nil {
			logrus.WithContext(ctx).WithError(err).Errorf("failed to get process resident memory")
			return
		}
		m.Observe(int64(procMemInfo.RSS), attrs...)
	}
	collectVirtMem := func(ctx context.Context, m metric.Int64ObserverResult) {
		procMemInfo, err := proc.MemoryInfoWithContext(ctx)
		if err != nil {
			logrus.WithContext(ctx).WithError(err).Errorf("failed to get process virtual memory")
			return
		}
		m.Observe(int64(procMemInfo.VMS), attrs...)
	}
	collectCpuPercent := func(ctx context.Context, m metric.Float64ObserverResult) {
		procCpuPercent, err := proc.PercentWithContext(ctx, time.Duration(0))
		if err != nil {
			logrus.WithContext(ctx).WithError(err).Errorf("failed to get CPU usage percent")
			return
		}

		m.Observe(procCpuPercent, attrs...)
	}

	if _, err := proc.MemoryInfo(); err == nil {
		meter.NewInt64GaugeObserver("process_resident_memory_bytes", collectRssMem)
		meter.NewInt64GaugeObserver("process_virtual_memory_bytes", collectVirtMem)
	}
	if _, err := proc.Percent(time.Duration(0)); err == nil {
		meter.NewFloat64GaugeObserver("process_cpu_usage_percent", collectCpuPercent)
	}

	return nil
}

func startGoRuntimeMetricCollect(meter metric.Meter, attrs []attribute.KeyValue) error {
	var (
		lastUpdate         time.Time = time.Now()
		mx                 sync.Mutex
		procRuntimeMemStat runtime.MemStats
	)
	runtime.ReadMemStats(&procRuntimeMemStat)

	getMemStats := func() *runtime.MemStats {
		mx.Lock()
		defer mx.Unlock()

		now := time.Now()
		if now.Sub(lastUpdate) > defCollectPeriod {
			runtime.ReadMemStats(&procRuntimeMemStat)
		}
		lastUpdate = now
		return &procRuntimeMemStat
	}

	meter.NewInt64GaugeObserver("go_cgo_calls", func(ctx context.Context, m metric.Int64ObserverResult) {
		m.Observe(runtime.NumCgoCall(), attrs...)
	})
	meter.NewInt64GaugeObserver("go_goroutines", func(ctx context.Context, m metric.Int64ObserverResult) {
		m.Observe(int64(runtime.NumGoroutine()), attrs...)
	})
	meter.NewInt64GaugeObserver("go_heap_objects_bytes", func(ctx context.Context, m metric.Int64ObserverResult) {
		m.Observe(int64(getMemStats().HeapInuse), attrs...)
	})
	meter.NewInt64GaugeObserver("go_heap_objects_counter", func(ctx context.Context, m metric.Int64ObserverResult) {
		m.Observe(int64(getMemStats().HeapObjects), attrs...)
	})
	meter.NewInt64GaugeObserver("go_stack_inuse_bytes", func(ctx context.Context, m metric.Int64ObserverResult) {
		m.Observe(int64(getMemStats().StackInuse), attrs...)
	})
	meter.NewInt64GaugeObserver("go_stack_sys_bytes", func(ctx context.Context, m metric.Int64ObserverResult) {
		m.Observe(int64(getMemStats().StackSys), attrs...)
	})
	meter.NewInt64GaugeObserver("go_total_allocs_bytes", func(ctx context.Context, m metric.Int64ObserverResult) {
		m.Observe(int64(getMemStats().TotalAlloc), attrs...)
	})
	meter.NewInt64GaugeObserver("go_heap_allocs_bytes", func(ctx context.Context, m metric.Int64ObserverResult) {
		m.Observe(int64(getMemStats().HeapAlloc), attrs...)
	})
	meter.NewInt64GaugeObserver("go_pause_gc_total_nanosec", func(ctx context.Context, m metric.Int64ObserverResult) {
		m.Observe(int64(getMemStats().PauseTotalNs), attrs...)
	})

	return nil
}

func startDumperMetricCollect(stats IDumper, meter metric.Meter, attrs []attribute.KeyValue) error {
	var (
		err        error
		lastStats  map[string]float64
		lastUpdate time.Time = time.Now()
		mx         sync.Mutex
	)

	if lastStats, err = stats.DumpStats(); err != nil {
		logrus.WithError(err).Errorf("failed to get stats dump")
		return err
	}

	getProtoStats := func() map[string]float64 {
		mx.Lock()
		defer mx.Unlock()

		now := time.Now()
		if now.Sub(lastUpdate) <= defCollectPeriod {
			return lastStats
		}
		if lastStats, err = stats.DumpStats(); err != nil {
			return lastStats
		}
		lastUpdate = now
		return lastStats
	}

	for key := range lastStats {
		metricName := key
		meter.NewFloat64CounterObserver(metricName, func(ctx context.Context, m metric.Float64ObserverResult) {
			if value, ok := getProtoStats()[metricName]; ok {
				m.Observe(value, attrs...)
			}
		})
	}

	return nil
}
