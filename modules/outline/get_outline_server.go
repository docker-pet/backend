package outline

import (
	"fmt"
	"strings"

	"github.com/docker-pet/backend/models"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func (m *OutlineModule) GetServerBySlug(slug string) (*models.OutlineServer, error) {
	record, err := m.Ctx.App.FindFirstRecordByFilter(
		"outline_servers",
		"slug={:slug}",
		dbx.Params{"slug": slug},
	)

	if err != nil {
		return nil, err
	}

	return ProxyOutlineServer(record), nil
}

func (m *OutlineModule) GetServerById(id string) (*models.OutlineServer, error) {
	record, err := m.Ctx.App.FindRecordById("outline_servers", id)

	if err != nil {
		return nil, err
	}

	return ProxyOutlineServer(record), nil
}

func (m *OutlineModule) GetAllServers() ([]*models.OutlineServer, error) {
	return proxyServerList(m.Ctx.App.FindAllRecords("outline_servers"))
}

func (m *OutlineModule) GetAllActiveServers() ([]*models.OutlineServer, error) {
	return proxyServerList(m.Ctx.App.FindAllRecords("outline_servers", dbx.HashExp{"enabled": true}))
}

func proxyServerList(records []*core.Record, err error) ([]*models.OutlineServer, error) {
	if err != nil {
		return nil, err
	}

	outlineServers := make([]*models.OutlineServer, len(records))
	for i, record := range records {
		outlineServers[i] = ProxyOutlineServer(record)
	}

	return outlineServers, nil
}

func (m *OutlineModule) formatJobDomain(server *models.OutlineServer) string {
	domain := server.OverrideDomain()
	if domain == "" {
		domain = fmt.Sprintf("%s.%s", server.Slug(), m.appConfig.AppConfig().AppDomain())
	}

	return domain
}

func (m *OutlineModule) formatConnectDomain(server *models.OutlineServer, user *models.User) string {
	domain := server.OverrideDomain()
	if user.OutlineReverseServerEnabled() && server.ReverseDomain() != "" {
		domain = server.ReverseDomain()
	}

	if domain != "" && !strings.Contains(domain, ".") {
    	domain = fmt.Sprintf("%s.%s", domain, m.appConfig.AppConfig().AppDomain())
	}

	if domain == "" {
		domain = fmt.Sprintf("%s.%s", server.Slug(), m.appConfig.AppConfig().AppDomain())
	}

	return domain
}

func ProxyOutlineServer(record *core.Record) *models.OutlineServer {
	outlineServer := &models.OutlineServer{}
	outlineServer.SetProxyRecord(record)
	return outlineServer
}
