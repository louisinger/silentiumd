package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/louisinger/echemythosd/internal/application"
	"github.com/louisinger/echemythosd/internal/config"
	badgerdb "github.com/louisinger/echemythosd/internal/infrastructure/db/badger"
	grpcservice "github.com/louisinger/echemythosd/internal/interface/grpc"
	"github.com/sirupsen/logrus"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("config OK")

	chainSource, err := cfg.GetChainsource()
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("chain source OK")

	scalarsRepository, err := badgerdb.NewScalarRepository(cfg.Datadir, logrus.StandardLogger())
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("db OK")

	service, err := application.NewSyncerService(
		scalarsRepository,
		chainSource,
		cfg.ChainParams,
		cfg.StartHeight,
	)
	if err != nil {
		logrus.Fatal(err)
	}

	if err := service.Start(); err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("syncer service OK")

	silentiumSvc := application.NewSilentiumService(scalarsRepository, chainSource)

	grpcSvc, err := grpcservice.NewService(
		grpcservice.Config{
			Port:       cfg.Port,
			AppService: silentiumSvc,
			NoTLS:      cfg.NoTLS,
		},
	)
	if err != nil {
		logrus.Fatal(err)
	}

	if err := grpcSvc.Start(); err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("grpc service OK")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan

	if err := service.Stop(); err != nil {
		log.Fatal(err)
	}

	logrus.Info("shutting down service...")
	logrus.Exit(0)
}
