package input

import (
	"github.com/noovertime7/kubemonitor/pkg/conv"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

type ConfigMap map[string]string

func (c ConfigMap) Get(key string) string {
	return c[key]
}

func (c ConfigMap) ParseSlice(key string) []string {
	return strings.Split(c.Get(key), ",")
}

func (c ConfigMap) ParseBool(key string) bool {
	val := c.Get(key)
	if val == "" {
		return false
	}

	b, err := strconv.ParseBool(val)
	if err != nil {
		logrus.Errorf("config parse bool error %s:%v ", key, err)
		return false
	}
	return b
}

func (c ConfigMap) ParseInt(key string) int {
	i, _ := conv.ToInt(c.Get(key))
	return i
}
