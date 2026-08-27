package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type config struct {
	Addr      string
	DataDir   string
	Selfcheck bool
}

func parseConfig(args []string) (config, error) {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	cfg := config{}
	fs.StringVar(&cfg.Addr, "addr", "127.0.0.1:19081", "HTTP 监听地址")
	fs.StringVar(&cfg.DataDir, "data", "./data", "事件日志与投影目录")
	fs.BoolVar(&cfg.Selfcheck, "selfcheck", false, "运行完整 HTTP 自检后退出")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	explicitAddr := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "addr" {
			explicitAddr = true
		}
	})
	if !explicitAddr {
		if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
			number, err := strconv.Atoi(port)
			if err != nil || number < 1 || number > 65535 {
				return cfg, fmt.Errorf("PORT 必须为 1 到 65535 的端口号")
			}
			cfg.Addr = net.JoinHostPort("127.0.0.1", strconv.Itoa(number))
		}
	}
	if err := validateAddress(cfg.Addr); err != nil {
		return cfg, err
	}
	if strings.TrimSpace(cfg.DataDir) == "" {
		return cfg, fmt.Errorf("-data 不能为空")
	}
	return cfg, nil
}

func validateAddress(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("-addr 必须为 host:port：%w", err)
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return fmt.Errorf("-addr 端口无效")
	}
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("-addr 必须使用回环地址")
	}
	return nil
}
