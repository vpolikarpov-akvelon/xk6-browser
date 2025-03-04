package k6test

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v3"

	"github.com/grafana/xk6-browser/env"
	"github.com/grafana/xk6-browser/k6ext"

	k6eventloop "go.k6.io/k6/js/eventloop"
	k6modulestest "go.k6.io/k6/js/modulestest"
	k6lib "go.k6.io/k6/lib"
	k6executor "go.k6.io/k6/lib/executor"
	k6testutils "go.k6.io/k6/lib/testutils"
	k6metrics "go.k6.io/k6/metrics"
)

// VU is a k6 VU instance.
// TODO: Do we still need this VU wrapper?
// ToGojaValue can be a helper function that takes a goja.Runtime (although it's
// not much of a helper from calling ToValue(i) directly...), and we can access
// EventLoop from modulestest.Runtime.EventLoop.
type VU struct {
	*k6modulestest.VU
	Loop      *k6eventloop.EventLoop
	toBeState *k6lib.State
	samples   chan k6metrics.SampleContainer
}

// ToGojaValue is a convenience method for converting any value to a goja value.
func (v *VU) ToGojaValue(i any) goja.Value { return v.Runtime().ToValue(i) }

// ActivateVU mimicks activation of the VU as in k6.
// It transitions the VU from the init stage to the execution stage by
// setting the VU's state to the state that was passed to NewVU.
func (v *VU) ActivateVU() {
	v.VU.StateField = v.toBeState
	v.VU.InitEnvField = nil
}

// AssertSamples asserts each sample VU received since AssertSamples
// is last called, then it returns the number of received samples.
func (v *VU) AssertSamples(assertSample func(s k6metrics.Sample)) int {
	var n int
	for _, bs := range k6metrics.GetBufferedSamples(v.samples) {
		for _, s := range bs.GetSamples() {
			assertSample(s)
			n++
		}
	}
	return n
}

// WithSamples is used to indicate we want to use a bidirectional channel
// so that the test can read the metrics being emitted to the channel.
type WithSamples chan k6metrics.SampleContainer

// NewVU returns a mock k6 VU.
//
// opts can be one of the following:
//   - WithSamplesListener: a bidirectional channel that will be used to emit metrics.
//   - env.LookupFunc: a lookup function that will be used to lookup environment variables.
func NewVU(tb testing.TB, opts ...any) *VU {
	tb.Helper()

	var (
		samples    = make(chan k6metrics.SampleContainer, 1000)
		lookupFunc = env.EmptyLookup
	)
	for _, opt := range opts {
		switch opt := opt.(type) {
		case WithSamples:
			samples = opt
		case env.LookupFunc:
			lookupFunc = opt
		}
	}

	root, err := k6lib.NewGroup("", nil)
	require.NoError(tb, err)

	testRT := k6modulestest.NewRuntime(tb)
	testRT.VU.InitEnvField.LookupEnv = lookupFunc

	tags := testRT.VU.InitEnvField.Registry.RootTagSet()

	state := &k6lib.State{
		Options: k6lib.Options{
			MaxRedirects: null.IntFrom(10),
			UserAgent:    null.StringFrom("TestUserAgent"),
			Throw:        null.BoolFrom(true),
			SystemTags:   &k6metrics.DefaultSystemTagSet,
			Batch:        null.IntFrom(20),
			BatchPerHost: null.IntFrom(20),
			// HTTPDebug:    null.StringFrom("full"),
			Scenarios: k6lib.ScenarioConfigs{
				"default": &TestExecutor{
					BaseConfig: k6executor.BaseConfig{
						Options: &k6lib.ScenarioOptions{
							Browser: map[string]any{
								"type": "chromium",
							},
						},
					},
				},
			},
		},
		Logger:         k6testutils.NewLogger(tb),
		Group:          root,
		BufferPool:     k6lib.NewBufferPool(),
		Samples:        samples,
		Tags:           k6lib.NewVUStateTags(tags.With("group", root.Path)),
		BuiltinMetrics: k6metrics.RegisterBuiltinMetrics(k6metrics.NewRegistry()),
	}

	ctx := k6ext.WithVU(testRT.VU.CtxField, testRT.VU)
	ctx = k6lib.WithScenarioState(ctx, &k6lib.ScenarioState{Name: "default"})
	testRT.VU.CtxField = ctx

	return &VU{VU: testRT.VU, Loop: testRT.EventLoop, toBeState: state, samples: samples}
}
