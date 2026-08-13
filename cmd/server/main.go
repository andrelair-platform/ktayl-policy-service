package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/internal/api"
	"github.com/spf13/viper"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	viper.SetDefault("port", "8080")
	viper.SetDefault("read_timeout_s", 10)
	viper.SetDefault("write_timeout_s", 10)
	viper.AutomaticEnv()

	port := viper.GetString("port")
	addr := net.JoinHostPort("", port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      api.NewRouter(log),
		ReadTimeout:  time.Duration(viper.GetInt("read_timeout_s")) * time.Second,
		WriteTimeout: time.Duration(viper.GetInt("write_timeout_s")) * time.Second,
	}

	go func() {
		log.Info("starting ktayl-policy-service", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown error", "err", err)
	}
	log.Info("server stopped")
}
