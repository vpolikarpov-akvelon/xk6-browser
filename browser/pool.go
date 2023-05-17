package browser

import (
	"sync"

	"github.com/grafana/xk6-browser/api"
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
