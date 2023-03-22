package js

import (
	_ "embed"
)

// WebVitalIIFEScript was downloaded from
// https://unpkg.com/web-vitals@3/dist/web-vitals.iife.js.
// Repo: https://github.com/GoogleChrome/web-vitals
//
//go:embed web-vitals-iife.js
var WebVitalIIFEScript string

// WebVitalScript is used to init WebVitalIIFEScript.
//
//go:embed webvital.js
var WebVitalScript string
