package server

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/getsentry/sentry-go"
	"github.com/langgenius/dify-plugin-daemon/internal/cluster"
	"github.com/langgenius/dify-plugin-daemon/internal/core/persistence"
	"github.com/langgenius/dify-plugin-daemon/internal/core/plugin_manager"
	"github.com/langgenius/dify-plugin-daemon/internal/db"
	"github.com/langgenius/dify-plugin-daemon/internal/oss"
	"github.com/langgenius/dify-plugin-daemon/internal/oss/local"
	"github.com/langgenius/dify-plugin-daemon/internal/oss/s3"
	"github.com/langgenius/dify-plugin-daemon/internal/oss/tencent_cos"
	"github.com/langgenius/dify-plugin-daemon/internal/types/app"
	"github.com/langgenius/dify-plugin-daemon/internal/utils/log"
	"github.com/langgenius/dify-plugin-daemon/internal/utils/routine"
)

func initOSS(config *app.Config) oss.OSS {
	// init storage
	var storage oss.OSS
	var err error
	switch config.PluginStorageType {
	case oss.OSS_TYPE_S3:
		storage, err = s3.NewS3Storage(
			config.S3UseAwsManagedIam,
			config.S3Endpoint,
			config.S3UsePathStyle,
			config.AWSAccessKey,
			config.AWSSecretKey,
			config.PluginStorageOSSBucket,
			config.AWSRegion,
		)
	case oss.OSS_TYPE_LOCAL:
		storage = local.NewLocalStorage(config.PluginStorageLocalRoot)
	case oss.OSS_TYPE_TENCENT_COS:
		storage, err = tencent_cos.NewTencentCOSStorage(
			config.TencentCOSSecretId,
			config.TencentCOSSecretKey,
			config.TencentCOSRegion,
			config.PluginStorageOSSBucket,
		)
	default:
		log.Panic("Invalid plugin storage type: %s", config.PluginStorageType)
	}

	if err != nil {
		log.Panic("Failed to create storage: %s", err)
	}

	return storage
}

func waitForSignal(app *App) os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Listening signal for gracefully shutdown")

	sig := <-sigChan
	log.Info("Received signal: %s", sig)
	app.cluster.Close()

	<-app.cluster.NotifyClusterStopped()
	log.Info("Cluster stopped.")

	return sig
}

func (app *App) Run(config *app.Config) {
	// init routine pool
	if config.SentryEnabled {
		routine.InitPool(config.RoutinePoolSize, sentry.ClientOptions{
			Dsn:              config.SentryDSN,
			AttachStacktrace: config.SentryAttachStacktrace,
			TracesSampleRate: config.SentryTracesSampleRate,
			SampleRate:       config.SentrySampleRate,
			EnableTracing:    config.SentryTracingEnabled,
		})
	} else {
		routine.InitPool(config.RoutinePoolSize)
	}

	// init db
	db.Init(config)

	// init oss
	oss := initOSS(config)

	// create manager
	manager := plugin_manager.InitGlobalManager(oss, config)

	// create cluster
	app.cluster = cluster.NewCluster(config, manager)

	// register plugin lifetime event
	manager.AddPluginRegisterHandler(app.cluster.RegisterPlugin)

	// init manager
	manager.Launch(config)

	// init persistence
	persistence.InitPersistence(oss, config)

	// launch cluster
	app.cluster.Launch()

	// start http server
	app.server(config)

	// block
	// select {}
	waitForSignal(app)
}
