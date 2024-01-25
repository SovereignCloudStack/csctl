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
	"gopkg.in/yaml.v3"
)

// CsmctlConfig contains information of CsmctlConfig yaml.
type CsmctlConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Config     struct {
		KubernetesVersion string `yaml:"kubernetesVersion"`
		ClusterStackName  string `yaml:"clusterStackName"`
		Provider          struct {
			Type       string                 `yaml:"type"`
			APIVersion string                 `yaml:"apiVersion"`
			Config     map[string]interface{} `yaml:"config"`
		} `yaml:"provider"`
	} `yaml:"config"`
}

// GetCsmctlConfig returns CsmctlConfig.
func GetCsmctlConfig(path string) (*CsmctlConfig, error) {
	configPath := filepath.Join(path, "csmctl.yaml")
	configFileData, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		return nil, fmt.Errorf("failed to read csmctl config: %w", err)
	}

	cs := &CsmctlConfig{}
	if err := yaml.Unmarshal(configFileData, &cs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal csmctl yaml: %w", err)
	}

	if cs.Config.Provider.Type == "" {
		return nil, fmt.Errorf("provider type must not be empty")
	}

	if len(cs.Config.Provider.Type) > 253 {
		return nil, fmt.Errorf("provider name must not be greater than 253")
	}

	match, err := regexp.MatchString(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`, cs.Config.Provider.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to provider name match regex: %w", err)
	}
	if !match {
		return nil, fmt.Errorf("invalid provider type: %q", cs.Config.Provider.Type)
	}

	if cs.Config.ClusterStackName == "" {
		return nil, fmt.Errorf("cluster stack name must not be empty")
	}

	// Validate kubernetes version
	matched, err := regexp.MatchString(`^v\d+\.\d+\.\d+$`, cs.Config.KubernetesVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to kubernetes match regex: %w", err)
	}
	if !matched {
		return nil, fmt.Errorf("invalid kubernetes version: %q", cs.Config.KubernetesVersion)
	}

	return cs, nil
}

// ParseKubernetesVersion parse the kubernetes version present in the Csmctl Config.
func (c *CsmctlConfig) ParseKubernetesVersion() (kubernetesversion.KubernetesVersion, error) {
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
