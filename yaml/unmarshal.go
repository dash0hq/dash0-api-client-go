package yaml

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	yamlv3 "go.yaml.in/yaml/v3"
	sigsyaml "sigs.k8s.io/yaml"
)

func Unmarshal(data []byte, v any) error {
	return sigsyaml.Unmarshal(quoteYAML11BoolKeys(data), v)
}

// UnmarshalStrict is [Unmarshal] for asset definitions: it fails when data has
// a field that v has no place for, instead of dropping it.
// The error names every unknown field by its full path, such as
// "spec.scope.datasets.restrction"; callers add the document name.
// Keys of maps, of "labels" and "annotations" objects, and of generated types
// with AdditionalProperties are free-form and always accepted.
// A field of a generated union type is accepted when one variant has a place
// for it.
// data may be YAML or JSON.
func UnmarshalStrict(data []byte, v any) error {
	jsonData, err := sigsyaml.YAMLToJSON(quoteYAML11BoolKeys(data))
	if err != nil {
		return err
	}
	var doc any
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		return err
	}
	switch unknown := unknownFields(reflect.TypeOf(v), doc, "", false); len(unknown) {
	case 0:
		return Unmarshal(data, v)
	case 1:
		return fmt.Errorf("unknown field %q", unknown[0])
	default:
		for i, path := range unknown {
			unknown[i] = strconv.Quote(path)
		}
		return fmt.Errorf("unknown fields %s", strings.Join(unknown, ", "))
	}
}

var jsonUnmarshalerType = reflect.TypeFor[json.Unmarshaler]()

// unknownFields returns the sorted paths of the keys in doc that
// encoding/json would drop when decoding doc into a value of type t.
// freeForm accepts any key at this level.
func unknownFields(t reflect.Type, doc any, path string, freeForm bool) []string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct && t.NumField() == 1 && t.Field(0).Name == "union" {
		return unionUnknownFields(t, doc, path)
	}

	var unknown []string
	switch doc := doc.(type) {
	case []any:
		if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
			return nil
		}
		for i, item := range doc {
			unknown = append(unknown, unknownFields(t.Elem(), item, path+"["+strconv.Itoa(i)+"]", false)...)
		}
	case map[string]any:
		extra, ok := extraKeysType(t)
		if !ok {
			return nil
		}
		for key, value := range doc {
			fieldPath := key
			if path != "" {
				fieldPath = path + "." + key
			}
			field, ok := jsonField(t, key)
			switch {
			case ok:
				name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
				unknown = append(unknown, unknownFields(field.Type, value, fieldPath, name == "labels" || name == "annotations")...)
			case extra != nil:
				unknown = append(unknown, unknownFields(extra, value, fieldPath, false)...)
			case !freeForm:
				unknown = append(unknown, fieldPath)
			}
		}
	}
	sort.Strings(unknown)
	return unknown
}

// extraKeysType reports whether unknownFields can check the keys of an object
// decoded into t, and the type of keys that t does not declare (nil when t
// rejects them).
func extraKeysType(t reflect.Type) (reflect.Type, bool) {
	switch {
	case t.Kind() == reflect.Map:
		return t.Elem(), true
	case t.Kind() != reflect.Struct:
		return nil, false
	}
	if f, ok := t.FieldByName("AdditionalProperties"); ok && f.Type.Kind() == reflect.Map {
		return f.Type.Elem(), true
	}
	// A custom decoder, such as time.Time's, decides its own keys.
	return nil, !reflect.PointerTo(t).Implements(jsonUnmarshalerType)
}

// jsonField returns the field of struct t that encoding/json decodes key into.
func jsonField(t reflect.Type, key string) (reflect.StructField, bool) {
	if t.Kind() != reflect.Struct {
		return reflect.StructField{}, false
	}
	for i := range t.NumField() {
		f := t.Field(i)
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "" {
			name = f.Name
		}
		if f.IsExported() && name != "-" && strings.EqualFold(name, key) {
			return f, true
		}
	}
	return reflect.StructField{}, false
}

// unionUnknownFields checks doc against each variant of an oapi-codegen union,
// found through its As<Variant> methods.
// doc is valid when one variant decodes it with no unknown field; otherwise
// the unknown fields of the closest variant are returned.
func unionUnknownFields(t reflect.Type, doc any, path string) []string {
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil
	}
	var closest []string
	found := false
	for i := range t.NumMethod() {
		m := t.Method(i)
		if !strings.HasPrefix(m.Name, "As") || m.Type.NumIn() != 1 || m.Type.NumOut() != 2 {
			continue
		}
		variant := m.Type.Out(0)
		if json.Unmarshal(raw, reflect.New(variant).Interface()) != nil {
			continue
		}
		unknown := unknownFields(variant, doc, path, false)
		if len(unknown) == 0 {
			return nil
		}
		if !found || len(unknown) < len(closest) {
			closest, found = unknown, true
		}
	}
	return closest
}

func ToJSON(data []byte) ([]byte, error) {
	var v any
	if err := yamlv3.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("error parsing YAML: %w", err)
	}
	out, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("error marshaling to JSON: %w", err)
	}
	return out, nil
}

var yaml11Bools = map[string]bool{
	"y": true, "Y": true, "yes": true, "Yes": true, "YES": true,
	"n": true, "N": true, "no": true, "No": true, "NO": true,
	"on": true, "On": true, "ON": true,
	"off": true, "Off": true, "OFF": true,
}

var yaml11BoolToken = regexp.MustCompile(`\b(?:[yYnN]|yes|Yes|YES|no|No|NO|on|On|ON|off|Off|OFF)\b`)

func quoteYAML11BoolKeys(data []byte) []byte {
	if !yaml11BoolToken.Match(data) || json.Valid(data) {
		return data
	}

	var doc yamlv3.Node
	if err := yamlv3.NewDecoder(bytes.NewReader(data)).Decode(&doc); err != nil {
		return data
	}

	var keys []*yamlv3.Node
	collectYAML11BoolKeys(&doc, &keys)

	starts := make([]int, 0, len(keys))
	ends := map[int]int{}
	for _, key := range keys {
		start, ok := byteOffset(data, key.Line, key.Column)
		if !ok || !bytes.HasPrefix(data[start:], []byte(key.Value)) {
			return data
		}
		starts = append(starts, start)
		ends[start] = start + len(key.Value)
	}
	if len(starts) == 0 {
		return data
	}
	sort.Sort(sort.Reverse(sort.IntSlice(starts)))

	out := bytes.Clone(data)
	for _, start := range starts {
		out = insertQuote(out, ends[start])
		out = insertQuote(out, start)
	}
	return out
}

func collectYAML11BoolKeys(node *yamlv3.Node, keys *[]*yamlv3.Node) {
	if node.Kind == yamlv3.MappingNode {
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind == yamlv3.ScalarNode && key.Style == 0 && key.Tag == "!!str" && yaml11Bools[key.Value] {
				*keys = append(*keys, key)
			}
		}
	}
	for _, child := range node.Content {
		collectYAML11BoolKeys(child, keys)
	}
}

func byteOffset(data []byte, line, column int) (int, bool) {
	offset := 0
	for l := 1; l < line; l++ {
		i := bytes.IndexByte(data[offset:], '\n')
		if i < 0 {
			return 0, false
		}
		offset += i + 1
	}
	for c := 1; c < column; c++ {
		if offset >= len(data) || data[offset] == '\n' {
			return 0, false
		}
		_, size := utf8.DecodeRune(data[offset:])
		offset += size
	}
	return offset, true
}

func insertQuote(data []byte, at int) []byte {
	return append(data[:at], append([]byte{'"'}, data[at:]...)...)
}
