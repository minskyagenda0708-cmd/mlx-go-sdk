package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
)

func emit(rt *Runtime, value any) error {
	return emitFormatted(
		os.Stdout,
		rt.Config.Output.Format,
		rt.Config.Output.Pretty,
		value,
	)
}

func emitWithGlobal(global globalOptions, value any) error {
	cfg := DefaultConfig()
	if override := strings.ToLower(strings.TrimSpace(global.Output)); override != "" {
		cfg.Output.Format = override
		cfg = cfg.Normalize()
		if err := cfg.Validate(); err != nil {
			return err
		}
	}

	return emitFormatted(
		os.Stdout,
		cfg.Output.Format,
		cfg.Output.Pretty,
		value,
	)
}

func emitFormatted(w io.Writer, format string, pretty bool, value any) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", outputFormatTable:
		return renderTable(w, value)
	case outputFormatJSON:
		return renderJSON(w, value, pretty)
	case outputFormatYAML:
		return renderYAML(w, value)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func renderJSON(w io.Writer, value any, pretty bool) error {
	var (
		body []byte
		err  error
	)

	if pretty {
		body, err = json.MarshalIndent(value, "", "  ")
	} else {
		body, err = json.Marshal(value)
	}
	if err != nil {
		return err
	}

	if _, err := w.Write(append(body, '\n')); err != nil {
		return err
	}

	return nil
}

func renderYAML(w io.Writer, value any) error {
	generic, err := normalizeValue(value)
	if err != nil {
		return err
	}
	if err := writeYAMLNode(w, generic, 0, true); err != nil {
		return err
	}

	_, err = io.WriteString(w, "\n")
	return err
}

func renderTable(w io.Writer, value any) error {
	generic, err := normalizeValue(value)
	if err != nil {
		return err
	}

	switch v := generic.(type) {
	case nil:
		_, err := fmt.Fprintln(w, "(empty)")
		return err
	case []any:
		return renderTableSlice(w, v)
	case map[string]any:
		return renderTableMap(w, v)
	default:
		_, err := fmt.Fprintln(w, formatScalar(v))
		return err
	}
}

func renderTableSlice(w io.Writer, rows []any) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "(empty)")
		return err
	}

	allMaps := true
	for _, row := range rows {
		if _, ok := row.(map[string]any); !ok {
			allMaps = false
			break
		}
	}
	if !allMaps {
		for _, row := range rows {
			if _, err := fmt.Fprintf(w, "- %s\n", formatCell(row)); err != nil {
				return err
			}
		}
		return nil
	}

	keys := make(map[string]struct{})
	for _, row := range rows {
		for key := range row.(map[string]any) {
			keys[key] = struct{}{}
		}
	}

	header := make([]string, 0, len(keys))
	for key := range keys {
		header = append(header, key)
	}
	sort.Strings(header)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, strings.Join(header, "\t")); err != nil {
		return err
	}
	for _, row := range rows {
		m := row.(map[string]any)
		cells := make([]string, 0, len(header))
		for _, key := range header {
			cells = append(cells, formatCell(m[key]))
		}
		if _, err := fmt.Fprintln(tw, strings.Join(cells, "\t")); err != nil {
			return err
		}
	}

	return tw.Flush()
}

func renderTableMap(w io.Writer, values map[string]any) error {
	if len(values) == 0 {
		_, err := fmt.Fprintln(w, "(empty)")
		return err
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, key := range keys {
		if _, err := fmt.Fprintf(
			tw,
			"%s\t%s\n",
			key,
			formatCell(values[key]),
		); err != nil {
			return err
		}
	}

	return tw.Flush()
}

func normalizeValue(value any) (any, error) {
	if value == nil {
		return nil, nil
	}

	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var generic any
	if err := json.Unmarshal(body, &generic); err != nil {
		return nil, err
	}

	return generic, nil
}

func writeYAMLNode(w io.Writer, node any, indent int, topLevel bool) error {
	prefix := strings.Repeat("  ", indent)

	switch v := node.(type) {
	case nil:
		_, err := io.WriteString(w, prefix+"null")
		return err
	case map[string]any:
		if len(v) == 0 {
			_, err := io.WriteString(w, prefix+"{}")
			return err
		}

		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for i, key := range keys {
			if !topLevel || i > 0 {
				if _, err := io.WriteString(w, "\n"); err != nil {
					return err
				}
			}
			if isScalarYAML(v[key]) {
				if _, err := fmt.Fprintf(
					w,
					"%s%s: %s",
					prefix,
					key,
					formatYAMLScalar(v[key]),
				); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprintf(w, "%s%s:", prefix, key); err != nil {
				return err
			}
			if err := writeYAMLNode(w, v[key], indent+1, false); err != nil {
				return err
			}
		}

		return nil
	case []any:
		if len(v) == 0 {
			_, err := io.WriteString(w, prefix+"[]")
			return err
		}

		for i, item := range v {
			if !topLevel || i > 0 {
				if _, err := io.WriteString(w, "\n"); err != nil {
					return err
				}
			}
			if isScalarYAML(item) {
				if _, err := fmt.Fprintf(
					w,
					"%s- %s",
					prefix,
					formatYAMLScalar(item),
				); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprintf(w, "%s-", prefix); err != nil {
				return err
			}
			if err := writeYAMLNode(w, item, indent+1, false); err != nil {
				return err
			}
		}

		return nil
	default:
		_, err := io.WriteString(w, prefix+formatYAMLScalar(v))
		return err
	}
}

func isScalarYAML(v any) bool {
	switch v.(type) {
	case nil, string, bool, float64, int, int64, uint64:
		return true
	default:
		return false
	}
}

func formatYAMLScalar(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		body, _ := json.Marshal(x)
		return string(body)
	default:
		return fmt.Sprint(x)
	}
}

func formatCell(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case map[string]any, []any:
		body, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(body)
	default:
		return fmt.Sprint(v)
	}
}

func formatScalar(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}
