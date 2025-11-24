package handler

import "testing"

func TestTruncateUri(t *testing.T) {
	tests := []struct {
        name     string
        uri		 string
		len      int
        expected string
    }{
        {"not uri lnd", "azertyuiop", 3, "azertyuiop"},
        {"uri lnd trunc 3", "azertyuiop@1.2.3.4:1234", 3, "aze...iop@1.2.3.4:1234"},
        {"uri lnd trunc 1", "azertyuiop@1.2.3.4:1234", 1, "a...p@1.2.3.4:1234"},
		{"uri lnd not trunc", "azertyuiop@1.2.3.4:1234", 10, "azertyuiop@1.2.3.4:1234"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := truncateUri(tt.uri, tt.len)
            if result != tt.expected {
                t.Errorf("got %s, want %s", result, tt.expected)
            }
        })
    }
}




func TestTruncate(t *testing.T) {
	tests := []struct {
        name     string
        uri		 string
		len      int
        expected string
    }{
        {"string trunc 3", "azertyuiop", 3, "aze...iop"},
		{"uri lnd trunc 3", "azertyuiop@1.2.3.4:1234", 3, "aze...234"},
        {"string trunc 1", "azertyuiop", 1, "a...p"},
		{"string not trunc", "azertyuiop", 5, "azertyuiop"},
		{"string not trunc 20", "azertyuiop", 20, "azertyuiop"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := truncate(tt.uri, tt.len)
            if result != tt.expected {
                t.Errorf("got %s, want %s", result, tt.expected)
            }
        })
    }
}
