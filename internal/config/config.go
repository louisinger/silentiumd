package config

import (
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	badgerdb "github.com/louisinger/silentiumd/internal/infrastructure/db/badger"
	"github.com/louisinger/silentiumd/internal/infrastructure/db/postgres"
	"github.com/louisinger/silentiumd/internal/infrastructure/jsonrpc"
	"github.com/louisinger/silentiumd/internal/ports"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	LogLevelKey    = "LOG_LEVEL"
	NetworkKey     = "NETWORK"
	StartHeightKey = "START_HEIGHT"
	RpcCookiePath  = "RPC_COOKIE_PATH"
	RpcUserKey     = "RPC_USER"
	RpcPassKey     = "RPC_PASS"
	RpcHostKey     = "RPC_HOST"
	PortKey        = "PORT"
	NoTLSKey       = "NO_TLS"

	// db
	DbTypeKey        = "DB_TYPE"
	BadgerDatadirKey = "BADGER_DATADIR"
	PostgresDSNKey   = "POSTGRES_DSN"
)

var (
	defaultLogLevel    = 4 // logrus.InfoLevel
	defaultDatadir     = btcutil.AppDataDir("silentiumd", false)
	defaultNetwork     = "mainnet"
	defaultStartHeight = int32(0)
	defaultRpcHost     = "localhost:8332"
	defaultPort        = uint32(9000)
	defaultNoTLS       = true
)

type Config struct {
	StartHeight   int32
	ChainParams   chaincfg.Params
	RpcCookiePath string
	RpcUser       string
	RpcPass       string
	RpcHost       string
	LogLevel      logrus.Level
	Port          uint32
	NoTLS         bool

	DBType        string
	BadgerDatadir string
	PostgresDSN   string
}

func Load() (*Config, error) {
	viper.SetEnvPrefix("silentium")
	viper.AutomaticEnv()

	viper.SetDefault(LogLevelKey, defaultLogLevel)
	viper.SetDefault(BadgerDatadirKey, defaultDatadir)
	viper.SetDefault(DbTypeKey, "badger")
	viper.SetDefault(StartHeightKey, defaultStartHeight)
	viper.SetDefault(NetworkKey, defaultNetwork)
	viper.SetDefault(RpcHostKey, defaultRpcHost)
	viper.SetDefault(PortKey, defaultPort)
	viper.SetDefault(NoTLSKey, defaultNoTLS)

	network := viper.GetString(NetworkKey)
	chainParams, err := toChainParams(network)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		StartHeight:   viper.GetInt32(StartHeightKey),
		RpcCookiePath: viper.GetString(RpcCookiePath),
		RpcUser:       viper.GetString(RpcUserKey),
		RpcPass:       viper.GetString(RpcPassKey),
		LogLevel:      logrus.Level(viper.GetUint32(LogLevelKey)),
		ChainParams:   chainParams,
		RpcHost:       viper.GetString(RpcHostKey),
		Port:          viper.GetUint32(PortKey),
		NoTLS:         viper.GetBool(NoTLSKey),
		DBType:        viper.GetString(DbTypeKey),
		BadgerDatadir: viper.GetString(BadgerDatadirKey),
		PostgresDSN:   viper.GetString(PostgresDSNKey),
	}

	logrus.SetLevel(cfg.LogLevel)

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.RpcCookiePath == "" {
		if c.RpcUser == "" || c.RpcPass == "" {
			return fmt.Errorf("rpc user and pass or cookie path must be set")
		}

		logrus.Warn("you're using rpc user and pass, consider using cookie file instead")
	}

	if c.DBType != "badger" && c.DBType != "postgres" {
		return fmt.Errorf("unknown db type: %s", c.DBType)
	}

	if c.DBType == "badger" && c.BadgerDatadir == "" {
		return fmt.Errorf("badger datadir must be set")
	}

	if c.DBType == "postgres" && c.PostgresDSN == "" {
		return fmt.Errorf("postgres dsn must be set")
	}

	return nil
}

func (c *Config) GetRepository() (ports.ScalarRepository, error) {
	switch c.DBType {
	case "badger":
		return badgerdb.New(c.BadgerDatadir, logrus.StandardLogger())
	case "postgres":
		return postgres.New(postgres.PostreSQLConfig{Dsn: c.PostgresDSN})
	default:
		return nil, fmt.Errorf("unknown db type: %s", c.DBType)
	}
}

func (c *Config) GetChainsource() (ports.ChainSource, error) {
	if c.RpcCookiePath == "" {
		return jsonrpc.NewUnsafe(c.RpcHost, c.RpcUser, c.RpcPass)
	}

	return jsonrpc.New(c.RpcHost, c.RpcCookiePath)
}

func toChainParams(network string) (chaincfg.Params, error) {
	switch network {
	case "mainnet":
		return chaincfg.MainNetParams, nil
	case "testnet":
		return chaincfg.TestNet3Params, nil
	case "regtest":
		return chaincfg.RegressionNetParams, nil
	default:
		return chaincfg.Params{}, fmt.Errorf("unknown network: %s", network)
	}
}
