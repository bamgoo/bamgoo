package bamgoo

import (
	"testing"
	"time"

	. "github.com/bamgoo/base"
)

func TestMappingRequired(t *testing.T) {
	cfg := Vars{
		"name": {Required: true, Nullable: false, Name: "name"},
	}
	out := Map{}
	res := basic.Mapping(cfg, Map{}, out, false, false)
	if res == nil || res.OK() {
		t.Fatalf("expected error for required field")
	}
}

func TestMappingDefaultAndType(t *testing.T) {
	basic.RegisterType("int", Type{
		Check: func(v Any, _ Var) bool {
			_, ok := v.(int64)
			return ok
		},
		Convert: func(v Any, _ Var) Any { return v },
	})

	cfg := Vars{
		"age": {Type: "int", Default: 18},
	}
	out := Map{}
	res := basic.Mapping(cfg, Map{}, out, false, false)
	if res != nil && res.Fail() {
		t.Fatalf("unexpected error: %v", res)
	}
	if out["age"].(int64) != 18 {
		t.Fatalf("default not applied: %v", out)
	}
}

func TestMappingChildren(t *testing.T) {
	cfg := Vars{
		"user": {
			Children: Vars{
				"id":   {Required: true},
				"name": {},
			},
		},
	}
	in := Map{"user": Map{"id": 1, "name": "tom"}}
	out := Map{}
	res := basic.Mapping(cfg, in, out, false, false)
	if res != nil && res.Fail() {
		t.Fatalf("unexpected error: %v", res)
	}
	user := out["user"].(Map)
	if user["name"] != "tom" {
		t.Fatalf("nested mapping failed: %v", user)
	}
}

func TestMappingEncodeDecode(t *testing.T) {
	secret, err := Encrypt(TEXT, "hello")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	cfg := Vars{
		"msg": {Decode: TEXT, Encode: TEXT},
	}
	in := Map{"msg": secret}
	out := Map{}
	res := basic.Mapping(cfg, in, out, false, false, time.UTC)
	if res != nil && res.Fail() {
		t.Fatalf("unexpected error: %v", res)
	}
	if out["msg"] != "hello" {
		t.Fatalf("decode failed: %v", out)
	}
}
