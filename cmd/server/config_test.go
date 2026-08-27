package main

import "testing"

func TestConfigPortAndLoopback(t *testing.T) {
	t.Setenv("PORT", "19123")
	cfg, err := parseConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != "127.0.0.1:19123" {
		t.Fatalf("PORT 地址=%s", cfg.Addr)
	}
	if _, err := parseConfig([]string{"-addr=0.0.0.0:19081"}); err == nil {
		t.Fatal("应拒绝非回环监听")
	}
	cfg, err = parseConfig([]string{"-addr=127.0.0.1:19222"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != "127.0.0.1:19222" {
		t.Fatal("显式 -addr 未生效")
	}
}
