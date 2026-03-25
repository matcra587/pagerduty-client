package agent

// Metadata holds pagination information for agent envelope responses.
type Metadata struct {
	Total  int  `json:"total"`
	More   bool `json:"more"`
	Offset int  `json:"offset,omitempty"`
	Limit  int  `json:"limit,omitempty"`
}

// EnvelopeError describes a failed operation in the agent envelope.
type EnvelopeError struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// Envelope is the top-level JSON structure returned in agent mode.
type Envelope struct {
	OK      bool           `json:"success"`
	Command string         `json:"command"`
	Data    any            `json:"data,omitempty"`
	Meta    *Metadata      `json:"metadata,omitempty"`
	Hints   []string       `json:"hints,omitempty"`
	Err     *EnvelopeError `json:"error,omitempty"`
}

// Success builds an Envelope for a successful operation.
func Success(command string, data any, meta Metadata, hints []string) Envelope {
	return Envelope{
		OK:      true,
		Command: command,
		Data:    data,
		Meta:    &meta,
		Hints:   hints,
	}
}

// Error builds an Envelope for a failed operation.
func Error(command string, code int, message, suggestion string) Envelope {
	return Envelope{
		OK:      false,
		Command: command,
		Err: &EnvelopeError{
			Code:       code,
			Message:    message,
			Suggestion: suggestion,
		},
	}
}
