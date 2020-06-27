package sora

import "testing"

func TestCleanupSDP(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{
			in:  "c=IN IP4 0.0.0.0\r\nb=TIAS:500000\r\nb=AS:500\r\n",
			out: "c=IN IP4 0.0.0.0\r\nb=AS:500\r\n",
		},
		{
			in:  "c=IN IP4 0.0.0.0\r\nb=TIAS:10\r\nb=AS:500\r\n",
			out: "c=IN IP4 0.0.0.0\r\nb=AS:500\r\n",
		},
	}

	for _, c := range cases {
		ret := cleanupSDP(c.in)
		if ret != c.out {
			t.Errorf("expected: %s, but got %s", c.out, ret)
		}
	}
}
