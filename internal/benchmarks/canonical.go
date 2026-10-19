package benchmarks

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

func digest(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }

// parseJSON rejects duplicate keys (including escaped aliases), trailing values,
// invalid Unicode and non-I-JSON numbers before typed decoding can discard them.
func parseJSON(b []byte) (any, error) {
	if !utf8.Valid(b) {
		return nil, fmt.Errorf("invalid UTF-8")
	}
	// encoding/json otherwise silently replaces unpaired surrogate escapes.
	for i := 0; i < len(b); i++ {
		if b[i] != '\\' {
			continue
		}
		i++
		if i >= len(b) || b[i] != 'u' {
			continue
		}
		if i+4 >= len(b) {
			return nil, fmt.Errorf("unicode escape")
		}
		n, e := strconv.ParseUint(string(b[i+1:i+5]), 16, 16)
		if e != nil {
			return nil, e
		}
		i += 4
		if n >= 0xd800 && n <= 0xdbff {
			if i+6 >= len(b) || string(b[i+1:i+3]) != "\\u" {
				return nil, fmt.Errorf("unpaired surrogate")
			}
			m, e := strconv.ParseUint(string(b[i+3:i+7]), 16, 16)
			if e != nil || m < 0xdc00 || m > 0xdfff {
				return nil, fmt.Errorf("unpaired surrogate")
			}
			i += 6
		} else if n >= 0xdc00 && n <= 0xdfff {
			return nil, fmt.Errorf("unpaired surrogate")
		}
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	var value func() (any, error)
	value = func() (any, error) {
		t, e := d.Token()
		if e != nil {
			return nil, e
		}
		switch t {
		case json.Delim('{'):
			m := map[string]any{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return nil, e
				}
				key, ok := k.(string)
				if !ok {
					return nil, fmt.Errorf("object key")
				}
				if _, ok = m[key]; ok {
					return nil, fmt.Errorf("duplicate key")
				}
				v, e := value()
				if e != nil {
					return nil, e
				}
				m[key] = v
			}
			_, e = d.Token()
			return m, e
		case json.Delim('['):
			a := []any{}
			for d.More() {
				v, e := value()
				if e != nil {
					return nil, e
				}
				a = append(a, v)
			}
			_, e = d.Token()
			return a, e
		default:
			return t, nil
		}
	}
	v, e := value()
	if e != nil {
		return nil, e
	}
	if _, e = d.Token(); e != io.EOF {
		return nil, fmt.Errorf("trailing JSON")
	}
	return v, nil
}

// canonical implements RFC 8785: UTF-16 key ordering, JSON string escaping
// without HTML escaping, and ECMAScript finite double representation.
func canonical(v any) ([]byte, error) {
	var b bytes.Buffer
	var put func(any) error
	quote := func(s string) {
		b.WriteByte('"')
		for _, r := range s {
			switch r {
			case '"', '\\':
				b.WriteByte('\\')
				b.WriteRune(r)
			case '\b':
				b.WriteString(`\b`)
			case '\t':
				b.WriteString(`\t`)
			case '\n':
				b.WriteString(`\n`)
			case '\f':
				b.WriteString(`\f`)
			case '\r':
				b.WriteString(`\r`)
			default:
				if r < 0x20 {
					fmt.Fprintf(&b, `\u%04x`, r)
				} else {
					b.WriteRune(r)
				}
			}
		}
		b.WriteByte('"')
	}
	put = func(v any) error {
		switch x := v.(type) {
		case nil:
			b.WriteString("null")
		case bool:
			b.WriteString(strconv.FormatBool(x))
		case string:
			quote(x)
		case json.Number:
			f, e := strconv.ParseFloat(string(x), 64)
			if e != nil || math.IsNaN(f) || math.IsInf(f, 0) {
				return fmt.Errorf("non-finite number")
			}
			if f == 0 {
				b.WriteByte('0')
				break
			}
			format := byte('e')
			if math.Abs(f) >= 1e-6 && math.Abs(f) < 1e21 {
				format = 'f'
			}
			s := strconv.FormatFloat(f, format, -1, 64)
			if p := strings.Index(s, "e"); p >= 0 {
				head, tail := s[:p+2], s[p+2:]
				tail = strings.TrimLeft(tail, "0")
				if tail == "" {
					tail = "0"
				}
				s = head + tail
			}
			b.WriteString(s)
		case []any:
			b.WriteByte('[')
			for i, v := range x {
				if i > 0 {
					b.WriteByte(',')
				}
				if e := put(v); e != nil {
					return e
				}
			}
			b.WriteByte(']')
		case map[string]any:
			keys := make([]string, 0, len(x))
			for k := range x {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool {
				a, c := utf16.Encode([]rune(keys[i])), utf16.Encode([]rune(keys[j]))
				for n := 0; n < len(a) && n < len(c); n++ {
					if a[n] != c[n] {
						return a[n] < c[n]
					}
				}
				return len(a) < len(c)
			})
			b.WriteByte('{')
			for i, k := range keys {
				if i > 0 {
					b.WriteByte(',')
				}
				quote(k)
				b.WriteByte(':')
				if e := put(x[k]); e != nil {
					return e
				}
			}
			b.WriteByte('}')
		default:
			return fmt.Errorf("unsupported canonical value %T", v)
		}
		return nil
	}
	if e := put(v); e != nil {
		return nil, e
	}
	return b.Bytes(), nil
}

// CanonicalJSON returns stored/report JSON: JCS plus one LF.
func CanonicalJSON(v any) ([]byte, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	v, e = parseJSON(b)
	if e != nil {
		return nil, e
	}
	b, e = canonical(v)
	return append(b, '\n'), e
}
func structuredHash(v any) string {
	b, e := CanonicalJSON(v)
	if e != nil {
		return ""
	}
	return digest(bytes.TrimSuffix(b, []byte{'\n'}))
}

func decodeStrict(b []byte, dst any) (any, error) {
	v, e := parseJSON(b)
	if e != nil {
		return nil, e
	}
	if e = shape(v, reflect.TypeOf(dst).Elem()); e != nil {
		return nil, e
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(dst); e != nil {
		return nil, e
	}
	return v, nil
}

// Require all declared fields and reject null collections/objects. Pointer
// fields are explicitly nullable; map entries are the spec's open maps.
func shape(v any, t reflect.Type) error {
	if t.Kind() == reflect.Pointer {
		if v == nil {
			return nil
		}
		return shape(v, t.Elem())
	}
	if v == nil {
		return fmt.Errorf("unexpected null")
	}
	switch t.Kind() {
	case reflect.Struct:
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("expected object")
		}
		if len(m) != t.NumField() {
			return fmt.Errorf("missing or unknown field")
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			key := f.Tag.Get("json")
			x, ok := m[key]
			if !ok {
				return fmt.Errorf("missing %s", key)
			}
			if e := shape(x, f.Type); e != nil {
				return fmt.Errorf("%s: %w", key, e)
			}
		}
	case reflect.Slice:
		a, ok := v.([]any)
		if !ok {
			return fmt.Errorf("expected array")
		}
		for _, x := range a {
			if e := shape(x, t.Elem()); e != nil {
				return e
			}
		}
	case reflect.Map:
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("expected map")
		}
		for _, x := range m {
			if e := shape(x, t.Elem()); e != nil {
				return e
			}
		}
	}
	return nil
}
