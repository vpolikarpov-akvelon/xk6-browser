package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/xk6-browser/browser"
	"github.com/grafana/xk6-browser/chromium"
	"github.com/grafana/xk6-browser/env"
	"github.com/grafana/xk6-browser/k6ext/k6test"
)

func TestBrowserTypeConnect(t *testing.T) {
	// Start a test browser so we can get its WS URL
	// and use it to connect through BrowserType.Connect.
	tb := newTestBrowser(t)
	vu := k6test.NewVU(t)
	bt := chromium.NewBrowserType(vu)
	vu.ActivateVU()

	b, err := bt.Connect(context.Background(), tb.wsURL)
	require.NoError(t, err)
	_, err = b.NewPage(nil)
	require.NoError(t, err)
}

func TestBrowserTypeLaunchToConnect(t *testing.T) {
	tb := newTestBrowser(t)
	bp := newTestBrowserProxy(t, tb)

	// Export WS URL env var
	// pointing to test browser proxy
	vu := k6test.NewVU(t, env.ConstLookup(env.WebSocketURLs, bp.wsURL()))

	// We have to call launch method through JS API in Goja
	// to take mapping layer into account, instead of calling
	// BrowserType.Launch method directly
	root := browser.New()
	mod := root.NewModuleInstance(vu)
	jsMod, ok := mod.Exports().Default.(*browser.JSModule)
	require.Truef(t, ok, "unexpected default mod export type %T", mod.Exports().Default)

	vu.ActivateVU()

	rt := vu.Runtime()
	require.NoError(t, rt.Set("browser", jsMod.Browser))
	_, err := rt.RunString(`
		const p = browser.newPage();
		p.close();
	`)
	require.NoError(t, err)

	// Verify the proxy, which's WS URL was set as
	// K6_BROWSER_WS_URL, has received a connection req
	require.True(t, bp.connected)
	// Verify that no new process pids have been added
	// to pid registry
	require.Len(t, root.PidRegistry.Pids(), 0)
}
