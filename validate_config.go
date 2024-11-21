package main

import (
	"fmt"
	"strconv"
)

func validatePort(port int) (bool, string) {
	switch {
	case port == 0:
		return false, "No port defined"
	case port > 0 && port <= 1023:
		return true, fmt.Sprintf("Warning, Specified port (%d) is within 'Well known port' range. Recommend port in range 49152 -> 65535", port)
	case port >= 1024 && port <= 49151:
		return true, fmt.Sprintf("Warning, Specified port (%d) is within 'Registered port' range. Recommend port in range 49152 -> 65535", port)
	case port >= 49152 && port <= 65535:
		return true, ""
	case port > 65535:
		return false, fmt.Sprintf("Specified Port (%d) too large.  Recommend port in range 49152 -> 65535", port)
	}
	return false, fmt.Sprintf("Unable to validate port (%v)", port)
}

func validatePortString(port string) (bool, string) {
	portInt, err := strconv.Atoi(port)
	if err != nil {
		return false, fmt.Sprintf("Invalid port (%v). Please ensure port is a valid integer.", port)
	}
	return validatePort(portInt)
}
