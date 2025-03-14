package svcutil

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidRange = errors.New("invalid range format")
var ErrEmptyRange = errors.New("empty range")
var ErrIPV6RangeNotSupported = errors.New("IPv6 range not supported, use comma-separated format")

type RangeType int

const (
	RangeTypeID RangeType = 0
	RangeTypeIP RangeType = 1
)

type Range struct {
	Type   RangeType
	Values []string
}

func NewIDRange(value string) *Range {
	ids, err := ParseIDRange(value)
	if err != nil {
		return nil
	}

	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = strconv.Itoa(id)
	}

	return &Range{
		Type:   RangeTypeID,
		Values: strIDs,
	}
}

func ParseIDRange(input string) ([]int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, ErrInvalidRange
	}

	var result []int

	if strings.Contains(input, "-") {
		parts := strings.Split(input, "-")
		if len(parts) != 2 {
			return nil, ErrInvalidRange
		}

		start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, ErrInvalidRange
		}

		end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, ErrInvalidRange
		}

		if start > end {
			return nil, ErrInvalidRange
		}

		for i := start; i <= end; i++ {
			result = append(result, i)
		}
	} else {
		parts := strings.Split(input, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			num, err := strconv.Atoi(part)
			if err != nil {
				return nil, ErrInvalidRange
			}

			result = append(result, num)
		}
	}

	if len(result) == 0 {
		return nil, ErrEmptyRange
	}

	return result, nil
}

func NewIPRange(value string) *Range {
	ips, err := ParseIPRange(value)
	if err != nil {
		return nil
	}

	return &Range{
		Type:   RangeTypeIP,
		Values: ips,
	}
}

func ParseIPRange(input string) ([]string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, ErrInvalidRange
	}

	var result []string

	if strings.Contains(input, "-") {
		parts := strings.Split(input, "-")
		if len(parts) != 2 {
			return nil, ErrInvalidRange
		}

		startIP := strings.TrimSpace(parts[0])
		endIP := strings.TrimSpace(parts[1])

		if !isValidIP(startIP) || !isValidIP(endIP) {
			return nil, ErrInvalidRange
		}

		if isIPv6(startIP) || isIPv6(endIP) {
			return nil, ErrIPV6RangeNotSupported
		}

		var err error
		result, err = generateIPRange(startIP, endIP)
		if err != nil {
			return nil, err
		}
	} else {
		parts := strings.Split(input, ",")
		for _, part := range parts {
			ip := strings.TrimSpace(part)
			if ip == "" {
				continue
			}

			if !isValidIP(ip) {
				return nil, ErrInvalidRange
			}

			result = append(result, ip)
		}
	}

	if len(result) == 0 {
		return nil, ErrEmptyRange
	}

	return result, nil
}

func isValidIP(ip string) bool {
	return isIPv4(ip) || isIPv6(ip)
}

func isIPv4(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}

	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			return false
		}

		if num < 0 || num > 255 {
			return false
		}

		if part[0] == '0' && len(part) > 1 {
			return false
		}
	}

	return true
}

func isIPv6(ip string) bool {
	if !strings.Contains(ip, ":") {
		return false
	}

	if strings.Contains(ip, ":::") {
		return false
	}

	parts := strings.Split(ip, ":")

	if len(parts) > 8 {
		return false
	}

	doubleColonCount := strings.Count(ip, "::")
	if doubleColonCount > 1 {
		return false
	}

	if doubleColonCount == 1 {
		if len(parts) > 7 {
			return false
		}
	} else {
		if len(parts) != 8 {
			return false
		}
	}

	for _, part := range parts {
		if part == "" {
			continue
		}

		if len(part) > 4 {
			return false
		}

		_, err := strconv.ParseUint(part, 16, 16)
		if err != nil {
			return false
		}
	}

	return true
}

func generateIPRange(startIP, endIP string) ([]string, error) {
	start := ipv4ToInt(startIP)
	end := ipv4ToInt(endIP)

	if start > end {
		return nil, ErrInvalidRange
	}

	var ips []string
	for i := start; i <= end; i++ {
		ips = append(ips, intToIPv4(i))
	}

	return ips, nil
}

func ipv4ToInt(ip string) uint32 {
	parts := strings.Split(ip, ".")
	var result uint32

	for i := 0; i < 4; i++ {
		num, _ := strconv.Atoi(parts[i])
		result = (result << 8) | uint32(num)
	}

	return result
}

func intToIPv4(ip uint32) string {
	return fmt.Sprintf(
		"%d.%d.%d.%d",
		(ip>>24)&0xFF,
		(ip>>16)&0xFF,
		(ip>>8)&0xFF,
		ip&0xFF,
	)
}
