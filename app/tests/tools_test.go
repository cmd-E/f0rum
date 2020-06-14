package testing

import (
	"testing"

	"github.com/IrvinIrvin/forum/app/tools"
)

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		str    string
		result bool
	}{
		{"[sdp;zof", false},
		{"", true},
		{" ", true},
		{"  sdsad  ", false},
		{" выалвызл ", false},
		{"      ", true},
	}
	for num, test := range tests {
		res := tools.IsEmpty(test.str)
		if res != test.result {
			t.Errorf("Test number %d. Str: [%s]. Got: %v, Expected: %v", num+1, test.str, res, test.result)
		}
	}
}

func TestGetID(t *testing.T) {
	tests := []struct {
		link string
		ID   string
	}{
		{"localhost:8080/posts/1", "1"},
		{"dsjcnksdln:dpaos/sdfv/000", "000"},
		{"fdgvfgdfv", "fdgvfgdfv"},
	}
	for num, test := range tests {
		res := tools.GetID(test.link)
		if res != test.ID {
			t.Errorf("Test num: %d. Test link: %s. Got [%v], expected: [%s]", num+1, test.link, res, test.ID)
		}
	}
}
