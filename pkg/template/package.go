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

package template

import (
	"fmt"
	"path/filepath"

	"helm.sh/helm/v3/pkg/action"
)

// CreatePackage creates the package for release.
func CreatePackage(src, dst string) error {
	if err := createHelmPackage(filepath.Join(src, "cluster-class"), dst); err != nil {
		return fmt.Errorf("failed to create package for ClusterClass: %w", err)
	}

	if err := createHelmPackage(filepath.Join(src, "cluster-addon"), dst); err != nil {
		return fmt.Errorf("failed to create package for ClusterAddon: %w", err)
	}

	return nil
}

func createHelmPackage(src, dst string) error {
	helmPkg := action.NewPackage()
	helmPkg.Destination = dst

	_, err := helmPkg.Run(src, map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to run helm package: %w", err)
	}

	return nil
}
