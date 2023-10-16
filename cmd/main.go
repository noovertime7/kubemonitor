/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"github.com/noovertime7/kubemonitor/pkg/metrics"
	monitorRuntime "github.com/noovertime7/kubemonitor/runtime"

	"github.com/noovertime7/kubemonitor/internal/writer"
	"github.com/noovertime7/kubemonitor/pkg/input"
	"github.com/noovertime7/kubemonitor/pkg/worker"

	"os"

	nativeZap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	kubemonitoriov1 "github.com/noovertime7/kubemonitor/api/v1"
	"github.com/noovertime7/kubemonitor/internal/controller"

	_ "github.com/noovertime7/kubemonitor/internal/handlers/clickhouse"
	_ "github.com/noovertime7/kubemonitor/internal/handlers/elasticsearch"
	_ "github.com/noovertime7/kubemonitor/internal/handlers/mysql"
	_ "github.com/noovertime7/kubemonitor/internal/handlers/postgresql"
	_ "github.com/noovertime7/kubemonitor/internal/handlers/redis"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(kubemonitoriov1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var (
		logLevel             string
		probeAddr            string
		enableLeaderElection bool
		metricsAddr          string
		maxWriterQueueSize   int
		writerBatch          int
		region               string
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&logLevel, "log-level", "info", "log level")
	flag.IntVar(&maxWriterQueueSize, "max-writer-queue-size", 1000000, "max-writer-queue-size")
	flag.IntVar(&writerBatch, "writer-batch", 1000, "writer-batch")
	flag.StringVar(&region, "region", "local", "monitor region")

	opts := zap.Options{
		Development: true,
		ZapOpts: []nativeZap.Option{
			nativeZap.AddCaller(),
			nativeZap.AddCallerSkip(1),
		},
		//Encoder:     getEncoder(),
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	logger := zap.New(zap.UseFlagOptions(&opts), zap.JSONEncoder(func(encoderConfig *zapcore.EncoderConfig) {
		encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}), zap.Level(SetLevel(logLevel)))

	ctrl.SetLogger(logger)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "eb138a99.kubemonitor.io",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	writersMgr := writer.NewWriter(maxWriterQueueSize, writerBatch, logger)
	wker := worker.NewWorker()

	// 启动kubeMetrics
	metrics.NewKubeMonitor(writersMgr, logger, region).Run(monitorRuntime.SystemContext.Done())

	if err = controller.NewPrometheusPushReconciler(mgr.GetClient(), mgr.GetScheme(), writersMgr).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PrometheusPush")
		os.Exit(1)
	}

	if err = controller.NewMonitorReconciler(mgr.GetClient(), mgr.GetScheme(), writersMgr, wker, input.Factory).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Monitor")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(monitorRuntime.SystemContext); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}

	wker.StopAll()
}

func SetLevel(level string) zapcore.LevelEnabler {
	atomicLevel := nativeZap.NewAtomicLevel()
	_ = atomicLevel.UnmarshalText([]byte(level))
	return atomicLevel
}
