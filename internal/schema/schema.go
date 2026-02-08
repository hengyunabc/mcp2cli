package schema

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/cobra"
)

type Kind string

const (
	KindString       Kind = "string"
	KindInteger      Kind = "integer"
	KindNumber       Kind = "number"
	KindBoolean      Kind = "boolean"
	KindObject       Kind = "object"
	KindArrayString  Kind = "array[string]"
	KindArrayInteger Kind = "array[integer]"
	KindArrayNumber  Kind = "array[number]"
	KindArrayBoolean Kind = "array[boolean]"
	KindArrayObject  Kind = "array[object]"
)

type Field struct {
	Name        string
	FlagName    string
	Description string
	Required    bool
	Kind        Kind
	Enum        []any
	Default     any
	HasDefault  bool
	Raw         map[string]any
}

type Spec struct {
	Fields []Field
}

func ParseToolInputSchema(tool mcp.Tool) (Spec, error) {
	requiredSet := map[string]bool{}
	for _, name := range tool.InputSchema.Required {
		requiredSet[name] = true
	}

	fieldNames := make([]string, 0, len(tool.InputSchema.Properties))
	for name := range tool.InputSchema.Properties {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	fields := make([]Field, 0, len(fieldNames))
	for _, name := range fieldNames {
		raw, err := toStringAnyMap(tool.InputSchema.Properties[name])
		if err != nil {
			return Spec{}, fmt.Errorf("invalid schema for property %q: %w", name, err)
		}

		kind, err := inferKind(raw)
		if err != nil {
			return Spec{}, fmt.Errorf("unsupported schema for property %q: %w", name, err)
		}

		field := Field{
			Name:        name,
			FlagName:    name,
			Description: asString(raw["description"]),
			Required:    requiredSet[name],
			Kind:        kind,
			Enum:        asAnySlice(raw["enum"]),
			Raw:         raw,
		}
		if defaultValue, ok := raw["default"]; ok {
			field.Default = defaultValue
			field.HasDefault = true
		}
		if field.Kind == KindObject || field.Kind == KindArrayObject {
			field.FlagName = field.Name + "-json"
		}
		fields = append(fields, field)
	}

	return Spec{Fields: fields}, nil
}

func RegisterFlags(cmd *cobra.Command, spec Spec) error {
	for _, field := range spec.Fields {
		usage := flagUsage(field)
		switch field.Kind {
		case KindString:
			cmd.Flags().String(field.FlagName, toDefaultString(field), usage)
		case KindInteger:
			cmd.Flags().Int64(field.FlagName, toDefaultInt64(field), usage)
		case KindNumber:
			cmd.Flags().Float64(field.FlagName, toDefaultFloat64(field), usage)
		case KindBoolean:
			cmd.Flags().Bool(field.FlagName, toDefaultBool(field), usage)
		case KindObject:
			cmd.Flags().String(field.FlagName, toDefaultJSONObject(field), usage)
		case KindArrayString, KindArrayInteger, KindArrayNumber, KindArrayBoolean, KindArrayObject:
			cmd.Flags().StringArray(field.FlagName, toDefaultStringArray(field), usage)
		default:
			return fmt.Errorf("unsupported flag kind for %q: %s", field.Name, field.Kind)
		}
	}
	return nil
}

func BuildArguments(cmd *cobra.Command, spec Spec) (map[string]any, error) {
	args := map[string]any{}
	for _, field := range spec.Fields {
		changed := cmd.Flags().Changed(field.FlagName)
		if !changed && !field.HasDefault {
			if field.Required {
				return nil, fmt.Errorf("missing required parameter --%s", field.FlagName)
			}
			continue
		}

		var (
			value any
			err   error
		)
		if changed {
			value, err = getChangedValue(cmd, field)
			if err != nil {
				return nil, err
			}
		} else {
			value = field.Default
		}

		if err := validateEnum(field, value); err != nil {
			return nil, err
		}

		args[field.Name] = value
	}
	return args, nil
}

func ParameterLines(spec Spec) []string {
	lines := make([]string, 0, len(spec.Fields))
	for _, field := range spec.Fields {
		parts := []string{
			fmt.Sprintf("--%s", field.FlagName),
			fmt.Sprintf("type=%s", field.Kind),
		}
		if field.Required {
			parts = append(parts, "required")
		}
		if field.HasDefault {
			parts = append(parts, "default="+valueAsText(field.Default))
		}
		if len(field.Enum) > 0 {
			enumValues := make([]string, 0, len(field.Enum))
			for _, item := range field.Enum {
				enumValues = append(enumValues, valueAsText(item))
			}
			parts = append(parts, "enum="+strings.Join(enumValues, "|"))
		}
		line := strings.Join(parts, ", ")
		if field.Description != "" {
			line += " - " + field.Description
		}
		lines = append(lines, line)
	}
	return lines
}

func InputSchemaMap(tool mcp.Tool) map[string]any {
	out := map[string]any{
		"type": tool.InputSchema.Type,
	}
	if len(tool.InputSchema.Defs) > 0 {
		out["$defs"] = tool.InputSchema.Defs
	}
	if len(tool.InputSchema.Properties) > 0 {
		out["properties"] = tool.InputSchema.Properties
	}
	if len(tool.InputSchema.Required) > 0 {
		out["required"] = tool.InputSchema.Required
	}
	return out
}

func OutputSchemaMap(tool mcp.Tool) (map[string]any, bool) {
	if tool.OutputSchema.Type == "" && len(tool.OutputSchema.Properties) == 0 && len(tool.OutputSchema.Defs) == 0 && len(tool.OutputSchema.Required) == 0 {
		return nil, false
	}
	out := map[string]any{}
	if tool.OutputSchema.Type != "" {
		out["type"] = tool.OutputSchema.Type
	}
	if len(tool.OutputSchema.Defs) > 0 {
		out["$defs"] = tool.OutputSchema.Defs
	}
	if len(tool.OutputSchema.Properties) > 0 {
		out["properties"] = tool.OutputSchema.Properties
	}
	if len(tool.OutputSchema.Required) > 0 {
		out["required"] = tool.OutputSchema.Required
	}
	return out, true
}

func getChangedValue(cmd *cobra.Command, field Field) (any, error) {
	switch field.Kind {
	case KindString:
		value, err := cmd.Flags().GetString(field.FlagName)
		if err != nil {
			return nil, fmt.Errorf("get --%s: %w", field.FlagName, err)
		}
		return value, nil
	case KindInteger:
		value, err := cmd.Flags().GetInt64(field.FlagName)
		if err != nil {
			return nil, fmt.Errorf("get --%s: %w", field.FlagName, err)
		}
		return value, nil
	case KindNumber:
		value, err := cmd.Flags().GetFloat64(field.FlagName)
		if err != nil {
			return nil, fmt.Errorf("get --%s: %w", field.FlagName, err)
		}
		return value, nil
	case KindBoolean:
		value, err := cmd.Flags().GetBool(field.FlagName)
		if err != nil {
			return nil, fmt.Errorf("get --%s: %w", field.FlagName, err)
		}
		return value, nil
	case KindObject:
		raw, err := cmd.Flags().GetString(field.FlagName)
		if err != nil {
			return nil, fmt.Errorf("get --%s: %w", field.FlagName, err)
		}
		return parseJSONValue("--"+field.FlagName, raw)
	case KindArrayString:
		value, err := cmd.Flags().GetStringArray(field.FlagName)
		if err != nil {
			return nil, fmt.Errorf("get --%s: %w", field.FlagName, err)
		}
		out := make([]any, 0, len(value))
		for _, item := range value {
			out = append(out, item)
		}
		return out, nil
	case KindArrayInteger:
		value, err := cmd.Flags().GetStringArray(field.FlagName)
		if err != nil {
			return nil, fmt.Errorf("get --%s: %w", field.FlagName, err)
		}
		out := make([]any, 0, len(value))
		for _, item := range value {
			parsed, err := strconv.ParseInt(item, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("--%s value %q is not a valid integer", field.FlagName, item)
			}
			out = append(out, parsed)
		}
		return out, nil
	case KindArrayNumber:
		value, err := cmd.Flags().GetStringArray(field.FlagName)
		if err != nil {
			return nil, fmt.Errorf("get --%s: %w", field.FlagName, err)
		}
		out := make([]any, 0, len(value))
		for _, item := range value {
			parsed, err := strconv.ParseFloat(item, 64)
			if err != nil {
				return nil, fmt.Errorf("--%s value %q is not a valid number", field.FlagName, item)
			}
			out = append(out, parsed)
		}
		return out, nil
	case KindArrayBoolean:
		value, err := cmd.Flags().GetStringArray(field.FlagName)
		if err != nil {
			return nil, fmt.Errorf("get --%s: %w", field.FlagName, err)
		}
		out := make([]any, 0, len(value))
		for _, item := range value {
			parsed, err := strconv.ParseBool(item)
			if err != nil {
				return nil, fmt.Errorf("--%s value %q is not a valid boolean", field.FlagName, item)
			}
			out = append(out, parsed)
		}
		return out, nil
	case KindArrayObject:
		values, err := cmd.Flags().GetStringArray(field.FlagName)
		if err != nil {
			return nil, fmt.Errorf("get --%s: %w", field.FlagName, err)
		}
		out := make([]any, 0, len(values))
		for _, raw := range values {
			item, err := parseJSONValue("--"+field.FlagName, raw)
			if err != nil {
				return nil, err
			}
			out = append(out, item)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported parameter kind %s for --%s", field.Kind, field.FlagName)
	}
}

func parseJSONValue(flagName string, raw string) (any, error) {
	var out any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("%s must be valid JSON: %w", flagName, err)
	}
	return out, nil
}

func validateEnum(field Field, value any) error {
	if len(field.Enum) == 0 {
		return nil
	}

	switch field.Kind {
	case KindArrayString, KindArrayInteger, KindArrayNumber, KindArrayBoolean, KindArrayObject:
		items, ok := value.([]any)
		if !ok {
			return fmt.Errorf("--%s enum validation failed: unexpected array value type", field.FlagName)
		}
		for _, item := range items {
			if !containsEnum(field.Enum, item) {
				return fmt.Errorf("--%s value %s is not in enum set", field.FlagName, valueAsText(item))
			}
		}
	default:
		if !containsEnum(field.Enum, value) {
			return fmt.Errorf("--%s value %s is not in enum set", field.FlagName, valueAsText(value))
		}
	}
	return nil
}

func containsEnum(enums []any, value any) bool {
	target := valueAsText(value)
	for _, item := range enums {
		if valueAsText(item) == target {
			return true
		}
	}
	return false
}

func inferKind(raw map[string]any) (Kind, error) {
	mainType := inferType(raw)
	switch mainType {
	case "string":
		return KindString, nil
	case "integer":
		return KindInteger, nil
	case "number":
		return KindNumber, nil
	case "boolean":
		return KindBoolean, nil
	case "object":
		return KindObject, nil
	case "array":
		itemsType := inferItemsType(raw["items"])
		switch itemsType {
		case "integer":
			return KindArrayInteger, nil
		case "number":
			return KindArrayNumber, nil
		case "boolean":
			return KindArrayBoolean, nil
		case "object":
			return KindArrayObject, nil
		case "", "string":
			return KindArrayString, nil
		default:
			return "", fmt.Errorf("unsupported array items type %q", itemsType)
		}
	default:
		return "", fmt.Errorf("unsupported type %q", mainType)
	}
}

func inferType(raw map[string]any) string {
	if typeValue, ok := raw["type"]; ok {
		if out := normalizeType(typeValue); out != "" {
			return out
		}
	}

	if anyOf, ok := raw["anyOf"].([]any); ok {
		for _, candidate := range anyOf {
			m, err := toStringAnyMap(candidate)
			if err != nil {
				continue
			}
			if out := inferType(m); out != "" && out != "null" {
				return out
			}
		}
	}
	if _, ok := raw["properties"]; ok {
		return "object"
	}
	if _, ok := raw["items"]; ok {
		return "array"
	}
	return ""
}

func normalizeType(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		for _, item := range typed {
			if candidate, ok := item.(string); ok && candidate != "null" {
				return candidate
			}
		}
	}
	return ""
}

func inferItemsType(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case map[string]any:
		return inferType(typed)
	case []any:
		for _, item := range typed {
			if m, ok := item.(map[string]any); ok {
				out := inferType(m)
				if out != "" && out != "null" {
					return out
				}
			}
		}
	}
	return ""
}

func flagUsage(field Field) string {
	parts := make([]string, 0, 3)
	if strings.TrimSpace(field.Description) != "" {
		parts = append(parts, field.Description)
	}
	if len(field.Enum) > 0 {
		values := make([]string, 0, len(field.Enum))
		for _, item := range field.Enum {
			values = append(values, valueAsText(item))
		}
		parts = append(parts, "enum="+strings.Join(values, "|"))
	}
	if field.Required {
		parts = append(parts, "required")
	}
	return strings.Join(parts, "; ")
}

func toStringAnyMap(value any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	if raw, ok := value.(map[string]any); ok {
		return raw, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	out, ok := value.(string)
	if !ok {
		return ""
	}
	return out
}

func asAnySlice(value any) []any {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []any:
		return typed
	case []string:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out
	default:
		return nil
	}
}

func toDefaultString(field Field) string {
	if !field.HasDefault {
		return ""
	}
	switch typed := field.Default.(type) {
	case string:
		return typed
	default:
		return valueAsText(typed)
	}
}

func toDefaultInt64(field Field) int64 {
	if !field.HasDefault {
		return 0
	}
	switch typed := field.Default.(type) {
	case float64:
		return int64(typed)
	case float32:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	case json.Number:
		out, err := typed.Int64()
		if err == nil {
			return out
		}
	case string:
		out, err := strconv.ParseInt(typed, 10, 64)
		if err == nil {
			return out
		}
	}
	return 0
}

func toDefaultFloat64(field Field) float64 {
	if !field.HasDefault {
		return 0
	}
	switch typed := field.Default.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int64:
		return float64(typed)
	case int:
		return float64(typed)
	case json.Number:
		out, err := typed.Float64()
		if err == nil {
			return out
		}
	case string:
		out, err := strconv.ParseFloat(typed, 64)
		if err == nil {
			return out
		}
	}
	return 0
}

func toDefaultBool(field Field) bool {
	if !field.HasDefault {
		return false
	}
	switch typed := field.Default.(type) {
	case bool:
		return typed
	case string:
		out, err := strconv.ParseBool(typed)
		if err == nil {
			return out
		}
	}
	return false
}

func toDefaultJSONObject(field Field) string {
	if !field.HasDefault {
		return ""
	}
	data, err := json.Marshal(field.Default)
	if err != nil {
		return ""
	}
	return string(data)
}

func toDefaultStringArray(field Field) []string {
	if !field.HasDefault {
		return nil
	}
	switch typed := field.Default.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, valueAsText(item))
		}
		return out
	case []string:
		out := make([]string, len(typed))
		copy(out, typed)
		return out
	default:
		return nil
	}
}

func valueAsText(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case int:
		return strconv.Itoa(typed)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 64)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case json.Number:
		return typed.String()
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(data)
	}
}
