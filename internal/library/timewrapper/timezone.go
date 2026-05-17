package timewrapper

import "time"

var jst = time.FixedZone("JST", 9*60*60)

// JST returns the shared JST location used by the application.
func JST() *time.Location {
	return jst
}

// InJST converts a timestamp to JST while preserving zero values.
func InJST(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}

	return value.In(jst)
}
