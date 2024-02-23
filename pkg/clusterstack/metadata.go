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
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Component contains component.
type Component struct {
	ClusterAddon string `yaml:"clusterAddon"`
	NodeImage    string `yaml:"nodeImage,omitempty"`
}

// Versions contains version information.
type Versions struct {
	ClusterStack string    `yaml:"clusterStack"`
	Kubernetes   string    `yaml:"kubernetes"`
	Components   Component `yaml:"components"`
}

// MetaData contains metadata.
type MetaData struct {
	APIVersion string   `yaml:"apiVersion"`
	Versions   Versions `yaml:"versions"`
}

// ParseMetaData parse the metadata file.
func ParseMetaData(path string) (MetaData, error) {
	metadataPath := filepath.Join(path, "metadata.yaml")
	fileInfo, err := os.ReadFile(filepath.Clean(metadataPath))
	if err != nil {
		return MetaData{}, fmt.Errorf("failed to read metadata file: %w", err)
	}

	metaData := MetaData{}

	if err := yaml.Unmarshal(fileInfo, &metaData); err != nil {
		return MetaData{}, fmt.Errorf("failed to unmarshal metadata yaml: %w", err)
	}

	return metaData, nil
}
