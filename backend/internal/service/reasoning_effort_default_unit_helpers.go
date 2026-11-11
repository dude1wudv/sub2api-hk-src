//go:build unit

package service

// ResetGatewayForwardingSettingsCacheForUnitTest isolates handler fixtures that
// use different SettingService repositories within the same test process.
// The returned cleanup restores the previous cache without changing production.
func ResetGatewayForwardingSettingsCacheForUnitTest() func() {
	previous, _ := gatewayForwardingCache.Load().(*cachedGatewayForwardingSettings)
	gatewayForwardingCache.Store(&cachedGatewayForwardingSettings{})
	return func() { gatewayForwardingCache.Store(previous) }
}
