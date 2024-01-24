/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clusterstack

import (
	"fmt"
	"regexp"
	"strconv"
)

// BumpVersion bumps the version of the cluster stacks component.
func BumpVersion(version string) (string, error) {
	// Define a regular expression to extract the numeric part of the version
	re := regexp.MustCompile(`(\d+)`)
	matches := re.FindStringSubmatch(version)

	// Check if a numeric part was found
	if len(matches) < 2 {
		return "", fmt.Errorf("invalid version format")
	}

	// Extract and convert the numeric part to an integer
	currentMajor, err := strconv.Atoi(matches[1])
	if err != nil {
		return "", fmt.Errorf("failed to parse major version: %w", err)
	}

	// Increment the major version
	newMajor := currentMajor + 1

	// Replace the old major version with the new one
	newVersion := re.ReplaceAllString(version, strconv.Itoa(newMajor))

	return newVersion, nil
}
