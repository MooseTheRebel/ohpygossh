package gohpygossh

import "testing"

// SetGlobalKnownHostsPath points the default policy's system-wide
// known_hosts file at path until the test ends. Tests that call it must not
// run in parallel.
func SetGlobalKnownHostsPath(t testing.TB, path string) {
	old := globalKnownHostsPath
	globalKnownHostsPath = path
	t.Cleanup(func() { globalKnownHostsPath = old })
}
