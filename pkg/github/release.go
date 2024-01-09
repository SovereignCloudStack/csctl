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

// Package github implements important functions for github client.
package github

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SovereignCloudStack/csmctl/pkg/hash"
)

// GHRelease contains fields for a release.
type GHRelease struct {
	Provider               string
	ClusterStackName       string
	KubernetesVersionMajor string
	KubernetesVersionMinor string
	ClusterStackVersion    string

	Hash hash.ReleaseHash
}

// GetLocalReleaseInfoWithHash gets the local release info.
// TODO: Later replaced by the original github release fetching code.
func GetLocalReleaseInfoWithHash(path string) (GHRelease, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return GHRelease{}, fmt.Errorf("failed to read github release path: %w", err)
	}

	if len(entries) != 1 {
		return GHRelease{}, fmt.Errorf("ambiguous release found")
	}

	splittedReleaseName := strings.Split(entries[0].Name(), "-")

	if len(splittedReleaseName) != 5 {
		return GHRelease{}, fmt.Errorf("wrong release found")
	}

	ghRelease := GHRelease{
		Provider:               splittedReleaseName[0],
		ClusterStackName:       splittedReleaseName[1],
		KubernetesVersionMajor: splittedReleaseName[2],
		KubernetesVersionMinor: splittedReleaseName[3],
		ClusterStackVersion:    splittedReleaseName[4],
	}

	hashPath := filepath.Join(path, entries[0].Name(), "hashes.json")
	hashFile, err := os.ReadFile(filepath.Clean(hashPath))
	if err != nil {
		return GHRelease{}, fmt.Errorf("failed to read hash.json: %w", err)
	}

	var data hash.ReleaseHash
	if err := json.Unmarshal(hashFile, &data); err != nil {
		return GHRelease{}, fmt.Errorf("failed to unmarshal release hash: %w", err)
	}
	ghRelease.Hash = data

	return ghRelease, nil
}
