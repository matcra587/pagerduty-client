package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cloudwatchEnv() AlertEnvelope {
	return UnwrapAlert(map[string]any{
		"details": map[string]any{
			"custom_details": map[string]any{
				"AlarmName":        "Saffron-Octopus-RDS",
				"AlarmDescription": "CPU too high on RDS instance",
				"AWSAccountId":     "498849832712",
				"NewStateValue":    "ALARM",
				"NewStateReason":   "Threshold Crossed: 1 datapoint [2.1533] was >= threshold (0.0175).",
				"Region":           "US East (N. Virginia)",
				"AlarmArn":         "arn:aws:cloudwatch:us-east-1:498849832712:alarm:Saffron-Octopus-RDS",
				"Trigger": map[string]any{
					"MetricName":         "CPUUtilization",
					"Namespace":          "AWS/RDS",
					"Threshold":          0.0175,
					"ComparisonOperator": "GreaterThanOrEqualToThreshold",
				},
			},
		},
	})
}

func TestCloudWatch_MatchesPayload(t *testing.T) {
	s, ok := CloudWatch{}.Normalise(cloudwatchEnv())
	assert.True(t, ok)
	assert.Equal(t, "AWS CloudWatch", s.Source)
}

func TestCloudWatch_RejectsNonCloudWatch(t *testing.T) {
	_, ok := CloudWatch{}.Normalise(gcpEnv())
	assert.False(t, ok)
}

func TestCloudWatch_ExtractsFields(t *testing.T) {
	s, _ := CloudWatch{}.Normalise(cloudwatchEnv())
	fieldMap := make(map[string]string)
	for _, f := range s.Fields {
		fieldMap[f.Label] = f.Value
	}
	assert.Equal(t, "Saffron-Octopus-RDS", fieldMap["Alarm"])
	assert.Equal(t, "ALARM", fieldMap["State"])
	assert.Equal(t, "US East (N. Virginia)", fieldMap["Region"])
	assert.Equal(t, "CPUUtilization (AWS/RDS)", fieldMap["Metric"])
	assert.Contains(t, fieldMap["Reason"], "Threshold Crossed")
	require.Empty(t, s.Links)
}
