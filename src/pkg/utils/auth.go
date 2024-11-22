// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

package utils

import (
	"fmt"
	"os"
	"strings"
)

// parseTokens parses the MY_CLI_TOKENS environment variable into a map.
func parseTokens() (map[string]string, error) {
	tokensEnv := os.Getenv("MY_CLI_TOKENS")
	if tokensEnv == "" {
		return nil, fmt.Errorf("MY_CLI_TOKENS environment variable is not set")
	}

	// Initialize a map to store the tokens
	tokenMap := make(map[string]string)

	// Split the env var into key=value pairs separated by ";"
	pairs := strings.Split(tokensEnv, ";")
	for _, pair := range pairs {
		// Split each pair into "key=value"
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid token format: %s", pair)
		}

		key := strings.TrimSpace(parts[0])   // hostname:port
		value := strings.TrimSpace(parts[1]) // token
		tokenMap[key] = value
	}

	return tokenMap, nil
}

// getToken retrieves a token for a specific hostname and port.
func getToken(hostname string, port int, tokenMap map[string]string) (string, error) {
	key := fmt.Sprintf("%s:%d", hostname, port)
	if token, exists := tokenMap[key]; exists {
		return token, nil
	}
	return "", fmt.Errorf("no token found for %s", key)
}

func main() {
	// Parse tokens from the environment variable
	tokenMap, err := parseTokens()
	if err != nil {
		fmt.Println("Error parsing tokens:", err)
		return
	}

	// Example usage
	hostname := "gitlab.enterprise.corp"
	port := 8080

	token, err := getToken(hostname, port, tokenMap)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("Token for %s:%d is %s\n", hostname, port, token)
	}

	// Try a non-existent port
	port = 9000
	token, err = getToken(hostname, port, tokenMap)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("Token for %s:%d is %s\n", hostname, port, token)
	}
}
