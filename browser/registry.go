package browser

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"

	"github.com/grafana/xk6-browser/api"
	"github.com/grafana/xk6-browser/env"
	"github.com/grafana/xk6-browser/otel"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// pidRegistry keeps track of the launched browser process IDs.
type pidRegistry struct {
	mu  sync.RWMutex
	ids []int
}

// registerPid registers the launched browser process ID.
func (r *pidRegistry) registerPid(pid int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ids = append(r.ids, pid)
}

// Pids returns the launched browser process IDs.
func (r *pidRegistry) Pids() []int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pids := make([]int, len(r.ids))
	copy(pids, r.ids)

	return pids
}

// remoteRegistry contains the details of the remote web browsers.
// At the moment it's the WS URLs.
type remoteRegistry struct {
	isRemote bool
	wsURLs   []string
}

// newRemoteRegistry will create a new RemoteRegistry. This will
// parse the K6_BROWSER_WS_URL env var to retrieve the defined
// list of WS URLs.
//
// K6_BROWSER_WS_URL can be defined as a single WS URL or a
// comma separated list of URLs.
func newRemoteRegistry(envLookup env.LookupFunc) (*remoteRegistry, error) {
	r := &remoteRegistry{}

	isRemote, wsURLs, err := checkForScenarios(envLookup)
	if err != nil {
		return nil, err
	}
	if isRemote {
		r.isRemote = isRemote
		r.wsURLs = wsURLs
		return r, nil
	}

	r.isRemote, r.wsURLs = checkForBrowserWSURLs(envLookup)

	return r, nil
}

func checkForBrowserWSURLs(envLookup env.LookupFunc) (bool, []string) {
	wsURL, isRemote := envLookup(env.WebSocketURLs)
	if !isRemote {
		return false, nil
	}

	if !strings.ContainsRune(wsURL, ',') {
		return true, []string{wsURL}
	}

	// If last parts element is a void string,
	// because WS URL contained an ending comma,
	// remove it
	parts := strings.Split(wsURL, ",")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}

	return true, parts
}

// checkForScenarios will parse the K6_INSTANCE_SCENARIOS env var if
// it has been defined.
func checkForScenarios(envLookup env.LookupFunc) (bool, []string, error) {
	scenariosJSON, isRemote := envLookup(env.InstanceScenarios)
	if !isRemote {
		return false, nil, nil
	}
	// prevent failing in unquoting empty string.
	if scenariosJSON == "" {
		return false, nil, nil
	}
	scenariosJSON, err := strconv.Unquote(scenariosJSON)
	if err != nil {
		return false, nil, fmt.Errorf("unqouting K6_INSTANCE_SCENARIOS: %w", err)
	}

	var scenarios []struct {
		ID       string `json:"id"`
		Browsers []struct {
			Handle string `json:"handle"`
		} `json:"browsers"`
	}
	if err := json.Unmarshal([]byte(scenariosJSON), &scenarios); err != nil {
		return false, nil, fmt.Errorf("parsing K6_INSTANCE_SCENARIOS: %w", err)
	}

	var wsURLs []string
	for _, s := range scenarios {
		for _, b := range s.Browsers {
			if strings.TrimSpace(b.Handle) == "" {
				continue
			}
			wsURLs = append(wsURLs, b.Handle)
		}
	}
	if len(wsURLs) == 0 {
		return false, wsURLs, nil
	}

	return true, wsURLs, nil
}

// isRemoteBrowser returns a WS URL and true when a WS URL is defined,
// otherwise it returns an empty string and false. If more than one
// WS URL was registered in newRemoteRegistry, a randomly chosen URL from
// the list in a round-robin fashion is selected and returned.
func (r *remoteRegistry) isRemoteBrowser() (string, bool) {
	if !r.isRemote {
		return "", false
	}

	// Choose a random WS URL from the provided list
	i, _ := rand.Int(rand.Reader, big.NewInt(int64(len(r.wsURLs))))
	wsURL := r.wsURLs[i.Int64()]

	return wsURL, true
}

// browserRegistry stores browser instances indexed per
// iteration as identified by VUID-scenario-iterationID.
type browserRegistry struct {
	m sync.Map
}

func (p *browserRegistry) setBrowser(id string, b api.Browser) {
	p.m.Store(id, b)
}

func (p *browserRegistry) getBrowser(id string) (b api.Browser, ok bool) {
	e, ok := p.m.Load(id)
	if ok {
		b, ok = e.(api.Browser)
		return b, ok
	}

	return nil, false
}

func (p *browserRegistry) deleteBrowser(id string) {
	p.m.Delete(id)
}

// trace represents a traces registry entry which holds the
// root span for the trace and a context that wraps that span.
type trace struct {
	ctx      context.Context
	rootSpan oteltrace.Span
}

type tracesRegistry struct {
	ctx context.Context
	tp  otel.TraceProvider

	mu sync.Mutex
	m  map[string]*trace
}

func newTracesRegistry(ctx context.Context, envLookup env.LookupFunc) (*tracesRegistry, error) {
	if !isTracingEnabled(envLookup) {
		return &tracesRegistry{
			ctx: ctx,
			tp:  otel.NewNoopTraceProvider(),
			m:   make(map[string]*trace),
		}, nil
	}

	// TODO: Default fallback to HTTP and localhost:4318?
	// Seems like we are missing logging in registries/mapping layer
	endpoint, proto, insecure := parseTracingConfig(envLookup)
	if endpoint == "" || proto == "" {
		return nil, errors.New(
			"tracing is enabled but K6_BROWSER_TRACING_ENDPOINT or K6_BROWSER_TRACING_PROTO were not set",
		)
	}

	tp, err := otel.NewTraceProvider(ctx, proto, endpoint, insecure)
	if err != nil {
		return nil, fmt.Errorf("creating trace provider: %w", err)
	}

	return &tracesRegistry{
		tp: tp,
		m:  make(map[string]*trace),
	}, nil
}

func isTracingEnabled(envLookup env.LookupFunc) bool {
	vs, ok := envLookup(env.EnableTracing)
	if !ok {
		return false
	}

	v, err := strconv.ParseBool(vs)
	return err == nil && v
}

func parseTracingConfig(envLookup env.LookupFunc) (endpoint, proto string, insecure bool) {
	endpoint, _ = envLookup(env.TracingEndpoint)
	proto, _ = envLookup(env.TracingProto)
	insecureStr, _ := envLookup(env.TracingInsecure)
	insecure, _ = strconv.ParseBool(insecureStr)

	return endpoint, proto, insecure
}

func (r *tracesRegistry) traceCtx(id string) context.Context {
	r.mu.Lock()
	defer r.mu.Unlock()

	if t, ok := r.m[id]; ok {
		return t.ctx
	}

	// TODO: Move trace initialization to IterStart event handling
	// once integrated with k6 event system.
	spanCtx, span := otel.Trace(r.ctx, "IterStart")

	r.m[id] = &trace{
		ctx:      spanCtx,
		rootSpan: span,
	}

	return spanCtx
}

// TODO: Move trace end to IterEnd event handling
// once integrated with k6 event system.
func (r *tracesRegistry) endTrace(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if t, ok := r.m[id]; ok {
		t.rootSpan.End()
		delete(r.m, id)
	}
}
