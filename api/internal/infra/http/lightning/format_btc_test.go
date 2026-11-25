package lightninghttp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatBTC(t *testing.T) {
	tests := map[string]struct {
		sat  int64
		want string
	}{
		"zero":                  {0, "0.00"},
		"one sat":               {1, "0.00"},
		"just below half cent":  {499999, "0.00"},
		"exactly half cent":     {500000, "0.01"},
		"one cent":              {1000000, "0.01"},
		"1.234567 BTC":          {123456700, "1.23"},
		"1.235 BTC rounds up":   {123500000, "1.24"},
		"one bitcoin":           {100000000, "1.00"},
		"21 million BTC":        {2100000000000000, "21000000.00"},
		"large with round up":   {999500000, "10.00"},
		"large without roundup": {999499999, "9.99"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, formatBTC(tc.sat))
		})
	}
}
