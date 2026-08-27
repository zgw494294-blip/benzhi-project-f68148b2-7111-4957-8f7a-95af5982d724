package main

import (
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/analysis"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/application"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/persistence"
	webdelivery "benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/web"
)

func build(dataDir string) (*webdelivery.Server, error) {
	repo, err := persistence.Open(dataDir)
	if err != nil {
		return nil, err
	}
	engine := analysis.NewEngine()
	app := application.NewService(repo, engine, "")
	return webdelivery.NewServer(app), nil
}
