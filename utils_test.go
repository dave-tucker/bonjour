package bonjour

import "testing"

func TestTrimDot(t *testing.T) {
	testString := "example.com."
	expected := "example.com"
	result := trimDot(testString)
	if result != expected {
		t.Fatalf("Expected:%s\nGot:%s\n", expected, result)
	}
}
