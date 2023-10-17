package metrics

import (
	"github.com/go-logr/logr"
	"github.com/noovertime7/kubemonitor/internal/writer"
	"github.com/noovertime7/kubemonitor/pkg/process"
	"github.com/noovertime7/kubemonitor/pkg/types"
	"time"
)

const (
	defaultPrefix  = "kubemonitor"
	metricInterval = 15 * time.Second
)

const (
	metricPrefix           = defaultPrefix
	metricEnqueueSum       = "metrics_enqueue_sum"
	metricEnqueueFailedSum = "metrics_enqueue_failed_sum"
	metricEnqueueFailedCnt = "metrics_enqueue_failed_count"
	metricQueueSize        = "current_queue_size"
	metricsWriteTotal      = "write_total"
	metricsWriteFailTotal  = "write_fail_total"
)

type KubeMonitor struct {
	w      writer.WritersManager
	logger logr.Logger
}

func NewKubeMonitor(w writer.WritersManager, logger logr.Logger) *KubeMonitor {
	k := &KubeMonitor{w: w, logger: logger}
	return k
}

func (k *KubeMonitor) Run(stopCh <-chan struct{}) {
	sList := types.NewSampleList()

	go func() {
		for {
			select {
			case <-time.After(metricInterval):
				k.collectMetrics(sList)
				processList := process.Process(sList, map[string]string{"source": "kubemonitor"})
				arr := processList.PopBackAll()
				k.w.WriteSamples(arr)
				k.logger.Info("kubeMonitor write samples success")
			case <-stopCh:
				k.logger.Info("kubemonitor metrics stop...")
				return
			}
		}
	}()
}

func (k *KubeMonitor) collectMetrics(sList *types.SampleList) {
	ss := k.w.QueueMetrics()

	sList.PushSample(metricPrefix, metricEnqueueSum, ss.QueueTotalCount)
	sList.PushSample(metricPrefix, metricEnqueueFailedSum, ss.QueueFailTotal)
	sList.PushSample(metricPrefix, metricEnqueueFailedCnt, ss.QueueFailCount)
	sList.PushSample(metricPrefix, metricQueueSize, ss.QueueSize)
	sList.PushSample(metricPrefix, metricsWriteTotal, ss.WriteTotalCount)
	sList.PushSample(metricPrefix, metricsWriteFailTotal, ss.WriteFailTotal)
}
