package outline

import (
	"time"

	"github.com/zmwangx/debounce"
)

func (m *OutlineModule) watchKeysChanges() {
	debouncers := make(map[string]func())

	m.tokenStore.SubscribeOnChange(func(serverId string) {
		m.tokenStore.SubscribeOnChange(func(serverId string) {
			if _, ok := debouncers[serverId]; !ok {
				debouncers[serverId], _ = debounce.Debounce(
					func() { m.configureCaddy(serverId) },
					2*time.Second, // TODO: make configurable
					debounce.WithLeading(true),
					debounce.WithTrailing(true),
				)
			}
			debouncers[serverId]()
		})
	})
}
