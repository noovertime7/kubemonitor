package controller

import (
	"github.com/noovertime7/kubemonitor/internal/labels"
	"github.com/noovertime7/kubemonitor/pkg/types"
	"time"
)

func Process(slist *types.SampleList, additionalLabels map[string]string) *types.SampleList {
	nlst := types.NewSampleList()
	if slist.Len() == 0 {
		return nlst
	}

	now := time.Now()
	ss := slist.PopBackAll()

	for i := range ss {
		if ss[i] == nil {
			continue
		}
		//
		//// drop metrics
		//if ic.MetricsDropFilter != nil {
		//	if ic.MetricsDropFilter.Match(ss[i].Metric) {
		//		continue
		//	}
		//}
		//
		//// pass metrics
		//if ic.MetricsPassFilter != nil {
		//	if !ic.MetricsPassFilter.Match(ss[i].Metric) {
		//		continue
		//	}
		//}
		//
		//// mapping values
		//for j := 0; j < len(ic.ProcessorEnum); j++ {
		//	if ic.ProcessorEnum[j].MetricsFilter.Match(ss[i].Metric) {
		//		v, has := ic.ProcessorEnum[j].ValueMappings[fmt.Sprint(ss[i].Value)]
		//		if has {
		//			ss[i].Value = v
		//		}
		//	}
		//}

		if ss[i].Timestamp.IsZero() {
			ss[i].Timestamp = now
		}

		//// name prefix
		//if len(ic.MetricsNamePrefix) > 0 {
		//	ss[i].Metric = ic.MetricsNamePrefix + ss[i].Metric
		//}
		//
		//// add instance labels
		//labels := ic.GetLabels()
		//for k, v := range labels {
		//	if v == "-" {
		//		delete(ss[i].Labels, k)
		//		continue
		//	}
		//	ss[i].Labels[k] = Expand(v)
		//}
		//
		// add global labels
		for k, v := range labels.GlobalLabels() {
			if _, has := ss[i].Labels[k]; !has {
				ss[i].Labels[k] = v
			}
		}

		// add additional Labels
		for k, v := range additionalLabels {
			if _, has := ss[i].Labels[k]; !has {
				ss[i].Labels[k] = v
			}
		}

		ss[i].Labels["instance"] = ss[i].Labels["address"]

		//
		//// add label: agent_hostname
		//if _, has := ss[i].Labels[agentHostnameLabelKey]; !has {
		//	if !Config.Global.OmitHostname {
		//		ss[i].Labels[agentHostnameLabelKey] = Config.GetHostname()
		//	}
		//}

		nlst.PushFront(ss[i])
	}

	return nlst
}
