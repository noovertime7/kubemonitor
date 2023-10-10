package worker

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"sync"
	"time"
)

type Worker interface {
	AddWorkerTask(name string)
	Run(name string, period time.Duration, f func()) error
	Stop(name string)
}

func NewWorker() Worker {
	return &worker{
		Tasks: map[string]chan struct{}{},
		lock:  &sync.Mutex{},
	}
}

type worker struct {
	Tasks map[string]chan struct{}
	lock  *sync.Mutex
}

func (m *worker) AddWorkerTask(name string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.Tasks[name] = make(chan struct{})
}

func (m *worker) Run(name string, period time.Duration, f func()) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	stopCh, has := m.Tasks[name]
	if !has {
		return fmt.Errorf("%s not register", name)
	}
	go wait.Until(f, period, stopCh)
	return nil
}

func (m *worker) Stop(name string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.Tasks[name] <- struct{}{}
	delete(m.Tasks, name)
}
