package runtime

import "context"

func init() {
	SetupContext(SetupSignalHandler())
}

var (
	SystemContext context.Context
)

func SetupContext(parentCh <-chan struct{}) {
	if SystemContext == nil {
		SystemContext, _ = contextForChannel(parentCh)
	}
}

func contextForChannel(parentCh <-chan struct{}) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-parentCh:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}
