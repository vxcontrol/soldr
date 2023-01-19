package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"

	"soldr/pkg/app/api/server"
	"soldr/pkg/app/api/storage/mem"
	useraction "soldr/pkg/app/api/user_action"
	"soldr/pkg/app/api/worker"
	"soldr/pkg/app/api/worker/events"
	"soldr/pkg/log"
	"soldr/pkg/observability"
	"soldr/pkg/secret"
	"soldr/pkg/storage"
	"soldr/pkg/storage/mysql"
	"soldr/pkg/system"
	"soldr/pkg/version"
)

const serviceName = "vxapi"

type Config struct {
	Debug             bool `config:"debug"`
	Develop           bool `config:"is_develop"`
	Log               LogConfig
	DB                DBConfig
	Tracing           TracingConfig
	PublicAPI         PublicAPIConfig
	EventWorker       EventWorkerConfig
	ServerEventWorker ServerEventWorkerConfig
}

type LogConfig struct {
	Level  string `config:"log_level"`
	Format string `config:"log_format"`
	Dir    string `config:"log_dir"`
}

type DBConfig struct {
	User         string `config:"db_user,required"`
	Pass         string `config:"db_pass,required"`
	Name         string `config:"db_name,required"`
	Host         string `config:"db_host,required"`
	Port         int    `config:"db_port,required"`
	MigrationDir string `config:"migration_dir"`
}

type PublicAPIConfig struct {
	Addr            string        `config:"api_listen_http"`
	AddrHTTPS       string        `config:"api_listen_https"`
	UseSSL          bool          `config:"api_use_ssl"`
	CertFile        string        `config:"api_ssl_crt"`
	KeyFile         string        `config:"api_ssl_key"`
	GracefulTimeout time.Duration `config:"public_api_graceful_timeout"`
	StaticPath      string        `config:"api_static_path"`
	StaticURL       string        `config:"api_static_url"`
	TemplatesDir    string        `config:"templates_dir"`
	CertsPath       string        `config:"certs_path"`
}

type TracingConfig struct {
	Addr string `config:"otel_addr"`
}

type EventWorkerConfig struct {
	PollInterval time.Duration `config:"event_worker_poll_interval"`
}

type ServerEventWorkerConfig struct {
	KeepDays int `config:"retention_events"`
}

func defaultConfig() Config {
	return Config{
		Log: LogConfig{
			Level:  "info",
			Format: "json",
			Dir:    "logs",
		},
		Tracing: TracingConfig{
			Addr: "otel.local:8148",
		},
		DB: DBConfig{
			MigrationDir: "db/api/migrations",
		},
		PublicAPI: PublicAPIConfig{
			Addr:            ":8080",
			AddrHTTPS:       ":8443",
			GracefulTimeout: time.Minute,
			TemplatesDir:    "templates",
			StaticPath:      "static",
			CertsPath:       filepath.Join("security", "certs", "api"),
		},
		EventWorker: EventWorkerConfig{
			PollInterval: 30 * time.Second,
		},
		ServerEventWorker: ServerEventWorkerConfig{
			KeepDays: 7,
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

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

	logLevels := []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	}
	if cfg.Debug {
		logLevels = append(logLevels, logrus.DebugLevel)
		cfg.Log.Level = "debug"
		cfg.Log.Format = "text"
	}
	logFile := &lumberjack.Logger{
		Filename:   path.Join(cfg.Log.Dir, "api.log"),
		MaxSize:    10,
		MaxBackups: 7,
		MaxAge:     14,
		Compress:   true,
	}
	logrus.SetLevel(log.ParseLevel(cfg.Log.Level))
	logrus.SetFormatter(log.ParseFormat(cfg.Log.Format))
	logrus.SetOutput(io.MultiWriter(os.Stdout, logFile))

	dsn := fmt.Sprintf("%s:%s@%s/%s?parseTime=true",
		cfg.DB.User,
		cfg.DB.Pass,
		fmt.Sprintf("tcp(%s:%d)", cfg.DB.Host, cfg.DB.Port),
		cfg.DB.Name,
	)
	db, err := mysql.New(&mysql.Config{DSN: secret.NewString(dsn)})
	if err != nil {
		logrus.WithError(err).Error("could not create DB instance")
		return
	}
	if err = db.RetryConnect(ctx, 10, 100*time.Millisecond); err != nil {
		logrus.WithError(err).Error("could not connect to database")
		return
	}
	if err = db.Migrate(cfg.DB.MigrationDir); err != nil {
		logrus.WithError(err).Error("could not apply migrations")
		return
	}
	dbWithORM, err := db.WithORM()
	if err != nil {
		logrus.WithError(err).Error("could not create ORM")
		return
	}
	if cfg.Debug {
		dbWithORM.LogMode(true)
	}

	// storages
	dbConnectionStorage := mem.NewDBConnectionStorage()
	s3ConnectionStorage := mem.NewS3ConnectionStorage()

	tracerClient := observability.NewProxyTracerClient(
		observability.NewOtlpTracerAndLoggerClient(cfg.Tracing.Addr),
		observability.NewHookTracerClient(&observability.HookClientConfig{
			ResendTimeout:   observability.DefaultResendTimeout,
			QueueSizeLimit:  observability.DefaultQueueSizeLimit,
			PacketSizeLimit: observability.DefaultPacketSizeLimit,
		}),
	)
	attr := attribute.String("api_server_id", system.MakeAgentID())
	tracerProvider, err := observability.NewTracerProvider(
		ctx,
		tracerClient,
		serviceName,
		version.GetBinaryVersion(),
		attr,
	)
	if err != nil {
		logrus.WithError(err).Error("could not create tracer provider")
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
		logrus.WithError(err).Error("could not create meter client")
		return
	}
	meterProvider, err := observability.NewMeterProvider(
		ctx,
		meterClient,
		serviceName,
		version.GetBinaryVersion(),
		attr,
	)
	if err != nil {
		logrus.WithError(err).Error("could not create meter provider")
		return
	}

	observability.InitObserver(
		ctx,
		tracerProvider,
		meterProvider,
		tracerClient,
		meterClient,
		serviceName,
		version.GetBinaryVersion(),
		logLevels,
	)

	gormMeter := meterProvider.Meter("vxapi-meter")
	if err = storage.InitGormMetrics(gormMeter); err != nil {
		logrus.WithError(err).Error("could not initialize vxapi-meter")
		return
	}

	// initialize system metric collection in current observer instance
	observability.Observer.StartProcessMetricCollect(serviceName, version.GetBinaryVersion(), attr)
	observability.Observer.StartGoRuntimeMetricCollect(serviceName, version.GetBinaryVersion(), attr)
	defer observability.Observer.Close()

	exchanger := events.NewExchanger()
	eventWorker := events.NewEventPoller(exchanger, cfg.EventWorker.PollInterval, dbWithORM)
	go func() {
		if err = eventWorker.Run(ctx); err != nil {
			logrus.WithError(err).Error("could not start event worker")
		}
	}()

	// run worker to synchronize all global binaries list to all instance DB
	go worker.SyncBinariesAndExtConns(ctx, dbWithORM)

	// run worker to synchronize all global released modules list to all instance DB
	go worker.SyncModulesToPolicies(ctx, dbWithORM)

	// run worker to synchronize events retention policy to all instance DB
	go worker.SyncRetentionEvents(ctx, dbWithORM, cfg.ServerEventWorker.KeepDays)

	uiStaticURL, err := url.Parse(cfg.PublicAPI.StaticURL)
	if err != nil {
		logrus.WithError(err).Error("error on parsing URL to redirect requests to the UI static")
		return
	}
	userActionLogger := useraction.NewLogger()

	router := server.NewRouter(
		server.RouterConfig{
			BaseURL:      "/api/v1",
			Debug:        cfg.Debug,
			UseSSL:       cfg.PublicAPI.UseSSL,
			StaticPath:   cfg.PublicAPI.StaticPath,
			StaticURL:    uiStaticURL,
			TemplatesDir: cfg.PublicAPI.TemplatesDir,
			CertsPath:    cfg.PublicAPI.CertsPath,
		},
		dbWithORM,
		exchanger,
		userActionLogger,
		dbConnectionStorage,
		s3ConnectionStorage,
	)

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return server.Server{
			Addr:            cfg.PublicAPI.Addr,
			GracefulTimeout: cfg.PublicAPI.GracefulTimeout,
		}.ListenAndServe(ctx, router)
	})
	if cfg.PublicAPI.UseSSL {
		group.Go(func() error {
			return server.Server{
				Addr:            cfg.PublicAPI.AddrHTTPS,
				CertFile:        cfg.PublicAPI.CertFile,
				KeyFile:         cfg.PublicAPI.KeyFile,
				GracefulTimeout: cfg.PublicAPI.GracefulTimeout,
			}.ListenAndServeTLS(ctx, router)
		})
	}
	if err = group.Wait(); err != nil {
		logrus.WithError(err).Error("could not start services")
	}
}
