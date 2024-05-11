package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/louisinger/silentiumd/internal/application"
	"github.com/louisinger/silentiumd/internal/config"
	grpcservice "github.com/louisinger/silentiumd/internal/interface/grpc"
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

	scalarsRepository, err := cfg.GetRepository()
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
			AppService: silentiumSvc,
			Port:       cfg.Port,
			TLSKey:     cfg.KeyFileTLS,
			TLSCert:    cfg.CertFileTLS,
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
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	<-sigChan

	if err := service.Stop(); err != nil {
		log.Fatal(err)
	}

	logrus.Info("shutting down service...")
	logrus.Exit(0)
}
