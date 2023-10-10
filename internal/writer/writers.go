package writer

import (
	"fmt"
	"github.com/go-logr/logr"
	types2 "github.com/noovertime7/kubemonitor/pkg/types"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/prometheus/prompb"
)

// Writers manage all writers and metric queue
type (
	Writers struct {
		writerMap map[string]Writer
		queue     *types2.SafeListLimited[*prompb.TimeSeries]
		logger    logr.Logger
		sync.Mutex

		Snapshot
	}

	Snapshot struct {
		FailCount  uint64
		FailTotal  uint64
		TotalCount uint64

		QueueSize uint64
	}
)

type WritersManager interface {
	Register(name string, opt WriterOption) error
	DeRegister(name string) error
	WriteSamples(samples []*types2.Sample)
	QueueMetrics() *Snapshot
	WriteTimeSeries(timeSeries []prompb.TimeSeries)
}

func NewWriter(maxSize int, logger logr.Logger) WritersManager {
	writers := &Writers{
		logger:    logger,
		writerMap: make(map[string]Writer),
		queue:     types2.NewSafeListLimited[*prompb.TimeSeries](maxSize),
	}
	go writers.loopRead()
	return writers
}

func (ws *Writers) DeRegister(name string) error {
	ws.Lock()
	defer ws.Unlock()

	_, has := ws.writerMap[name]
	if has {
		delete(ws.writerMap, name)
		return nil
	}
	return fmt.Errorf("%s not register wirters", name)
}

func (ws *Writers) Register(name string, opt WriterOption) error {
	ws.Lock()
	defer ws.Unlock()

	writer, err := newWriter(opt)
	if err != nil {
		return err
	}

	ws.writerMap[name] = writer

	return nil
}

func (ws *Writers) loopRead() {
	for {
		series := ws.queue.PopBackN(10)
		if len(series) == 0 {
			time.Sleep(time.Millisecond * 400)
			continue
		}

		items := make([]prompb.TimeSeries, len(series))
		for i := 0; i < len(series); i++ {
			items[i] = *series[i]
		}

		ws.WriteTimeSeries(items)
	}
}

// WriteSamples convert samples to []prompb.TimeSeries and batch write to queue
func (ws *Writers) WriteSamples(samples []*types2.Sample) {
	if len(samples) == 0 {
		return
	}
	//if config.Config.TestMode {
	//	printTestMetrics(samples)
	//	return
	//}
	//if config.Config.DebugMode {
	//	printTestMetrics(samples)
	//}

	items := make([]*prompb.TimeSeries, 0, len(samples))
	for _, sample := range samples {
		item := sample.ConvertTimeSeries("ms")
		if item == nil || len(item.Labels) == 0 {
			continue
		}
		items = append(items, item)
	}
	success := ws.queue.PushFrontN(items)
	l := ws.queue.Len()
	if !success {
		log.Printf("E! write %d samples failed, please increase queue size(%d)", len(items), l)
	}
	go ws.snapshot(uint64(len(items)), uint64(l), success)
}

func (ws *Writers) snapshot(count, size uint64, success bool) {
	ws.Lock()
	defer ws.Unlock()
	ws.TotalCount += count
	ws.QueueSize = size
	if !success {
		ws.FailCount++
		ws.FailTotal += count
	}
}

func (ws *Writers) QueueMetrics() *Snapshot {
	ws.Lock()
	defer ws.Unlock()
	ss := ws.Snapshot
	return &ss
}

// WriteTimeSeries write prompb.TimeSeries to all writers
func (ws *Writers) WriteTimeSeries(timeSeries []prompb.TimeSeries) {
	if len(timeSeries) == 0 {
		return
	}

	wg := sync.WaitGroup{}
	for key := range ws.writerMap {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			ws.writerMap[key].Write(timeSeries)
		}(key)
	}
	wg.Wait()
}

func printTestMetrics(samples []*types2.Sample) {
	for _, sample := range samples {
		printTestMetric(sample)
	}
}

// printTestMetric print metric to stdout, only used in debug/test mode
func printTestMetric(sample *types2.Sample) {
	var sb strings.Builder

	sb.WriteString(sample.Timestamp.Format("15:04:05"))
	sb.WriteString(" ")
	sb.WriteString(sample.Metric)

	arr := make([]string, 0, len(sample.Labels))
	for key, val := range sample.Labels {
		arr = append(arr, fmt.Sprintf("%s=%v", key, val))
	}

	sort.Strings(arr)

	for _, pair := range arr {
		sb.WriteString(" ")
		sb.WriteString(pair)
	}

	sb.WriteString(" ")
	sb.WriteString(fmt.Sprint(sample.Value))
}
