package application

import (
	"JiraConnector/connector/config"
	"fmt"
)

type Application struct {
	cfg *config.Config
}

var App *Application

func Applicate() error {
	cfg, err := config.NewConfig()
	if err != nil {
		return fmt.Errorf("error during open configuration: %v", err)
	}
	App = &Application{}
	App.cfg = cfg
	return nil
}

func (app *Application) Config() *config.Config {
	return app.cfg
}
