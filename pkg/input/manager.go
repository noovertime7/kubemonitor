package input

import (
	"fmt"
	"github.com/noovertime7/kubemonitor/pkg/types"
	"sync"
)

var Factory = NewHandlerFactory()

type SharedHandlerFactory struct {
	RootList     map[string]*types.SampleList
	lock         *sync.Mutex
	handlers     []HandlerFactory
	supportModel map[string]HandlerFactory
}

func NewHandlerFactory() *SharedHandlerFactory {
	return &SharedHandlerFactory{
		RootList:     make(map[string]*types.SampleList),
		handlers:     []HandlerFactory{},
		supportModel: make(map[string]HandlerFactory),
		lock:         &sync.Mutex{},
	}
}

func (m *SharedHandlerFactory) Gather(model string) error {
	handler, ok := m.supportModel[model]
	if !ok {
		return fmt.Errorf("%s not supported", model)
	}
	list, ok := m.RootList[model]
	if !ok {
		return fmt.Errorf("%s data list not found", model)
	}
	if err := handler.Gather(list); err != nil {
		return err
	}
	return nil
}

func (m *SharedHandlerFactory) GetHandler(model string) (HandlerFactory, error) {
	handler, ok := m.supportModel[model]
	if !ok {
		return nil, fmt.Errorf("%s not register", model)
	}
	return handler, nil
}

func (m *SharedHandlerFactory) InitConfig(model string, cfg map[string]string) error {
	handler, ok := m.supportModel[model]
	if !ok {
		return fmt.Errorf("%s not register", model)
	}
	return handler.Init(cfg)
}

func (m *SharedHandlerFactory) InitConfigWithGather(model string, cfg map[string]string) error {
	err := m.InitConfig(model, cfg)
	if err != nil {
		return err
	}
	return m.Gather(model)
}

func (m *SharedHandlerFactory) RegisterHandler(handler HandlerFactory) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.handlers = append(m.handlers, handler)
	m.supportModel[handler.Name()] = handler
	m.RootList[handler.Name()] = types.NewSampleList()
}

func (m *SharedHandlerFactory) PopBackAll(model string) ([]*types.Sample, error) {
	list, ok := m.RootList[model]
	if !ok {
		return nil, fmt.Errorf("%s data list not found", model)
	}
	return list.PopBackAll(), nil
}

func (m *SharedHandlerFactory) List(model string) (*types.SampleList, error) {
	list, ok := m.RootList[model]
	if !ok {
		return nil, fmt.Errorf("%s data list not found", model)
	}
	return list, nil
}
