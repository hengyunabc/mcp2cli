package render

import (
	"encoding/json"
	"fmt"
	"io"
)

func JSON(w io.Writer, value any, pretty bool) error {
	var (
		data []byte
		err  error
	)
	if pretty {
		data, err = json.MarshalIndent(value, "", "  ")
	} else {
		data, err = json.Marshal(value)
	}
	if err != nil {
		return fmt.Errorf("marshal JSON output: %w", err)
	}
	if _, err := fmt.Fprintln(w, string(data)); err != nil {
		return fmt.Errorf("write JSON output: %w", err)
	}
	return nil
}

func PrettyJSONString(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "<invalid-json>"
	}
	return string(data)
}
