package benchmarks

import (
	"testing"
)

// JCS, rather than Go's HTML/string/float encoding, defines the locks.
func TestCanonicalJSON(t *testing.T) {
	got, err := CanonicalJSON(map[string]any{"z": "<>&\u2028\u2029", "n": 1e-7, "small": 1e-6, "big": 1e21, "\ue000": 1, "😀": 2})
	want := "{\"big\":1e+21,\"n\":1e-7,\"small\":0.000001,\"z\":\"<>&\u2028\u2029\",\"😀\":2,\"\ue000\":1}\n"
	if err != nil || string(got) != want {
		t.Fatalf("got %q (%v), want %q", got, err, want)
	}
	for _, b := range []string{`{"x":1,"\u0078":2}`, `{"x":"\uD800"}`, `{"x":1} {}`, `{"x":1e999}`} {
		v, e := parseJSON([]byte(b))
		if e == nil {
			_, e = canonical(v)
		}
		if e == nil {
			t.Fatalf("accepted non-I-JSON: %s", b)
		}
	}
}
