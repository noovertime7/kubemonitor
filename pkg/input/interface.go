package input

import "github.com/noovertime7/kubemonitor/pkg/types"

type HandlerFactory interface {
	Name() string
	Init(config map[string]string) error
	Gather(slist *types.SampleList) error
}
