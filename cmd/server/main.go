package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	webdelivery "benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/web"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "服务错误：", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}
	if cfg.Selfcheck {
		return runSelfcheck(cfg)
	}
	delivery, err := build(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("恢复持久化数据失败: %w", err)
	}
	server := webdelivery.NewHTTPServer(cfg.Addr, delivery.Handler())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.ListenAndServe() }()
	fmt.Printf("木年轮定年证据发布台已监听 http://%s\n", cfg.Addr)
	select {
	case sig := <-signals:
		fmt.Printf("收到 %s，准备关闭\n", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
