package worker

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"sync"
	"time"
)

type Worker interface {
	AddWorkerTask(name string)
	Run(name string, period time.Duration, f func()) error
	Exist(name string) bool
	Stop(name string)
	Range()
	StopAll()
}
type workers struct {
	Tasks sync.Map
}

type workerTask struct {
	stopCh chan struct{}
}

func NewWorker() Worker {
	return &workers{
		Tasks: sync.Map{},
	}
}

func (w *workers) Exist(name string) bool {
	_, ok := w.Tasks.Load(name)
	return ok
}

func (w *workers) AddWorkerTask(name string) {
	_, ok := w.Tasks.Load(name)
	if !ok {
		task := &workerTask{
			stopCh: make(chan struct{}),
		}
		w.Tasks.Store(name, task)
	}
}

func (w *workers) Run(name string, period time.Duration, f func()) error {
	task, ok := w.Tasks.Load(name)
	if !ok {
		return fmt.Errorf("%s not registered", name)
	}

	taskObj := task.(*workerTask)
	go wait.Until(f, period, taskObj.stopCh)
	return nil
}

func (w *workers) Stop(name string) {
	task, ok := w.Tasks.Load(name)
	if !ok {
		logrus.Errorf("stop failed,not load %s", name)
		return
	}

	taskObj := task.(*workerTask)
	close(taskObj.stopCh)
	w.Tasks.Delete(name)

	logrus.Info(name, " stop")
}

func (w *workers) Range() {
	w.Tasks.Range(walk)
}

func walk(key, value interface{}) bool {
	logrus.Info("worker: ", key)
	return true
}

func (w *workers) StopAll() {
	w.Tasks.Range(func(key, value interface{}) bool {
		taskObj := value.(*workerTask)
		close(taskObj.stopCh)
		w.Tasks.Delete(key)
		return true
	})
}
