package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
	"github.com/natefinch/lumberjack"
	"github.com/oklog/run"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"

	"soldr/internal/app"
	"soldr/internal/app/api/server"
	srvevents "soldr/internal/app/api/server/events"
	"soldr/internal/app/api/storage/mem"
	"soldr/internal/app/api/utils/meter"
	"soldr/internal/app/api/worker"
	"soldr/internal/log"
	"soldr/internal/observability"
	"soldr/internal/secret"
	"soldr/internal/storage/mysql"
	"soldr/internal/system"
	"soldr/internal/version"
)

type Config struct {
	Debug            bool `config:"debug"`
	Develop          bool `config:"is_develop"`
	Log              LogConfig
	DB               DBConfig
	Tracing          TracingConfig
	PublicAPI        PublicAPIConfig
	EventWorker      EventWorkerConfig
	UserActionWorker UserActionWorkerConfig
}

type LogConfig struct {
	Level  string `config:"log_level"`
	Format string `config:"log_format"`
}

type DBConfig struct {
	User string `config:"db_user,required"`
	Pass string `config:"db_pass,required"`
	Name string `config:"db_name,required"`
	Host string `config:"db_host,required"`
	Port int    `config:"db_port,required"`
}

// TODO: refactor old env names
type PublicAPIConfig struct {
	Addr            string        `config:"api_listen_http"`
	AddrHTTPS       string        `config:"api_listen_https"`
	UseSSL          bool          `config:"api_use_ssl"`
	CertFile        string        `config:"api_ssl_crt"`
	KeyFile         string        `config:"api_ssl_key"`
	GracefulTimeout time.Duration `config:"public_api_graceful_timeout"`
}

type TracingConfig struct {
	Addr string `config:"otel_addr"`
}

type EventWorkerConfig struct {
	PollInterval time.Duration `config:"event_worker_poll_interval"`
}

type UserActionWorkerConfig struct {
	MaxMessages uint `config:"user_action_worker_max_messages"`
}

func defaultConfig() Config {
	return Config{
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
		Tracing: TracingConfig{
			Addr: "otel.local:8148",
		},
		PublicAPI: PublicAPIConfig{
			Addr:      ":8080",
			AddrHTTPS: ":8443",
		},
		EventWorker: EventWorkerConfig{
			PollInterval: 30 * time.Second,
		},
		UserActionWorker: UserActionWorkerConfig{
			MaxMessages: 100,
		},
	}
}

func main() {
	var printVersion bool
	flag.BoolVar(&printVersion, "version", false, "Print current version and exit")
	flag.Parse()

	if printVersion {
		fmt.Printf("version is %s\n", version.GetBinaryVersion())
		os.Exit(0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := defaultConfig()
	cfgLoader := confita.NewLoader(
		env.NewBackend(),
	)
	if err := cfgLoader.Load(ctx, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "could not load configuration: %s", err)
		return
	}

	if cfg.Develop {
		version.IsDevelop = "true"
	}

	if cfg.Debug {
		cfg.Log.Level = "debug"
	}
	logLevel, err := log.ParseLevel(cfg.Log.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not parse log level: %s", err)
		return
	}
	logFormat, err := log.ParseFormat(cfg.Log.Format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not parse log format: %s", err)
		return
	}
	logDir := "logs"
	if dir, ok := os.LookupEnv("LOG_DIR"); ok {
		logDir = dir
	}
	logFile := &lumberjack.Logger{
		Filename:   path.Join(logDir, "app.log"),
		MaxSize:    100,
		MaxBackups: 7,
		MaxAge:     14,
		Compress:   true,
	}
	logger := log.New(log.Config{Level: logLevel, Format: logFormat}, io.MultiWriter(os.Stdout, logFile))
	ctx = log.AttachToContext(ctx, logger)

	dsn := fmt.Sprintf("%s:%s@%s/%s?parseTime=true",
		cfg.DB.User,
		cfg.DB.Pass,
		fmt.Sprintf("tcp(%s:%d)", cfg.DB.Host, cfg.DB.Port),
		cfg.DB.Name,
	)
	db, err := mysql.New(&mysql.Config{DSN: secret.NewString(dsn)})
	if err != nil {
		logger.WithError(err).Error("could not apply migrations")
		return
	}
	if err = db.RetryConnect(ctx, 10, 100*time.Millisecond); err != nil {
		logger.WithError(err).Error("could not connect to database")
		return
	}

	migrationDir := "db/api/migrations"
	if dir, ok := os.LookupEnv("MIGRATION_DIR"); ok {
		migrationDir = dir
	}
	if err = db.Migrate(migrationDir); err != nil {
		logger.WithError(err).Error("could not apply migrations")
		return
	}
	dbWithORM, err := db.WithORM(ctx)
	if err != nil {
		logger.WithError(err).Error("could not create ORM")
		return
	}
	if cfg.Debug {
		dbWithORM.LogMode(true)
	}

	// storages
	serviceDBConnectionStorage := mem.NewServiceDBConnectionStorage()
	serviceS3ConnectionStorage := mem.NewServiceS3ConnectionStorage()

	tracerClient := observability.NewProxyTracerClient(
		observability.NewOtlpTracerClient(cfg.Tracing.Addr),
		observability.NewHookTracerClient(&observability.HookClientConfig{
			ResendTimeout:   observability.DefaultResendTimeout,
			QueueSizeLimit:  observability.DefaultQueueSizeLimit,
			PacketSizeLimit: observability.DefaultPacketSizeLimit,
		}),
	)
	attr := attribute.String("api_server_id", system.MakeAgentID())
	tracerProvider, err := observability.NewTracerProvider(ctx, tracerClient, "vxapi", version.GetBinaryVersion(), attr)
	if err != nil {
		logger.WithError(err).Error("could not create tracer provider")
		return
	}
	meterClient := observability.NewProxyMeterClient(
		observability.NewOtlpMeterClient(cfg.Tracing.Addr),
		observability.NewHookMeterClient(&observability.HookClientConfig{
			ResendTimeout:   observability.DefaultResendTimeout,
			QueueSizeLimit:  observability.DefaultQueueSizeLimit,
			PacketSizeLimit: observability.DefaultPacketSizeLimit,
		}),
	)
	if err != nil {
		logger.WithError(err).Error("could not create meter client")
		return
	}
	meterProvider, err := observability.NewMeterProvider(ctx, meterClient, "vxapi", version.GetBinaryVersion(), attr)
	if err != nil {
		logger.WithError(err).Error("could not create meter provider")
		return
	}

	logLevels := []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	}
	if cfg.Debug {
		logLevels = append(logLevels, logrus.DebugLevel)
	}
	observability.InitObserver(ctx, tracerProvider, meterProvider, tracerClient, meterClient, "vxapi", logLevels)

	gormMeter := meterProvider.Meter("vxapi-meter")
	if err = meter.InitGormMetrics(gormMeter); err != nil {
		logger.WithError(err).Error("could not initialize vxapi-meter")
		return
	}

	// initialize system metric collection in current observer instance
	observability.Observer.StartProcessMetricCollect("vxapi", version.GetBinaryVersion(), attr)
	observability.Observer.StartGoRuntimeMetricCollect("vxapi", version.GetBinaryVersion(), attr)

	exchanger := srvevents.NewExchanger()
	eventWorker := srvevents.NewEventPoller(exchanger, cfg.EventWorker.PollInterval, dbWithORM)
	go func() {
		if err = eventWorker.Run(ctx); err != nil {
			logger.WithError(err).Error("could not start event worker")
		}
	}()

	// run worker to synchronize all global binaries list to all instance DB
	go worker.SyncBinariesAndExtConns(ctx, dbWithORM)

	// run worker to synchronize all global released modules list to all instance DB
	go worker.SyncModulesToPolicies(ctx, dbWithORM)

	// run worker to synchronize events retention policy to all instance DB
	go worker.SyncRetentionEvents(ctx, dbWithORM)

	runGroup := app.NewAppGroup()
	signal.Ignore(syscall.SIGHUP, syscall.SIGPIPE)
	runGroup.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))

	router := server.NewRouter(
		dbWithORM,
		exchanger,
		serviceDBConnectionStorage,
		serviceS3ConnectionStorage,
	)
	api := server.NewAPI(server.Config{
		Addr:      cfg.PublicAPI.Addr,
		AddrHTTPS: cfg.PublicAPI.AddrHTTPS,
		UseSSL:    cfg.PublicAPI.UseSSL,
		CertFile:  cfg.PublicAPI.CertFile,
		KeyFile:   cfg.PublicAPI.KeyFile,
	}, router, logger.WithField("component", "api"))
	runGroup.Add(api.Start, api.Stop)

	defer observability.Observer.Close()

	logger.Info("starting services")
	runGroup.Run()
}
