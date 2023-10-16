package controller

import (
	"context"
	kubemonitoriov1 "github.com/noovertime7/kubemonitor/api/v1"
	"github.com/noovertime7/kubemonitor/pkg/worker"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type monitorWorker struct {
	client  client.Client
	monitor *kubemonitoriov1.Monitor
	worker  worker.Worker
}

func NewMonitorWorker(c client.Client, m *kubemonitoriov1.Monitor, w worker.Worker) *monitorWorker {
	return &monitorWorker{
		client:  c,
		monitor: m,
		worker:  w,
	}
}

func (m *monitorWorker) RunAfterPatchStatus(ctx context.Context, name string, period time.Duration, f func() error) error {
	if err := m.worker.Run(name, period, func() {
		err := f()
		if err != nil {
			logrus.WithFields(map[string]interface{}{
				"name": m.monitor.Name,
			}).Errorf("work error: %v", err)
		}

		err = m.UpdateStatus(ctx, time.Now())
		if err != nil {
			logrus.WithFields(map[string]interface{}{
				"name": m.monitor.Name,
			}).Errorf("update status error: %v", err)
		}
	}); err != nil {
		return err
	}
	return nil
}

func (m *monitorWorker) UpdateStatus(ctx context.Context, pushTime time.Time) error {
	monitor := &kubemonitoriov1.Monitor{}
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := m.client.Get(ctx, client.ObjectKeyFromObject(m.monitor), monitor); err != nil {
			return err
		}
		monitor.Status = kubemonitoriov1.MonitorStatus{LastPush: metav1.Time{Time: pushTime}}
		return m.client.Status().Update(ctx, monitor)
	})
}

func (m *monitorWorker) AddWorkerTask(name string) {
	m.worker.AddWorkerTask(name)
}

func (m *monitorWorker) StopWithRange(name string) {
	m.worker.Stop(name)
	m.Range()
}

func (m *monitorWorker) Range() {
	m.worker.Range()
}
