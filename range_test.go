package svcutil

import (
	"reflect"
	"testing"
)

func TestNewIDRange(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Range
		wantErr  bool
	}{
		{
			name:  "hyphen range",
			input: "1-5",
			expected: &Range{
				Type:   RangeTypeID,
				Values: []string{"1", "2", "3", "4", "5"},
			},
			wantErr: false,
		},
		{
			name:  "comma separated values",
			input: "1,3,5,7",
			expected: &Range{
				Type:   RangeTypeID,
				Values: []string{"1", "3", "5", "7"},
			},
			wantErr: false,
		},
		{
			name:  "single value",
			input: "42",
			expected: &Range{
				Type:   RangeTypeID,
				Values: []string{"42"},
			},
			wantErr: false,
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid range format",
			input:    "1-a",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid value",
			input:    "a,b,c",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid range order",
			input:    "5-1",
			expected: nil,
			wantErr:  true,
		},
		{
			name:  "with whitespace",
			input: " 1 , 3 , 5 ",
			expected: &Range{
				Type:   RangeTypeID,
				Values: []string{"1", "3", "5"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewIDRange(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewIDRange(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("NewIDRange(%q) = %+v, want %+v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseIDRange(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []int
		wantErr  bool
	}{
		{
			name:     "simple range",
			input:    "1-5",
			expected: []int{1, 2, 3, 4, 5},
			wantErr:  false,
		},
		{
			name:     "single value range",
			input:    "1",
			expected: []int{1},
			wantErr:  false,
		},
		{
			name:     "comma separated",
			input:    "1,3,5",
			expected: []int{1, 3, 5},
			wantErr:  false,
		},
		{
			name:     "mixed format",
			input:    "1,3-5,7",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid format",
			input:    "1-2-3",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid numbers",
			input:    "1-a",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "reversed range",
			input:    "5-1",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "only commas",
			input:    ",,",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "spaces between values",
			input:    " 1 , 2 , 3 ",
			expected: []int{1, 2, 3},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseIDRange(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIDRange(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseIDRange(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewIPRange(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Range
		wantErr  bool
	}{
		{
			name:  "IP range",
			input: "192.168.1.1-192.168.1.5",
			expected: &Range{
				Type: RangeTypeIP,
				Values: []string{
					"192.168.1.1", "192.168.1.2", "192.168.1.3",
					"192.168.1.4", "192.168.1.5",
				},
			},
			wantErr: false,
		},
		{
			name:  "single IP range",
			input: "192.168.1.3",
			expected: &Range{
				Type: RangeTypeIP,
				Values: []string{
					"192.168.1.3",
				},
			},
			wantErr: false,
		},
		{
			name:  "comma separated IPs",
			input: "192.168.1.1,192.168.1.100",
			expected: &Range{
				Type:   RangeTypeIP,
				Values: []string{"192.168.1.1", "192.168.1.100"},
			},
			wantErr: false,
		},
		{
			name:  "comma separated IPv6 range",
			input: "2001:db8::1,2001:db8::10",
			expected: &Range{
				Type:   RangeTypeIP,
				Values: []string{"2001:db8::1", "2001:db8::10"},
			},
			wantErr: false,
		},
		{
			name:     "IPv6 range",
			input:    "2001:db8::1-2001:db8::10",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid IP",
			input:    "192.168.1.256",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "reversed range",
			input:    "192.168.1.10-192.168.1.1",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewIPRange(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewIPRange(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("NewIPRange(%q) = %+v, want %+v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsIPv4(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"valid IPv4", "192.168.1.1", true},
		{"invalid IPv4 - too high octet", "192.168.1.256", false},
		{"invalid IPv4 - too many octets", "192.168.1.1.5", false},
		{"invalid IPv4 - too few octets", "192.168.1", false},
		{"invalid IPv4 - non-numeric", "192.168.1.a", false},
		{"invalid IPv4 - leading zero", "192.168.1.01", false},
		{"IPv6 address", "2001:db8::1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIPv4(tt.ip); got != tt.expected {
				t.Errorf("isIPv4(%q) = %v, want %v", tt.ip, got, tt.expected)
			}
		})
	}
}

func TestIsIPv6(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"valid IPv6", "2001:db8::1", true},
		{"valid IPv6 full", "2001:0db8:0000:0000:0000:0000:0000:0001", true},
		{"valid IPv6 mixed case", "2001:DB8::1", true},
		{"invalid IPv6 - too many parts", "2001:db8:1:2:3:4:5:6:7:8", false},
		{"invalid IPv6 - too many double colons", "2001::db8::1", false},
		{"invalid IPv6 - invalid chars", "2001:db8::xyz", false},
		{"invalid IPv6 - too long segment", "2001:db8::10000", false},
		{"IPv4 address", "192.168.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIPv6(tt.ip); got != tt.expected {
				t.Errorf("isIPv6(%q) = %v, want %v", tt.ip, got, tt.expected)
			}
		})
	}
}

func TestIPv4ToIntAndBack(t *testing.T) {
	tests := []struct {
		name string
		ip   string
	}{
		{"simple IP", "192.168.1.1"},
		{"all zeros", "0.0.0.0"},
		{"all max", "255.255.255.255"},
		{"mixed", "10.20.30.40"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intValue := ipv4ToInt(tt.ip)
			ipValue := intToIPv4(intValue)
			if ipValue != tt.ip {
				t.Errorf("Round trip failed: %s -> %d -> %s", tt.ip, intValue, ipValue)
			}
		})
	}
}

func TestGenerateIPRange(t *testing.T) {
	tests := []struct {
		name     string
		start    string
		end      string
		expected []string
		wantErr  bool
	}{
		{
			name:     "small range",
			start:    "192.168.1.1",
			end:      "192.168.1.3",
			expected: []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
			wantErr:  false,
		},
		{
			name:     "single IP",
			start:    "192.168.1.1",
			end:      "192.168.1.1",
			expected: []string{"192.168.1.1"},
			wantErr:  false,
		},
		{
			name:     "across octet",
			start:    "192.168.1.254",
			end:      "192.168.2.1",
			expected: []string{"192.168.1.254", "192.168.1.255", "192.168.2.0", "192.168.2.1"},
			wantErr:  false,
		},
		{
			name:     "reversed range",
			start:    "192.168.1.5",
			end:      "192.168.1.1",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generateIPRange(tt.start, tt.end)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateIPRange(%q, %q) error = %v, wantErr %v", tt.start, tt.end, err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("generateIPRange(%q, %q) = %v, want %v", tt.start, tt.end, result, tt.expected)
			}
		})
	}
}
