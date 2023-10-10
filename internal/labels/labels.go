package labels

func Labels(labels map[string]string) map[string]string {
	ret := make(map[string]string)
	for k, v := range labels {
		ret[k] = v
	}
	return ret
}

func GlobalLabels() map[string]string {
	return map[string]string{
		"source": "kubemonitor",
	}
}
