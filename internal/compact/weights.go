package compact

import (
	"os"
	"strconv"
)

// Resource identifies a PagerDuty resource type for column budget allocation.
type Resource string

const (
	// ResourceIncident is an incident.
	ResourceIncident Resource = "incident"
	// ResourceAlert is an alert within an incident.
	ResourceAlert Resource = "alert"
	// ResourceService is a PagerDuty service.
	ResourceService Resource = "service"
	// ResourceUser is a PagerDuty user.
	ResourceUser Resource = "user"
	// ResourceTeam is a PagerDuty team.
	ResourceTeam Resource = "team"
	// ResourceSchedule is an on-call schedule.
	ResourceSchedule Resource = "schedule"
	// ResourceOnCall is an on-call entry.
	ResourceOnCall Resource = "oncall"
	// ResourceLogEntry is an incident timeline entry.
	ResourceLogEntry Resource = "log_entry"
	// ResourceNote is an incident note.
	ResourceNote Resource = "note"
	// ResourceEscalationPolicy is an escalation policy.
	ResourceEscalationPolicy Resource = "escalation_policy"
	// ResourceMaintenanceWindow is a maintenance window.
	ResourceMaintenanceWindow Resource = "maintenance_window"
	// ResourceNone is the default when no resource context applies.
	ResourceNone Resource = ""
)

// ResourceWeights defines per-field importance weights and a token
// budget for a resource type. Higher weights mean the field is more
// likely to survive compaction.
type ResourceWeights struct {
	Fields        map[string]float64
	DefaultWeight float64
	Budget        int
}

// ForField returns the weight for field, falling back to DefaultWeight
// if the field is not listed.
func (w ResourceWeights) ForField(field string) float64 {
	if v, ok := w.Fields[field]; ok {
		return v
	}

	return w.DefaultWeight
}

// resourceWeights holds the per-resource weight profiles.
var resourceWeights = map[Resource]ResourceWeights{
	ResourceIncident: {
		Budget:        400,
		DefaultWeight: 0.1,
		Fields: map[string]float64{
			"id":                    1.0,
			"title":                 1.0,
			"status":                1.0,
			"urgency":               1.0,
			"priority":              1.0,
			"integration":           1.0,
			"service":               0.9,
			"assignments":           0.9,
			"incident_number":       0.8,
			"created_at":            0.8,
			"escalation_policy":     0.7,
			"teams":                 0.7,
			"acknowledgements":      0.7,
			"body":                  0.8,
			"alert_counts":          0.6,
			"last_status_change_at": 0.5,
			"html_url":              0.4,
			"description":           0.3,
			"incident_key":          0.3,
		},
	},
	ResourceAlert: {
		Budget:        150,
		DefaultWeight: 0.1,
		Fields: map[string]float64{
			"id":          1.0,
			"status":      1.0,
			"severity":    0.9,
			"service":     0.8,
			"incident":    0.8,
			"body":        0.7,
			"summary":     0.7,
			"created_at":  0.7,
			"suppressed":  0.6,
			"alert_key":   0.5,
			"integration": 0.4,
			"html_url":    0.3,
		},
	},
	ResourceService: {
		Budget:        120,
		DefaultWeight: 0.1,
		Fields: map[string]float64{
			"id":                      1.0,
			"name":                    1.0,
			"status":                  0.9,
			"escalation_policy":       0.8,
			"teams":                   0.7,
			"last_incident_timestamp": 0.6,
			"description":             0.5,
			"html_url":                0.3,
		},
	},
	ResourceUser: {
		Budget:        100,
		DefaultWeight: 0.1,
		Fields: map[string]float64{
			"id":        1.0,
			"name":      1.0,
			"email":     0.8,
			"teams":     0.7,
			"role":      0.5,
			"time_zone": 0.4,
			"html_url":  0.3,
		},
	},
	ResourceTeam: {
		Budget:        60,
		DefaultWeight: 0.1,
		Fields: map[string]float64{
			"id":          1.0,
			"name":        1.0,
			"parent":      0.5,
			"description": 0.4,
			"html_url":    0.3,
		},
	},
	ResourceSchedule: {
		Budget:        150,
		DefaultWeight: 0.1,
		Fields: map[string]float64{
			"id":                  1.0,
			"name":                1.0,
			"users":               0.7,
			"escalation_policies": 0.7,
			"time_zone":           0.6,
			"description":         0.3,
			"html_url":            0.3,
		},
	},
	ResourceOnCall: {
		Budget:        100,
		DefaultWeight: 0.1,
		Fields: map[string]float64{
			"user":              1.0,
			"escalation_policy": 0.9,
			"escalation_level":  0.9,
			"schedule":          0.8,
			"start":             0.7,
			"end":               0.7,
		},
	},
	ResourceLogEntry: {
		Budget:        200,
		DefaultWeight: 0.1,
		Fields: map[string]float64{
			"created_at":    1.0,
			"agent":         0.8,
			"channel":       0.7,
			"event_details": 0.6,
			"contexts":      0.4,
			"incident":      0.2,
			"service":       0.2,
			"user":          0.2,
		},
	},
	ResourceNote: {
		Budget:        80,
		DefaultWeight: 0.1,
		Fields: map[string]float64{
			"id":         1.0,
			"content":    1.0,
			"user":       0.8,
			"created_at": 0.7,
		},
	},
	ResourceEscalationPolicy: {
		Budget:        120,
		DefaultWeight: 0.1,
		Fields: map[string]float64{
			"id":               1.0,
			"name":             1.0,
			"escalation_rules": 0.9,
			"num_loops":        0.8,
			"teams":            0.7,
			"description":      0.5,
			"html_url":         0.3,
		},
	},
}

// WeightsForResource looks up the weight profile for a resource type.
// It returns false if no profile is defined.
func WeightsForResource(r Resource) (ResourceWeights, bool) {
	w, ok := resourceWeights[r]
	return w, ok
}

// budgetOverride reads PDC_AGENT_BUDGET from the environment and
// returns its value as an int. Returns 0 if unset, invalid or negative.
func budgetOverride() int {
	s := os.Getenv("PDC_AGENT_BUDGET")
	if s == "" {
		return 0
	}

	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0
	}

	return n
}
