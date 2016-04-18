package bonjour

import "testing"

var testService = NewServiceRecord("My Service", "_http._tcp.", "local")

func TestServiceName(t *testing.T) {
	expected := "_http._tcp.local."
	result := testService.ServiceName()
	if result != expected {
		t.Fatalf("Expected:%s\nGot:%s", expected, result)

	}
}

func TestServiceInstanceName(t *testing.T) {
	expected := "My Service._http._tcp.local."
	result := testService.ServiceInstanceName()
	if result != expected {
		t.Fatalf("Expected:%s\nGot:%s", expected, result)

	}
}
