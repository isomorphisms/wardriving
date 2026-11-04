package main

import "testing"

func TestShellIdentityToken(t *testing.T) {
	for _, v := range []struct {
		value string
		want  bool
	}{
		{"uid=2000(shell) gid=2000(shell)", true},
		{"uid=20000(app)", false}, {"uid=0(root)", false}, {"xuid=2000(shell)", false},
	} {
		if shellUID(v.value) != v.want {
			t.Fatalf("identity %q", v.value)
		}
	}
}
