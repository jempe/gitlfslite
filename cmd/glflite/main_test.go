package main

import (
	"bufio"
	"strings"
	"testing"
)

func TestAskYesNo(t *testing.T) {
	tests := map[string]bool{
		"yes\n":    true,
		"y\n":      true,
		"  YES \n": true,
		"Y":        true,
		"no\n":     false,
		"n\n":      false,
		"\n":       false,
		"":         false,
		"maybe\n":  false,
	}

	for input, want := range tests {
		if got := askYesNo(bufio.NewReader(strings.NewReader(input)), "Continue?"); got != want {
			t.Errorf("askYesNo(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestAskYesNoSharedReader(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("no\nyes\n"))

	if askYesNo(reader, "First?") {
		t.Error("first answer should be no")
	}

	if !askYesNo(reader, "Second?") {
		t.Error("second answer should be yes")
	}
}
