package integration

import (
	"maps"
	"slices"
)

// Generic is the fallback normaliser for unrecognised integration payloads.
// It extracts custom_details as flat key-value pairs. Normalise always
// returns ok=false so Detect skips it during the detection loop; it is
// called explicitly as the fallback.
type Generic struct{}

func (Generic) Normalise(env AlertEnvelope) (Summary, bool) {
	if env.CustomDetails == nil {
		return Summary{Source: "Unknown"}, false
	}

	var fields []Field
	for _, k := range slices.Sorted(maps.Keys(env.CustomDetails)) {
		fields = append(fields, Field{Label: k, Value: FormatValue(env.CustomDetails[k])})
	}
	return Summary{Source: "Unknown", Fields: fields}, false
}
