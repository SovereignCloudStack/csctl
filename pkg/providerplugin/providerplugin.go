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

// Package providerplugin implements calling the provider specific csmctl plugin.
package providerplugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/SovereignCloudStack/csmctl/pkg/clusterstack"
)

// GetProviderExecutable returns the path to the provider plugin (like "csmctl-docker").
// If there is not "config" for the provider in csmctl.yaml, then needed is false and path is the empty string.
func GetProviderExecutable(config *clusterstack.CsmctlConfig) (needed bool, path string, err error) {
	if len(config.Config.Provider.Config) == 0 {
		return false, "", nil
	}
	pluginName := "csmctl-" + config.Config.Provider.Type
	_, err = os.Stat(pluginName)
	if err == nil {
		path, err := filepath.Abs(pluginName)
		if err != nil {
			return false, "", err
		}
		return true, path, nil
	}
	path, err = exec.LookPath(pluginName)
	if err != nil {
		return false, "", fmt.Errorf("could not find plugin %s in $PATH or current working directory", pluginName)
	}
	return true, path, nil
}

// CreateNodeImages calls the provider plugin command to create nodes images.
func CreateNodeImages(config *clusterstack.CsmctlConfig, clusterStackPath, clusterStackReleaseDir string) error {
	needed, path, err := GetProviderExecutable(config)
	if err != nil {
		return err
	}
	if !needed {
		fmt.Printf("No provider specifig configuration in csmctl.yaml. No need to call a plugin for provider %q\n", config.Config.Provider.Type)
		return nil
	}
	args := []string{"create-node-images", clusterStackPath, clusterStackReleaseDir}
	fmt.Printf("Calling Provider Plugin: %s\n", path)
	cmd := exec.Command(path, args...) // #nosec G204
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
