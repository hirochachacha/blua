package strconv

import "testing"

var unquoteCases = []struct {
	input  string
	output string
}{
	{`"test"`, "test"},
	{`'test'`, "test"},
	{`"test\nart"`, "test\nart"},
	{`"\27Lua"`, "\x1bLua"},
	{`"\027Lua"`, "\x1bLua"},
	{`"\x19\x93\r\n\x1a\n"`, "\x19\x93\r\n\x1a\n"},
	{`"\"test\""`, `"test"`},
	{`"\'test\'"`, `'test'`},
	{`"'test'"`, `'test'`},
	{`"\u{0}\u{00000000}\x00\0"`, "\x00\x00\x00\x00"},
}

func TestUnquote(t *testing.T) {
	for i, test := range unquoteCases {
		got, err := Unquote(test.input)
		if err != nil {
			t.Fatal(err)
		}
		if got != test.output {
			t.Errorf("%d: got %v, want %v", i+1, got, test.output)
		}
	}
}

var escapeCases = []struct {
	input  string
	output string
}{
	{"test", "test"},
	{"test\x1bart", "test\\027art"},
	{"test\x10art", "test\\016art"},
	{"test\xfeart", "test\\254art"},
	{"test\xffart", "test\\255art"},
}

func TestEscape(t *testing.T) {
	for i, test := range escapeCases {
		got := Escape(test.input)
		if got != test.output {
			t.Errorf("%d: got %v, want %v", i+1, got, test.output)
		}
	}
}
