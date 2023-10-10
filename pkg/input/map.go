package input

type ConfigMap map[string]string

func (c ConfigMap) Get(key string) string {
	return c[key]
}
