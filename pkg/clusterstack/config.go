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

// Package clusterstack has necessary function to work on cluster stack.
package clusterstack

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/SovereignCloudStack/cluster-stack-operator/pkg/kubernetesversion"
	"github.com/SovereignCloudStack/cluster-stack-operator/pkg/version"
	"gopkg.in/yaml.v3"
)

// CsctlConfig contains information of CsctlConfig yaml.
type CsctlConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Config     struct {
		KubernetesVersion string `yaml:"kubernetesVersion"`
		ClusterStackName  string `yaml:"clusterStackName"`
		Provider          struct {
			Type       string   `yaml:"type"`
			APIVersion string   `yaml:"apiVersion"`
			Config     struct{} `yaml:"config"`
		} `yaml:"provider"`
	} `yaml:"config"`
}

// GetCsctlConfig returns CsctlConfig.
func GetCsctlConfig(path string) (CsctlConfig, error) {
	configPath := filepath.Join(path, "csctl.yaml")
	configFileData, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		return CsctlConfig{}, fmt.Errorf("failed to read csctl config: %w", err)
	}

	cs := CsctlConfig{}
	if err := yaml.Unmarshal(configFileData, &cs); err != nil {
		return CsctlConfig{}, fmt.Errorf("failed to unmarshal csctl yaml: %w", err)
	}

	if cs.Config.Provider.Type == "" {
		return CsctlConfig{}, fmt.Errorf("provider type must not be empty")
	}

	if len(cs.Config.Provider.Type) > 253 {
		return CsctlConfig{}, fmt.Errorf("provider name must not be greater than 253")
	}

	match, err := regexp.MatchString(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`, cs.Config.Provider.Type)
	if err != nil {
		return CsctlConfig{}, fmt.Errorf("failed to provider name match regex: %w", err)
	}
	if !match {
		return CsctlConfig{}, fmt.Errorf("invalid provider type: %q", cs.Config.Provider.Type)
	}

	if cs.Config.ClusterStackName == "" {
		return CsctlConfig{}, fmt.Errorf("cluster stack name must not be empty")
	}

	// Validate kubernetes version
	matched, err := regexp.MatchString(`^v\d+\.\d+\.\d+$`, cs.Config.KubernetesVersion)
	if err != nil {
		return CsctlConfig{}, fmt.Errorf("failed to kubernetes match regex: %w", err)
	}
	if !matched {
		return CsctlConfig{}, fmt.Errorf("invalid kubernetes version: %q", cs.Config.KubernetesVersion)
	}

	return cs, nil
}

// ParseKubernetesVersion parse the kubernetes version present in the Csctl Config.
func (c *CsctlConfig) ParseKubernetesVersion() (kubernetesversion.KubernetesVersion, error) {
	splitted := strings.Split(c.Config.KubernetesVersion, ".")

	if len(splitted) != 3 {
		return kubernetesversion.KubernetesVersion{}, kubernetesversion.ErrInvalidFormat
	}

	major, err := strconv.Atoi(strings.TrimPrefix(splitted[0], "v"))
	if err != nil {
		return kubernetesversion.KubernetesVersion{}, kubernetesversion.ErrInvalidMajorVersion
	}

	minor, err := strconv.Atoi(splitted[1])
	if err != nil {
		return kubernetesversion.KubernetesVersion{}, kubernetesversion.ErrInvalidMinorVersion
	}

	return kubernetesversion.KubernetesVersion{
		Major: major,
		Minor: minor,
	}, nil
}

// GetClusterStackReleaseDirectoryName returns cluster stack release directory.
// e.g. - docker-ferrol-1-27-v1/ .
func GetClusterStackReleaseDirectoryName(metadata *MetaData, config *CsctlConfig) (string, error) {
	// Parse the cluster stack version from dot format `v1-alpha.0` to a version way of struct
	// and parse the kubernetes version from `v1.27.3` to a major minor way
	// and create the release directory at the end.
	clusterStackVersion, err := version.New(metadata.Versions.ClusterStack)
	if err != nil {
		return "", fmt.Errorf("failed to parse cluster stack version: %w", err)
	}
	kubernetesVerion, err := config.ParseKubernetesVersion()
	if err != nil {
		return "", fmt.Errorf("failed to parse kubernetes version: %w", err)
	}
	clusterStackReleaseDirName := fmt.Sprintf("%s-%s-%s-%s", config.Config.Provider.Type, config.Config.ClusterStackName, kubernetesVerion.String(), clusterStackVersion.String())

	return clusterStackReleaseDirName, nil
}
