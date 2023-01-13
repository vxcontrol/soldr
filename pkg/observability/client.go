package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

const (
	DefaultResendTimeout   = 5 * time.Second
	DefaultQueueSizeLimit  = 10 * 1024 * 1024 // 10 MB
	DefaultPacketSizeLimit = 100 * 1024       // 100 KB
)

const (
	defUploadChan   = 2048
	defReconnPeriod = 5 * time.Second
)

// HookClientConfig is a custom struct to configure mocking otlp clients
type HookClientConfig struct {
	ResendTimeout   time.Duration                         `json:"resend_timeout"`
	QueueSizeLimit  int                                   `json:"queue_size_limit"`
	PacketSizeLimit int                                   `json:"packet_size_limit"`
	UploadCallback  func(context.Context, [][]byte) error `json:"-"`
}

// hookClient is a custom client to implement traces and metrics client interfaces
type hookClient struct {
	cfg        *HookClientConfig
	queue      [][]byte
	stopped    bool
	stoppedMX  *sync.RWMutex
	flushChan  chan struct{}
	uploadChan chan []byte
	startChan  chan struct{}
	stopChan   chan struct{}
}

func newHookClient(cfg *HookClientConfig) *hookClient {
	return &hookClient{
		cfg:        cfg,
		queue:      make([][]byte, 0),
		stopped:    true,
		stoppedMX:  &sync.RWMutex{},
		flushChan:  make(chan struct{}),
		uploadChan: make(chan []byte, defUploadChan),
		startChan:  make(chan struct{}),
		stopChan:   make(chan struct{}),
	}
}

func (hc *hookClient) flush(ctx context.Context) {
	if len(hc.queue) == 0 {
		return
	}

	defer func() {
		var total int
		for idx := len(hc.queue) - 1; idx >= 0; idx-- {
			total += len(hc.queue[idx])
			if total > hc.cfg.QueueSizeLimit {
				hc.queue = hc.queue[idx+1:]
				break
			}
		}
	}()

	if hc.cfg.UploadCallback == nil {
		return
	}

	var (
		partNum    int
		packetSize int
		packetData = make([][]byte, 0, len(hc.queue))
	)
	upload := func() bool {
		if hc.cfg.UploadCallback == nil {
			return false
		}
		if hc.cfg.UploadCallback(ctx, packetData) != nil {
			return false
		}
		hc.queue = hc.queue[partNum:]
		packetData = packetData[:0]
		packetSize = 0
		partNum = 0
		return true
	}
	for range hc.queue {
		partSize := len(hc.queue[partNum])
		if packetSize+partSize > hc.cfg.PacketSizeLimit && partNum != 0 {
			if !upload() {
				return
			}
		}
		packetData = append(packetData, hc.queue[partNum])
		packetSize += partSize
		partNum++
	}
	if partNum != 0 {
		upload()
	}
}

func (hc *hookClient) uploader() {
	ticker := time.NewTicker(hc.cfg.ResendTimeout)
	defer ticker.Stop()

	readAllUploads := func() {
		nchan := len(hc.uploadChan)
		for idx := 0; idx < nchan; idx++ {
			hc.queue = append(hc.queue, <-hc.uploadChan)
		}
	}

	hc.startChan <- struct{}{}
	for {
		select {
		case <-ticker.C:
			hc.flush(context.Background())
		case <-hc.flushChan:
			ticker.Reset(hc.cfg.ResendTimeout)
			hc.flush(context.Background())
			hc.flushChan <- struct{}{}
		case tr := <-hc.uploadChan:
			ticker.Reset(hc.cfg.ResendTimeout)
			hc.queue = append(hc.queue, tr)
			readAllUploads()
		case <-hc.stopChan:
			readAllUploads()
			close(hc.stopChan)
			return
		}
	}
}

// Flush is function to force synchronous call for upload callback
func (hc *hookClient) Flush(ctx context.Context) error {
	hc.stoppedMX.RLock()
	defer hc.stoppedMX.RUnlock()
	if hc.stopped {
		return fmt.Errorf("client has already stopped")
	}

	hc.flushChan <- struct{}{}
	// make synchronous flush method for next upload calls
	<-hc.flushChan

	return nil
}

// Start is function to run uploader routine for caching tracing and metrics
func (hc *hookClient) Start(ctx context.Context) error {
	hc.stoppedMX.Lock()
	defer hc.stoppedMX.Unlock()
	if !hc.stopped {
		return fmt.Errorf("client has already started")
	}

	go hc.uploader()
	// wait for start uploader routine
	<-hc.startChan
	hc.stopped = false

	return nil
}

// Stop is function to stop uploader routine
func (hc *hookClient) Stop(ctx context.Context) error {
	hc.stoppedMX.Lock()
	defer hc.stoppedMX.Unlock()
	if hc.stopped {
		return fmt.Errorf("client has already stopped")
	}

	// stopChan is blocking, so the following works fine
	hc.stopChan <- struct{}{}
	// wait for stop uploader routine
	<-hc.stopChan
	hc.stopChan = make(chan struct{})
	hc.flush(ctx)
	hc.stopped = true

	return nil
}

// UploadTraces is function to transfer traces and spans via another transport
func (hc *hookClient) UploadTraces(ctx context.Context, protoSpans []*tracepb.ResourceSpans) error {
	hc.stoppedMX.RLock()
	defer hc.stoppedMX.RUnlock()
	if hc.stopped {
		return fmt.Errorf("client was stopped")
	}

	for _, span := range protoSpans {
		pkg, err := proto.Marshal(span)
		if err != nil {
			return err
		}
		hc.uploadChan <- pkg
	}

	return nil
}

// UploadMetrics is function to transfer metrics via another transport
func (hc *hookClient) UploadMetrics(ctx context.Context, protoMetrics []*metricspb.ResourceMetrics) error {
	hc.stoppedMX.RLock()
	defer hc.stoppedMX.RUnlock()
	if hc.stopped {
		return fmt.Errorf("client was stopped")
	}

	for _, metric := range protoMetrics {
		pkg, err := proto.Marshal(metric)
		if err != nil {
			return err
		}
		hc.uploadChan <- pkg
	}

	return nil
}
