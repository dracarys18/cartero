package queue

import (
	"encoding/json"
	"fmt"

	"cartero/internal/types"
)

func marshalEnvelope(env types.Envelope) (map[string]any, error) {
	data, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal envelope: %w", err)
	}
	return map[string]any{"data": string(data)}, nil
}

func unmarshalEnvelope(fields map[string]any) (types.Envelope, error) {
	raw, ok := fields["data"]
	if !ok {
		return types.Envelope{}, fmt.Errorf("missing data field in message")
	}

	var data string
	switch v := raw.(type) {
	case string:
		data = v
	default:
		return types.Envelope{}, fmt.Errorf("unexpected data field type: %T", raw)
	}

	var env types.Envelope
	if err := json.Unmarshal([]byte(data), &env); err != nil {
		return types.Envelope{}, fmt.Errorf("failed to unmarshal envelope: %w", err)
	}

	return env, nil
}
