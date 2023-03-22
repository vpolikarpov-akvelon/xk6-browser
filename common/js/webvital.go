package js

import (
	_ "embed"
)

// WebVitalScript is used to calculate the web vitals.
//
//go:embed webvital.js
var WebVitalScript string
