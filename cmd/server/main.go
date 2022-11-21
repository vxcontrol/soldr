package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"

	"github.com/jinzhu/gorm"
	"github.com/judwhite/go-svc"
	"github.com/qor/validations"
	"github.com/sirupsen/logrus"
	"github.com/takama/daemon"
	"gopkg.in/natefinch/lumberjack.v2"

	"soldr/internal/app/server/certs"
	certsConfig "soldr/internal/app/server/certs/config"
	"soldr/internal/app/server/config"
	"soldr/internal/app/server/mmodule"
	"soldr/internal/controller"
	"soldr/internal/db"
	"soldr/internal/observability"
	"soldr/internal/storage"
	"soldr/internal/system"
	"soldr/internal/utils"
	"soldr/internal/vxproto"
)

const (
	serviceName = "vxserver"
	loggerKey   = "component"

	loaderTypeFS = "fs"
	loaderTypeS3 = "s3"
	loaderTypeDB = "db"
)

// PackageVer is semantic version of vxserver
var PackageVer string

// PackageRev is revision of vxserver build
var PackageRev string

// Server implements daemon structure
type Server struct {
	version       string
	config        *config.Config
	module        *mmodule.MainModule
	certProvider  certs.Provider
	wg            sync.WaitGroup
	svc           daemon.Daemon
	tracerClient  otlptrace.Client
	metricsClient otlpmetric.Client
}

// Init for preparing server main module struct
func (s *Server) Init(env svc.Environment) (err error) {
	var (
		gormDB *gorm.DB
		logger = logrus.WithField(loggerKey, "init")
	)
	if s.config.Loader.Config == loaderTypeDB {
		dsn, err := dsnFromConfig(&s.config.DB)
		if err != nil {
			return fmt.Errorf("failed to compose a DSN from config: %w", err)
		}
		gdb, err := db.New(dsn)
		if err != nil {
			return fmt.Errorf("failed to initialize a connection to DB: %w", err)
		}
		migrationDir := "db/server/migrations"
		if dir, ok := os.LookupEnv("MIGRATION_DIR"); ok {
			migrationDir = dir
		}
		err = gdb.MigrateUp(migrationDir)
		if err != nil {
			return fmt.Errorf("failed to migrate current DB: %w", err)
		}
		logger.Info("DB migration was completed")

		gormDB, err = initGorm(dsn, s.config.LogDir)
		if err != nil {
			return fmt.Errorf("gorm connection initialization failed: %w", err)
		}
	}

	var cl controller.IConfigLoader
	switch s.config.Loader.Config {
	case loaderTypeFS:
		cl, err = controller.NewConfigFromFS(s.config.Base)
	case loaderTypeS3:
		var s3ConnParams *storage.S3ConnParams
		s3ConnParams, err = s3ConnParamsFromConfig(&s.config.S3)
		if err != nil {
			err = fmt.Errorf("failed to compose s3 connection params from config: %w", err)
			break
		}
		cl, err = controller.NewConfigFromS3(s3ConnParams)
	case loaderTypeDB:
		var dsn *db.DSN
		dsn, err = dsnFromConfig(&s.config.DB)
		if err != nil {
			err = fmt.Errorf("failed to compose a DSN from config: %w", err)
			break
		}
		cl, err = controller.NewConfigFromDB(dsn)
	default:
		err = fmt.Errorf("unknown configuration loader type")
	}
	if err != nil {
		logger.WithError(err).Error("failed to initialize config loader")
		return
	}
	logger.Info("modules configuration loader was created")

	var fl controller.IFilesLoader
	switch s.config.Loader.Files {
	case loaderTypeFS:
		fl, err = controller.NewFilesFromFS(s.config.Base)
	case loaderTypeS3:
		var s3ConnParams *storage.S3ConnParams
		s3ConnParams, err = s3ConnParamsFromConfig(&s.config.S3)
		if err != nil {
			err = fmt.Errorf("failed to compose s3 connection params from config: %w", err)
			break
		}
		fl, err = controller.NewFilesFromS3(s3ConnParams)
	default:
		err = fmt.Errorf("unknown files loader type")
	}
	if err != nil {
		logger.WithError(err).Error("failed initialize files loader")
		return
	}
	logger.Info("modules files loader was created")

	utils.RemoveUnusedTempDir()
	store, err := storage.NewS3(&storage.S3ConnParams{
		Endpoint:   s.config.S3.Endpoint,
		AccessKey:  s.config.S3.AccessKey,
		SecretKey:  s.config.S3.SecretKey,
		BucketName: s.config.S3.BucketName,
	})
	if err != nil {
		logger.WithError(err).Error("failed to initialize the store connection")
	}

	s.certProvider, err = initCertProvider(&s.config.Certs, store)
	if err != nil {
		logger.WithError(err).Error("failed to initialize a certs provider")
		return err
	}
	if s.module, err = mmodule.New(
		s.config.Listen,
		cl,
		fl,
		gormDB,
		store,
		s.certProvider,
		s.version,
		&s.config.Validator,
		s.tracerClient,
		s.metricsClient,
		logrus.StandardLogger().WithField("module", "main"),
	); err != nil {
		logger.WithError(err).Error("failed to initialize main module")
		return
	}
	logger.Info("main module was created")

	return
}

func dsnFromConfig(c *config.DB) (*db.DSN, error) {
	if c == nil {
		return nil, fmt.Errorf("passed config is nil")
	}
	if len(c.Host) == 0 {
		return nil, fmt.Errorf("host is empty")
	}
	if len(c.Port) == 0 {
		return nil, fmt.Errorf("port is empty")
	}
	if len(c.User) == 0 {
		return nil, fmt.Errorf("user is empty")
	}
	if len(c.Pass) == 0 {
		return nil, fmt.Errorf("password is empty")
	}
	if len(c.Name) == 0 {
		return nil, fmt.Errorf("db name is empty")
	}
	return &db.DSN{
		Host:     c.Host,
		Port:     c.Port,
		User:     c.User,
		Password: c.Pass,
		DBName:   c.Name,
	}, nil
}

func s3ConnParamsFromConfig(c *config.S3) (*storage.S3ConnParams, error) {
	if c == nil {
		return nil, fmt.Errorf("passed config is nil")
	}
	if len(c.Endpoint) == 0 {
		return nil, fmt.Errorf("endpoint is empty")
	}
	if len(c.AccessKey) == 0 {
		return nil, fmt.Errorf("access key is empty")
	}
	if len(c.SecretKey) == 0 {
		return nil, fmt.Errorf("secret key is empty")
	}
	if len(c.BucketName) == 0 {
		return nil, fmt.Errorf("bucket name is empty")
	}
	return &storage.S3ConnParams{
		Endpoint:   c.Endpoint,
		AccessKey:  c.AccessKey,
		SecretKey:  c.SecretKey,
		BucketName: c.BucketName,
	}, nil
}

func initCertProvider(c *config.CertsConfig, s3 storage.IFileReader) (certs.Provider, error) {
	if c == nil {
		return nil, fmt.Errorf("passed config object is nil")
	}
	createFileProvider := func(store storage.IFileReader, base string) (certs.Provider, error) {
		conf := &certsConfig.Config{
			StaticProvider: &certsConfig.StaticProvider{
				Reader:   store,
				CertsDir: base,
			},
		}
		p, err := certs.NewProvider(conf)
		if err != nil {
			return nil, fmt.Errorf("failed to create a certs provider: %w", err)
		}
		return p, nil
	}
	switch c.Type {
	case loaderTypeFS:
		store, err := storage.NewFS()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize a file store: %w", err)
		}
		return createFileProvider(store, c.Base)
	case loaderTypeS3:
		return createFileProvider(s3, c.Base)
	default:
		return nil, fmt.Errorf("store type %s is not available for certificate providers", c.Type)
	}
}

func initGorm(dsn *db.DSN, logDir string) (*gorm.DB, error) {
	addr := fmt.Sprintf(
		"%s:%s@%s/%s?parseTime=true",
		dsn.User,
		dsn.Password,
		fmt.Sprintf(
			"tcp(%s:%s)", dsn.Host, dsn.Port,
		),
		dsn.DBName,
	)
	conn, err := gorm.Open("mysql", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a gorm connection: %w", err)
	}
	logger := logrus.New()
	logger.SetOutput(&lumberjack.Logger{
		Filename:   filepath.Join(logDir, "server-gorm.log"),
		MaxSize:    100,
		MaxBackups: 7,
		MaxAge:     14,
		Compress:   true,
	})
	conn.SetLogger(logger)
	conn.LogMode(true)

	validations.RegisterCallbacks(conn)

	conn.DB().SetMaxIdleConns(10)
	conn.DB().SetMaxOpenConns(50)
	conn.DB().SetConnMaxLifetime(time.Hour)

	return conn, nil
}

// Start logic of server main module
func (s *Server) Start() (err error) {
	ctx := context.Background()
	startCtx, startSpan := observability.Observer.NewSpan(ctx, observability.SpanKindInternal, "start_server_service")
	defer startSpan.End()

	logrus.WithContext(startCtx).Info("vxserver is starting...")
	serverConfig, err := s.getVXProtoServerConfig()
	if err != nil {
		logrus.WithContext(startCtx).WithError(err).Info("failed to start vxserver")
		return fmt.Errorf("failed to get server config: %w", err)
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		err = s.module.Start(serverConfig)
	}()

	if s.config.IsProfiling {
		go s.runProfiling()
	}

	// Wait a little time to catch error on start
	time.Sleep(time.Second)
	logrus.WithContext(startCtx).Info("vxserver started")

	return
}

func (s *Server) runProfiling() {
	mux := http.NewServeMux()
	mux.HandleFunc("/profiler/", pprof.Index)
	mux.HandleFunc("/profiler/profile", pprof.Profile)
	mux.HandleFunc("/profiler/cmdline", pprof.Cmdline)
	mux.HandleFunc("/profiler/symbol", pprof.Symbol)
	mux.HandleFunc("/profiler/trace", pprof.Trace)
	mux.HandleFunc("/profiler/allocs", pprof.Handler("allocs").ServeHTTP)
	mux.HandleFunc("/profiler/block", pprof.Handler("block").ServeHTTP)
	mux.HandleFunc("/profiler/goroutine", pprof.Handler("goroutine").ServeHTTP)
	mux.HandleFunc("/profiler/heap", pprof.Handler("heap").ServeHTTP)
	mux.HandleFunc("/profiler/mutex", pprof.Handler("mutex").ServeHTTP)
	mux.HandleFunc("/profiler/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	logrus.WithField(loggerKey, "init").WithError(http.ListenAndServe(":7777", mux)).Info("profiling monitor was exited")
}

func (s *Server) getVXProtoServerConfig() (*vxproto.ServerConfig, error) {
	c := &vxproto.CommonConfig{
		Host: s.config.Listen,
	}
	var err error
	c.TLSConfig, err = s.getTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get the server's TLS config")
	}
	return &vxproto.ServerConfig{
		CommonConfig: c,
		API:          s.config.APIVersionsConfig,
	}, nil
}

func (s *Server) getTLSConfig() (*tls.Config, error) {
	c := &tls.Config{
		MinVersion:               tls.VersionTLS13,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		ClientAuth:               tls.RequireAndVerifyClientCert,
	}
	var err error
	c.ClientCAs, err = s.createClientCertPool()
	if err != nil {
		return nil, err
	}
	sc, err := s.certProvider.SC()
	if err != nil {
		return nil, fmt.Errorf("failed to get server certificate: %w", err)
	}
	c.Certificates = sc
	return c, nil
}

func (s *Server) createClientCertPool() (*x509.CertPool, error) {
	vxca, err := s.certProvider.VXCA()
	if err != nil {
		return nil, fmt.Errorf("failed to get the VXCA from the cert provider: %w", err)
	}
	clientCertPool := x509.NewCertPool()
	if !clientCertPool.AppendCertsFromPEM(vxca) {
		return nil, fmt.Errorf("failed to append the client root certificate to the pool: %w", err)
	}
	return clientCertPool, nil
}

// Stop logic of server main module
func (s *Server) Stop() (err error) {
	ctx := context.Background()
	stopCtx, stopSpan := observability.Observer.NewSpan(ctx, observability.SpanKindInternal, "stop_server_service")
	logrus.WithContext(stopCtx).Info("vxserver is stopping...")
	stopSpan.End()
	if err = s.module.Stop(); err != nil {
		return
	}
	if err = s.tracerClient.Stop(stopCtx); err != nil {
		logrus.WithError(err).Warn("stopping the tracer client has returned an error")
	}
	if err = s.metricsClient.Stop(stopCtx); err != nil {
		logrus.WithError(err).Warn("stopping the metrics client has returned an error")
	}
	logrus.WithContext(stopCtx).Info("vxserver is waiting of modules release...")
	s.wg.Wait()
	logrus.WithContext(stopCtx).Info("vxserver stopped")

	return
}

// storeConfig is storing of current server configuration to file
func (s *Server) storeConfig() (string, error) {
	cfgDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", err
	}
	cfgPath := filepath.Join(cfgDir, "config.json")
	cfgData, _ := json.MarshalIndent(s.config, "", " ")

	if err := ioutil.WriteFile(cfgPath, cfgData, 0o600); err != nil {
		return "", err
	}

	return cfgPath, nil
}

// Manage by daemon commands or run the daemon
func (s *Server) Manage() (string, error) {
	switch s.config.Command {
	case "install":
		configPath, err := s.storeConfig()
		if err != nil {
			logrus.WithField(loggerKey, "daemon").WithError(err).Error("failed to store config file")
			return "vxserver service install failed", err
		}
		opts := []string{
			"-service",
			"-config", configPath,
		}
		if s.config.Debug {
			opts = append(opts, "-debug")
		}
		return s.svc.Install(opts...)
	case "uninstall":
		return s.svc.Remove()
	case "start":
		return s.svc.Start()
	case "stop":
		return s.svc.Stop()
	case "status":
		return s.svc.Status()
	case "":
	default:
		return "", fmt.Errorf("invalid value of 'command' argument: %s", s.config.Command)
	}

	if err := svc.Run(s); err != nil {
		logrus.WithField(loggerKey, "runtime").WithError(err).Error("vxserver executing failed")
		return "vxserver running failed", err
	}

	logrus.WithField(loggerKey, "shutdown").Info("vxserver exited normaly")
	return "vxserver exited normaly", nil
}

func getLogDir(configLogDir string) (string, error) {
	if len(configLogDir) == 0 {
		configLogDir = filepath.Dir(os.Args[0])
	}
	logDir, err := filepath.Abs(configLogDir)
	if err != nil {
		return "", fmt.Errorf("invalid value of 'logdir' argument: %s", logDir)
	}
	return logDir, nil
}

func initObserver(server *Server, _ *logrus.Entry) (func(), error) {
	server.tracerClient = observability.NewProxyTracerClient(
		observability.NewOtlpTracerClient(server.config.OtelAddr),
		observability.NewHookTracerClient(&observability.HookClientConfig{
			ResendTimeout:   100 * time.Millisecond,
			QueueSizeLimit:  100 * 1024 * 1024, // 100 MB
			PacketSizeLimit: 500 * 1024,        // 500 KB
		}),
	)
	server.metricsClient = observability.NewProxyMeterClient(
		observability.NewOtlpMeterClient(server.config.OtelAddr),
		observability.NewHookMeterClient(&observability.HookClientConfig{
			ResendTimeout:   100 * time.Millisecond,
			QueueSizeLimit:  100 * 1024 * 1024, // 100 MB
			PacketSizeLimit: 500 * 1024,        // 500 KB
		}),
	)
	attr := attribute.String("server_id", system.MakeAgentID())
	ctx := context.Background()
	tracerProvider, err := observability.NewTracerProvider(ctx, server.tracerClient, serviceName, server.version, attr)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a tracer provider: %w", err)
	}
	meterProvider, err := observability.NewMeterProvider(ctx, server.metricsClient, serviceName, server.version, attr)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a metrics provider: %w", err)
	}

	logLevels := []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	}
	if server.config.Debug {
		logrus.SetLevel(logrus.DebugLevel)
		logLevels = append(logLevels, logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
	observability.InitObserver(ctx, tracerProvider, meterProvider, server.tracerClient, server.metricsClient, serviceName, logLevels)
	return func() {
		observability.Observer.Close()
	}, nil
}

func overrideEnv() {
	// override specific environment variables
	serverPrefix := "AGENT_SERVER_"
	for _, env := range os.Environ() {
		envPair := strings.Split(env, "=")
		if len(envPair) != 2 {
			continue
		}
		if strings.HasPrefix(envPair[0], serverPrefix) {
			os.Setenv(strings.TrimPrefix(envPair[0], serverPrefix), envPair[1])
		}
	}
}

func main() {
	overrideEnv()
	logger := logrus.WithField(loggerKey, "daemon")
	if err := run(logger); err != nil {
		logger.WithError(err).Fatalf("failed to run %s", serviceName)
	}
}

func run(logger *logrus.Entry) error {
	cfg, err := config.ReadConfig()
	if err != nil {
		return fmt.Errorf("failed to get the configuration: %w", err)
	}
	server := Server{
		config: cfg,
	}
	if PackageVer != "" {
		server.version = PackageVer
	} else {
		server.version = "develop"
	}
	if PackageRev != "" {
		server.version += "-" + PackageRev
	}
	if cfg.IsPrintVersionOnly {
		logger.Infof("%s version is %s", serviceName, server.version)
		return nil
	}

	server.config.LogDir, err = getLogDir(server.config.LogDir)
	if err != nil {
		return err
	}

	logFile := &lumberjack.Logger{
		Filename:   filepath.Join(server.config.LogDir, "server.log"),
		MaxSize:    100,
		MaxBackups: 7,
		MaxAge:     14,
		Compress:   true,
	}
	if server.config.Service {
		logrus.SetOutput(logFile)
	} else {
		logrus.SetOutput(io.MultiWriter(os.Stdout, logFile))
	}

	observerTearDown, err := initObserver(&server, logrus.NewEntry(logrus.StandardLogger()))
	if err != nil {
		return fmt.Errorf("failed to initialize an observer: %w", err)
	}
	defer observerTearDown()

	kind := daemon.SystemDaemon
	var dependencies []string
	switch targetOS := runtime.GOOS; targetOS {
	case "linux":
		dependencies = []string{"multi-user.target", "sockets.target"}
	case "windows":
		dependencies = []string{"tcpip"}
	case "darwin":
		if os.Geteuid() == 0 {
			kind = daemon.GlobalDaemon
		} else {
			kind = daemon.UserAgent
		}
	default:
		return fmt.Errorf("OS %s is not supported", targetOS)
	}

	server.svc, err = daemon.New(serviceName, "VXServer service to agents control", kind, dependencies...)
	if err != nil {
		return fmt.Errorf("failed to create the service %s: %w", serviceName, err)
	}

	status, err := server.Manage()
	if err != nil {
		return fmt.Errorf("%s: %w", status, err)
	}
	logger.Info(status)
	return nil
}
