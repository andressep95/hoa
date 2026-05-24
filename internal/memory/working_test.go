package memory

import "testing"

func TestHumanBytes(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "0B"},
		{999, "999B"},
		{1023, "1023B"},
		{1024, "1.0KB"},
		{1500, "1.5KB"},
		{1_500_000, "1.4MB"},
		{2 * 1024 * 1024, "2.0MB"},
		{3 * 1024 * 1024 * 1024, "3.0GB"},
	}
	for _, c := range cases {
		got := HumanBytes(c.in)
		if got != c.want {
			t.Errorf("HumanBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
