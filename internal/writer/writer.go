package writer

import (
	"bytes"
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/prometheus/prompb"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"time"
)

type Writer struct {
	Opts   WriterOption
	Client api.Client
}

type WriterOption struct {
	Url           string   `toml:"url"`
	BasicAuthUser string   `toml:"basic_auth_user"`
	BasicAuthPass string   `toml:"basic_auth_pass"`
	Headers       []string `toml:"headers"`

	Timeout             int64 `toml:"timeout"`
	DialTimeout         int64 `toml:"dial_timeout"`
	MaxIdleConnsPerHost int   `toml:"max_idle_conns_per_host"`
}

// newWriter creates a new Writer from config.WriterOption
func newWriter(opt WriterOption) (Writer, error) {
	cli, err := api.NewClient(api.Config{
		Address: opt.Url,
		RoundTripper: &http.Transport{
			// TLSClientConfig: tlsConfig,
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout: time.Duration(opt.DialTimeout) * time.Second,
			}).DialContext,
			ResponseHeaderTimeout: time.Duration(opt.Timeout) * time.Second,
			MaxIdleConnsPerHost:   opt.MaxIdleConnsPerHost,
			MaxIdleConns:          100,
		},
	})

	if err != nil {
		return Writer{}, err
	}

	return Writer{
		Opts:   opt,
		Client: cli,
	}, nil
}

func (w Writer) Write(items []prompb.TimeSeries) error {
	if len(items) == 0 {
		return nil
	}

	req := &prompb.WriteRequest{
		Timeseries: items,
	}

	data, err := proto.Marshal(req)
	if err != nil {
		logrus.Error("W! marshal prom data to proto got error:", err, "data:", items)
		return err
	}

	if err := w.post(snappy.Encode(nil, data)); err != nil {
		logrus.Error("W! post to", w.Opts.Url, "got error:", err)
		logrus.Error("W! example timeseries:", items[0].String())
	}
	return nil
}

func (w Writer) post(req []byte) error {
	httpReq, err := http.NewRequest("POST", w.Opts.Url, bytes.NewReader(req))
	if err != nil {
		logrus.Error("W! create remote write request got error:", err)
		return err
	}

	httpReq.Header.Add("Content-Encoding", "snappy")
	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("User-Agent", "categraf")
	httpReq.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	for i := 0; i < len(w.Opts.Headers); i += 2 {
		httpReq.Header.Add(w.Opts.Headers[i], w.Opts.Headers[i+1])
		if w.Opts.Headers[i] == "Host" {
			httpReq.Host = w.Opts.Headers[i+1]
		}
	}

	if w.Opts.BasicAuthUser != "" {
		httpReq.SetBasicAuth(w.Opts.BasicAuthUser, w.Opts.BasicAuthPass)
	}

	resp, body, err := w.Client.Do(context.Background(), httpReq)
	if err != nil {
		logrus.Error("W! push data with remote write request got error:", err, "response body:", string(body))
		return err
	}

	if resp.StatusCode >= 400 {
		err = fmt.Errorf("push data with remote write request got status code: %v, response body: %s", resp.StatusCode, string(body))
		return err
	}

	return nil
}
