package api

import "github.com/dop251/goja"

// Browser is the public interface of a CDP browser.
type Browser interface {
	Close()
	Contexts() []BrowserContext
	IsConnected() bool
	SetupContext(opts goja.Value) (BrowserContext, error)
	NewPage() (Page, error)
	On(string) (bool, error)
	UserAgent() string
	Version() string
}
