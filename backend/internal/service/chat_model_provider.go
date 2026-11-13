package service

import "github.com/Wei-Shaw/sub2api/internal/config"

func ProvideGatewayModelCatalogService(gateway *GatewayService, openai *OpenAIGatewayService, cfg *config.Config) *GatewayModelCatalogService {
	switches := 10
	if cfg != nil && cfg.Gateway.MaxAccountSwitches > 0 {
		switches = cfg.Gateway.MaxAccountSwitches
	}
	return NewGatewayModelCatalogService(gateway, openai, switches)
}
