package browser

import (
	"sync"

	"github.com/grafana/xk6-browser/api"
	"github.com/grafana/xk6-browser/common"
)

type browserPool struct {
	m sync.Map
}

func (p *browserPool) setBrowser(id string, b api.Browser) {
	p.m.Store(id, b)
}

func (p *browserPool) getBrowser(id string) (b api.Browser, ok bool) {
	e, ok := p.m.Load(id)
	if ok {
		b, ok = e.(api.Browser)
		return b, ok
	}

	return nil, false
}

func (p *browserPool) deleteBrowser(id string) {
	p.m.Delete(id)
}

type browserProcessPool struct {
	mu sync.Mutex
	m  map[string]*common.BrowserProcess
}

func newBrowserProcessPool() *browserProcessPool {
	return &browserProcessPool{
		m: make(map[string]*common.BrowserProcess),
	}
}

func (p *browserProcessPool) setBrowserProcess(handle string, bp *common.BrowserProcess) {
	p.m[handle] = bp
}

func (p *browserProcessPool) getBrowserProcess(handle string) (bp *common.BrowserProcess, ok bool) {
	bp, ok = p.m[handle]
	return
}

func (p *browserProcessPool) lockBrowserProcess() {
	p.mu.Lock()
}

func (p *browserProcessPool) unlockBrowserProcess() {
	p.mu.Unlock()
}
