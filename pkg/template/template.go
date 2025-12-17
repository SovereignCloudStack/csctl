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

// Package template implements important functions for templating.
package template

import (
	"fmt"
	"os"
	"path/filepath"

	csctlclusterstack "github.com/SovereignCloudStack/csctl/pkg/clusterstack"
	"github.com/valyala/fasttemplate"
)

// CustomWalkFunc is the type for the walk function.
type CustomWalkFunc func(src, dst, path string, info os.FileInfo, meta *csctlclusterstack.MetaData) error

// MyWalk is the custom walking function to walk in the cluster stacks.
func MyWalk(src, dst string, walkFn CustomWalkFunc, meta *csctlclusterstack.MetaData) error {
	if err := filepath.Walk(src, func(path string, info os.FileInfo, _ error) error {
		return walkFn(src, dst, path, info, meta)
	}); err != nil {
		return fmt.Errorf("failed to walk files: %w", err)
	}

	return nil
}

func visitFile(src, dst, path string, info os.FileInfo, meta *csctlclusterstack.MetaData) error {
	relativePath, err := filepath.Rel(src, path)
	if err != nil {
		return fmt.Errorf("failed to relate directory: %w", err)
	}

	destPath := filepath.Join(dst, relativePath)
	if info.IsDir() {
		if err := os.MkdirAll(destPath, 0o750); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		return nil
	}

	fileData, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	tmp, err := fasttemplate.NewTemplate(string(fileData), "<< ", " >>")
	if err != nil {
		return fmt.Errorf("failed to create new template: %w", err)
	}

	output := tmp.ExecuteString(map[string]interface{}{
		".ClusterClassVersion": meta.Versions.ClusterStack,
		".ClusterAddonVersion": meta.Versions.Components.ClusterAddon,
		".NodeImageVersion":    meta.Versions.Components.NodeImage,
	})

	if err := os.WriteFile(destPath, []byte(output), 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// GenerateOutputFromTemplate is used to generate the template with replaced values.
func GenerateOutputFromTemplate(src, dst string, meta *csctlclusterstack.MetaData) error {
	return MyWalk(src, dst, visitFile, meta)
}
