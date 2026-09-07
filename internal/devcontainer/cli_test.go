package devcontainer

import "testing"

func TestParseResult(t *testing.T) {
	out := "some noise\n{\"outcome\":\"success\",\"containerId\":\"abc\",\"remoteUser\":\"dev\"}\n"
	r, err := ParseResult([]byte(out))
	if err != nil || r.Outcome != "success" || r.ContainerID != "abc" {
		t.Errorf("r = %+v, err = %v", r, err)
	}
	r, err = ParseResult([]byte(`{"outcome":"error","message":"Dev container failed","description":"boom"}`))
	if err != nil || r.Outcome != "error" || r.Description != "boom" {
		t.Errorf("r = %+v, err = %v", r, err)
	}
	if _, err := ParseResult(nil); err == nil {
		t.Error("пустой вывод должен быть ошибкой")
	}
}
