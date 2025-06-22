package models

type OutlineServerSyncRemote struct {
	RemoteAdminEndpoint      string                  `json:"remoteAdminEndpoint"`
	RemoteAdminBasicAuth     *OutlineServerBasicAuth `json:"remoteAdminBasicAuth"`
	RemoteSyncCronExpression string                  `json:"remoteSyncCronExpression"`
	OutlineConfig            *OutlineConfiguration   `json:"outlineConfig"`
}

type OutlineServerSyncLocal struct {
	OutlineConfig            *OutlineConfiguration `json:"outlineConfig"`
}

type OutlineConfiguration struct {
	TCP *OutlineConfigurationProtocol `json:"tcp"`
	UDP *OutlineConfigurationProtocol `json:"udp"`
}

type OutlineConfigurationProtocol struct {
	Port   int    `json:"port"`
	Path   string `json:"path"`
	Prefix string `json:"prefix"`
}

type OutlineServerBasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
