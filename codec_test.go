package bamgoo

import (
	"reflect"
	"testing"

	. "github.com/bamgoo/base"
)

func TestCodecJSON(t *testing.T) {
	in := map[string]Any{"a": 1, "b": "x"}
	bts, err := Marshal(JSON, in)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	var out map[string]Any
	if err := Unmarshal(JSON, bts, &out); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if out["b"] != "x" {
		t.Fatalf("unexpected value: %v", out)
	}
}

func TestCodecTextRoundtrip(t *testing.T) {
	in := "hello world"
	enc, err := Encrypt(TEXT, in)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	dec, err := Decrypt(TEXT, enc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(dec.([]byte)) != in {
		t.Fatalf("roundtrip mismatch: %v", dec)
	}
}

func TestCodecDigitsRoundtrip(t *testing.T) {
	in := []int64{1, 23, 456}
	enc, err := Encrypt(DIGITS, in)
	if err != nil {
		t.Fatalf("encrypt digits: %v", err)
	}
	dec, err := Decrypt(DIGITS, enc)
	if err != nil {
		t.Fatalf("decrypt digits: %v", err)
	}
	out, ok := dec.([]int64)
	if !ok {
		t.Fatalf("unexpected type: %T", dec)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("roundtrip mismatch: %v vs %v", in, out)
	}
}

func TestCodecTextsRoundtrip(t *testing.T) {
	in := []string{"a", "b", "c"}
	enc, err := Encrypt(TEXTS, in)
	if err != nil {
		t.Fatalf("encrypt texts: %v", err)
	}
	dec, err := Decrypt(TEXTS, enc)
	if err != nil {
		t.Fatalf("decrypt texts: %v", err)
	}
	out, ok := dec.([]string)
	if !ok {
		t.Fatalf("unexpected type: %T", dec)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("roundtrip mismatch: %v vs %v", in, out)
	}
}
