package config

import (
	"fmt"
	"os"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/louisinger/echemythosd/internal/infrastructure/jsonrpc"
	"github.com/louisinger/echemythosd/internal/ports"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	LogLevelKey    = "LOG_LEVEL"
	DatadirKey     = "DATADIR"
	NetworkKey     = "NETWORK"
	StartHeightKey = "START_HEIGHT"
	RpcCookiePath  = "RPC_COOKIE_PATH"
	RpcUserKey     = "RPC_USER"
	RpcPassKey     = "RPC_PASS"
	RpcHostKey     = "RPC_HOST"
	PortKey        = "PORT"
	NoTLSKey       = "NO_TLS"
)

var (
	defaultLogLevel    = 4 // logrus.InfoLevel
	defaultDatadir     = btcutil.AppDataDir("echemythosd", false)
	defaultNetwork     = "mainnet"
	defaultStartHeight = int32(0)
	defaultRpcHost     = "localhost:8332"
	defaultPort        = uint32(9000)
	defaultNoTLS       = true
)

type Config struct {
	Datadir       string
	StartHeight   int32
	ChainParams   chaincfg.Params
	RpcCookiePath string
	RpcUser       string
	RpcPass       string
	RpcHost       string
	LogLevel      logrus.Level
	Port          uint32
	NoTLS         bool
}

func Load() (*Config, error) {
	viper.SetEnvPrefix("ECHEMYTHOS")
	viper.AutomaticEnv()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	viper.SetDefault(LogLevelKey, defaultLogLevel)
	viper.SetDefault(DatadirKey, defaultDatadir)
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
		Datadir:       viper.GetString(DatadirKey),
		StartHeight:   viper.GetInt32(StartHeightKey),
		RpcCookiePath: viper.GetString(RpcCookiePath),
		RpcUser:       viper.GetString(RpcUserKey),
		RpcPass:       viper.GetString(RpcPassKey),
		LogLevel:      logrus.Level(viper.GetUint32(LogLevelKey)),
		ChainParams:   chainParams,
		RpcHost:       viper.GetString(RpcHostKey),
		Port:          viper.GetUint32(PortKey),
		NoTLS:         viper.GetBool(NoTLSKey),
	}

	logrus.SetLevel(cfg.LogLevel)

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	if err := cfg.initDatadir(); err != nil {
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

	return nil
}

func (c *Config) initDatadir() error {
	err := os.Mkdir(c.Datadir, os.ModePerm)
	if os.IsExist(err) {
		return nil
	}

	return err
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
