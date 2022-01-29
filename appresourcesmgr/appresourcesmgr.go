package appresourcesmgr

import (
	// "fmt"
	"github.com/xtforgame/agak/config"
)

type AppResourcesManager struct {
	appConfig *config.AppConfig
}

func NewAppResourcesManager(appConfig *config.AppConfig) *AppResourcesManager {
	appResourcesMgr := &AppResourcesManager{
		appConfig: appConfig,
	}
	return appResourcesMgr
}

func (appResourcesMgr *AppResourcesManager) Init() {
}
