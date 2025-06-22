package core

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	"resty.dev/v3"
)

type AppContext struct {
	App        core.App
	HttpClient *resty.Client
	Modules    map[string]Module
}

func NewHttpClient() *resty.Client {
	client := resty.New()
	client.SetRetryCount(3)
	client.SetRetryWaitTime(1 * time.Second)
	client.SetRetryMaxWaitTime(5 * time.Second)
	client.SetTimeout(7 * time.Second)
	return client
}
