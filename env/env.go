// Package env provides types to interact with environment setup.
package env

// Execution specific.
const (
	// InstanceScenarios is an environment variable that can be used to
	// define the extra scenarios details to use when running remotely.
	InstanceScenarios = "K6_INSTANCE_SCENARIOS"

	// WebSocketURLs is an environment variable that can be used to
	// define the WS URLs to connect to when running remotely.
	WebSocketURLs = "K6_BROWSER_WS_URL"

	// BrowserArguments is an environment variable that can be used to
	// pass extra arguments to the browser process.
	BrowserArguments = "K6_BROWSER_ARGS"

	// BrowserExecutablePath is an environment variable that can be used
	// to define the path to the browser to execute.
	BrowserExecutablePath = "K6_BROWSER_EXECUTABLE_PATH"

	// BrowserEnableDebugging is an environment variable that can be used to
	// define if the browser should be launched with debugging enabled.
	BrowserEnableDebugging = "K6_BROWSER_DEBUG"

	// BrowserHeadless is an environment variable that can be used to
	// define if the browser should be launched in headless mode.
	BrowserHeadless = "K6_BROWSER_HEADLESS"

	// BrowserIgnoreDefaultArgs is an environment variable that can be
	// used to define if the browser should ignore default arguments.
	BrowserIgnoreDefaultArgs = "K6_BROWSER_IGNORE_DEFAULT_ARGS"

	// BrowserGlobalTimeout is an environment variable that can be used
	// to set the global timeout for the browser.
	BrowserGlobalTimeout = "K6_BROWSER_TIMEOUT"
)

// Logging and debugging.
const (
	// EnableProfiling is an environment variable that can be used to
	// enable profiling for the browser. It will start up a debugging
	// server on ProfilingServerAddr.
	EnableProfiling = "K6_BROWSER_ENABLE_PPROF"

	// ProfilingServerAddr is the address of the profiling server.
	ProfilingServerAddr = "localhost:6060"

	// LogCaller is an environment variable that can be used to enable
	// the caller function information in the browser logs.
	LogCaller = "K6_BROWSER_LOG_CALLER"

	// LogLevel is an environment variable that can be used to set the
	// log level for the browser logs.
	LogLevel = "K6_BROWSER_LOG"

	// LogCategoryFilter is an environment variable that can be used to
	// filter the browser logs based on their category. It supports
	// regular expressions.
	LogCategoryFilter = "K6_BROWSER_LOG_CATEGORY_FILTER"
)

// LookupFunc defines a function to look up a key from the environment.
type LookupFunc func(key string) (string, bool)
