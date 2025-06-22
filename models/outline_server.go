package models

import (
	"encoding/json"

	"github.com/biter777/countries"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

var _ core.RecordProxy = (*OutlineServer)(nil)

type OutlineServer struct {
	core.BaseRecordProxy
}

func (a *OutlineServer) Slug() string {
	return a.GetString("slug")
}

func (a *OutlineServer) SetSlug(value string) {
	a.Set("slug", value)
}

func (a *OutlineServer) Country() countries.CountryCode {
	return countries.ByName(a.GetString("country"))
}

func (a *OutlineServer) SetCountry(value string) {
	a.Set("country", value)
}

func (a *OutlineServer) Enabled() bool {
	return a.GetBool("enabled")
}

func (a *OutlineServer) SetEnabled(value bool) {
	a.Set("enabled", value)
}

func (a *OutlineServer) Premium() bool {
	return a.GetBool("premium")
}

func (a *OutlineServer) SetPremium(value bool) {
	a.Set("premium", value)
}

func (a *OutlineServer) OverrideDomain() string {
	return a.GetString("overrideDomain")
}

func (a *OutlineServer) SetOverrideDomain(value string) {
	a.Set("overrideDomain", value)
}

func (a *OutlineServer) SyncType() OutlineServerSyncType {
	return OutlineServerSyncType(a.GetString("syncType"))
}

func (a *OutlineServer) SetSyncType(value OutlineServerSyncType) {
	a.Set("syncType", string(value))
}

func (a *OutlineServer) OutlineConfig() *OutlineConfiguration {
	if a.SyncType() == OutlineLocalSync {
		return a.SyncLocalConfig().OutlineConfig
	} else {
		return a.SyncRemoteConfig().OutlineConfig
	}
}

func (a *OutlineServer) SyncRemoteConfig() *OutlineServerSyncRemote {
	var config OutlineServerSyncRemote
	a.UnmarshalJSONField("syncConfig", &config)
	if config.OutlineConfig == nil {
		config.OutlineConfig = &OutlineConfiguration{}
	}
	if config.OutlineConfig.TCP == nil {
		config.OutlineConfig.TCP = &OutlineConfigurationProtocol{}
	}
	if config.OutlineConfig.UDP == nil {
		config.OutlineConfig.UDP = &OutlineConfigurationProtocol{}
	}
	if config.RemoteAdminBasicAuth == nil {
		config.RemoteAdminBasicAuth = &OutlineServerBasicAuth{}
	}

	return &config
}

func (a *OutlineServer) SetSyncRemoteConfig(config *OutlineServerSyncRemote) {
	if config.OutlineConfig == nil {
		config.OutlineConfig = &OutlineConfiguration{}
	}
	if config.OutlineConfig.TCP == nil {
		config.OutlineConfig.TCP = &OutlineConfigurationProtocol{}
	}
	if config.OutlineConfig.UDP == nil {
		config.OutlineConfig.UDP = &OutlineConfigurationProtocol{}
	}
	if config.RemoteAdminBasicAuth == nil {
		config.RemoteAdminBasicAuth = &OutlineServerBasicAuth{}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		panic(err)
	}
	a.Set("syncConfig", data)
}

func (a *OutlineServer) SyncLocalConfig() *OutlineServerSyncLocal {
	var config OutlineServerSyncLocal
	a.UnmarshalJSONField("syncConfig", &config)
	if config.OutlineConfig == nil {
		config.OutlineConfig = &OutlineConfiguration{}
	}
	if config.OutlineConfig.TCP == nil {
		config.OutlineConfig.TCP = &OutlineConfigurationProtocol{}
	}
	if config.OutlineConfig.UDP == nil {
		config.OutlineConfig.UDP = &OutlineConfigurationProtocol{}
	}

	return &config
}

func (a *OutlineServer) SetSyncLocalConfig(config *OutlineServerSyncLocal) {
	if config.OutlineConfig == nil {
		config.OutlineConfig = &OutlineConfiguration{}
	}
	if config.OutlineConfig.TCP == nil {
		config.OutlineConfig.TCP = &OutlineConfigurationProtocol{}
	}
	if config.OutlineConfig.UDP == nil {
		config.OutlineConfig.UDP = &OutlineConfigurationProtocol{}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		panic(err)
	}
	a.Set("syncConfig", data)
}

func (a *OutlineServer) MetricsSecret() string {
	return a.GetString("metricsSecret")
}

func (a *OutlineServer) GenerateMetricsSecret() {
	a.Set("metricsSecret", security.RandomString(32))
}
