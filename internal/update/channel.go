package update

import "fmt"

// Channel selects which update track to follow.
type Channel int

const (
	// ChannelUnknown indicates no channel was configured. Resolves to ChannelStable.
	ChannelUnknown Channel = iota
	// ChannelStable tracks the latest tagged GitHub release.
	ChannelStable
	// ChannelDev tracks the latest commit on main.
	ChannelDev
)

// String returns the channel name.
func (c Channel) String() string {
	switch c {
	case ChannelStable:
		return "stable"
	case ChannelDev:
		return "dev"
	default:
		return "unknown"
	}
}

// Effective returns ChannelStable when the channel is ChannelUnknown,
// otherwise returns itself.
func (c Channel) Effective() Channel {
	if c == ChannelUnknown {
		return ChannelStable
	}
	return c
}

// ParseChannel converts a string to a Channel. An empty string
// returns ChannelStable. Unrecognised values return an error.
func ParseChannel(s string) (Channel, error) {
	switch s {
	case "", "stable":
		return ChannelStable, nil
	case "dev":
		return ChannelDev, nil
	default:
		return ChannelUnknown, fmt.Errorf("invalid channel %q: must be \"stable\" or \"dev\"", s)
	}
}
