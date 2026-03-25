package integration

import "fmt"

// CloudWatch normalises AWS CloudWatch alarm payloads.
// Ref: https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/Notify_Users_Alarm_Changes.html
type CloudWatch struct{}

func (CloudWatch) Normalise(env AlertEnvelope) (Summary, bool) {
	cd := env.CustomDetails
	if cd == nil {
		return Summary{}, false
	}
	if _, has := cd["AlarmName"]; !has {
		return Summary{}, false
	}
	if _, has := cd["NewStateValue"]; !has {
		return Summary{}, false
	}
	if _, has := cd["Trigger"]; !has {
		return Summary{}, false
	}

	s := Summary{Source: "AWS CloudWatch"}

	if v, ok := cd["AlarmName"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Alarm", Value: v})
	}
	if v, ok := cd["AlarmDescription"].(string); ok && v != "" {
		s.Fields = append(s.Fields, Field{Label: "Description", Value: v})
	}
	if v, ok := cd["NewStateValue"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "State", Value: v})
	}
	if v, ok := cd["Region"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Region", Value: v})
	}
	if v, ok := cd["AWSAccountId"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Account", Value: v})
	}
	if trigger, ok := cd["Trigger"].(map[string]any); ok {
		metric, _ := trigger["MetricName"].(string)
		ns, _ := trigger["Namespace"].(string)
		if metric != "" {
			display := metric
			if ns != "" {
				display = fmt.Sprintf("%s (%s)", metric, ns)
			}
			s.Fields = append(s.Fields, Field{Label: "Metric", Value: display})
		}
		if thr, ok := trigger["Threshold"].(float64); ok {
			op, _ := trigger["ComparisonOperator"].(string)
			s.Fields = append(s.Fields, Field{Label: "Threshold", Value: fmt.Sprintf("%s %g", op, thr)})
		}
	}
	if v, ok := cd["NewStateReason"].(string); ok {
		s.Fields = append(s.Fields, Field{Label: "Reason", Value: v})
	}
	return s, true
}
