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

// Package hash contains important functions of hash.
package hash

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/sumdb/dirhash"
)

const (
	clusterAddonDirName        = "cluster-addon"
	nodeImageDirName           = "node-image"
	clusterAddonValuesFileName = "cluster-addon-values.yaml"
)

// ReleaseHash contains the information of release hash.
type ReleaseHash struct {
	ClusterStack       string `json:"clusterStack"`
	ClusterAddonDir    string `json:"clusterAddonDir"`
	ClusterAddonValues string `json:"clusterAddonValues"`
	NodeImageDir       string `json:"nodeImageDir,omitempty"`
}

// GetHash returns the release hash.
func GetHash(path string) (ReleaseHash, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return ReleaseHash{}, fmt.Errorf("failed to read dir: %w", err)
	}

	releaseHash := ReleaseHash{}

	hash, err := dirhash.HashDir(path, "", dirhash.DefaultHash)
	if err != nil {
		return ReleaseHash{}, fmt.Errorf("failed to calculate cluster stack hash: %w", err)
	}
	hash = clean(hash)
	fmt.Printf("path %q: cluster stack hash: %q\n", path, hash)

	releaseHash.ClusterStack = hash

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		if entry.IsDir() && (entry.Name() == clusterAddonDirName || entry.Name() == nodeImageDirName) {
			hash, err := dirhash.HashDir(entryPath, "", dirhash.DefaultHash)
			if err != nil {
				return ReleaseHash{}, fmt.Errorf("failed to hash dir: %w", err)
			}
			hash = clean(hash)

			switch entry.Name() {
			case clusterAddonDirName:
				releaseHash.ClusterAddonDir = hash
			case nodeImageDirName:
				releaseHash.NodeImageDir = hash
			}
		} else if !entry.IsDir() && entry.Name() == clusterAddonValuesFileName {
			file, _ := os.Open(filepath.Clean(entryPath))

			fileHash := sha256.New()
			if _, err := io.Copy(fileHash, file); err != nil {
				return ReleaseHash{}, fmt.Errorf("failed to copy dir: %w", err)
			}
			releaseHash.ClusterAddonValues = clean(base64.StdEncoding.EncodeToString(fileHash.Sum(nil)))
		}
	}

	return releaseHash, nil
}

// ValidateWithLatestReleaseHash compare current hash with latest release hash.
func (r ReleaseHash) ValidateWithLatestReleaseHash(latestReleaseHash ReleaseHash) error {
	if r.ClusterAddonDir == latestReleaseHash.ClusterAddonDir &&
		r.ClusterAddonValues == latestReleaseHash.ClusterAddonValues &&
		r.NodeImageDir == latestReleaseHash.NodeImageDir {
		return fmt.Errorf("no change in the cluster stack")
	}

	return nil
}

func clean(hash string) string {
	hash = strings.TrimPrefix(hash, "h1:")
	hash = strings.ReplaceAll(hash, "/", "")
	hash = strings.ReplaceAll(hash, "=", "")
	hash = strings.ReplaceAll(hash, "+", "")
	hash = strings.ToLower(hash)

	return hash
}

// GetClusterStackHash returns the 7 character hash of the cluster stack content.
func (r ReleaseHash) GetClusterStackHash() string {
	return r.ClusterStack[:7]
}
