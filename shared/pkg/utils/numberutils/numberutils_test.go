package numberutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		name  string
		input int64
		want  string
	}{
		{name: "zero", input: 0, want: "0 byte"},
		{name: "single byte", input: 1, want: "1 byte"},
		{name: "below one KiB", input: 1023, want: "1023 byte"},
		{name: "exact one KiB", input: 1024, want: "1 KB"},
		{name: "fractional KiB rounds up", input: 1536, want: "2 KB"},
		{name: "just below one MiB", input: 1024*1024 - 1, want: "1024 KB"},
		{name: "exact one MiB", input: 1024 * 1024, want: "1 MB"},
		{name: "fractional MiB", input: 1572864, want: "1.5 MB"},
		{name: "ten MiB", input: 10 * 1024 * 1024, want: "10 MB"},
		{name: "one hundred MiB", input: 100 * 1024 * 1024, want: "100 MB"},
		{name: "one GiB", input: 1024 * 1024 * 1024, want: "1024 MB"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, FormatBytes(tc.input))
		})
	}
}
