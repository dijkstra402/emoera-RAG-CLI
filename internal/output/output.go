package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

func Write(writer io.Writer, mode string, value any) error {
	if mode == "json" || mode == "jsonl" {
		encoder := json.NewEncoder(writer)
		encoder.SetEscapeHTML(false)
		if mode == "json" {
			encoder.SetIndent("", "  ")
		}
		return encoder.Encode(value)
	}
	return writeTable(writer, value)
}

func writeTable(writer io.Writer, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var generic any
	if err := json.Unmarshal(encoded, &generic); err != nil {
		return err
	}
	switch data := generic.(type) {
	case map[string]any:
		keys := make([]string, 0, len(data))
		for key := range data {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(writer, "%-18s %s\n", strings.ToUpper(key), scalar(data[key]))
		}
	case []any:
		for index, item := range data {
			fmt.Fprintf(writer, "%d\t%s\n", index+1, scalar(item))
		}
	default:
		fmt.Fprintln(writer, scalar(data))
	}
	return nil
}

func scalar(value any) string {
	switch typed := value.(type) {
	case nil:
		return "-"
	case string:
		return typed
	case float64, bool:
		return fmt.Sprint(typed)
	default:
		encoded, _ := json.Marshal(typed)
		return string(encoded)
	}
}
