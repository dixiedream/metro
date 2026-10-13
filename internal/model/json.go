package model

import (
	"encoding/json"
	"fmt"
)

func (b BeatKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.String())
}

func (b *BeatKind) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		switch s {
		case "accent":
			*b = BeatAccent
		case "silent":
			*b = BeatSilent
		default:
			*b = BeatRegular
		}
		return nil
	}
	var n int
	if err := json.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("beat kind: %w", err)
	}
	*b = BeatKind(n)
	return nil
}
