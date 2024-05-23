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
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/SovereignCloudStack/csctl/pkg/clusterstack"
	"helm.sh/helm/v3/pkg/action"
)

// CreatePackage creates the package for release.
func CreatePackage(src, dst string, newType bool, config *clusterstack.CsctlConfig, metadata *clusterstack.MetaData) error {
	fmt.Printf("path now: %q\n", filepath.Join(src, "cluster-class"))
	if err := createHelmPackage(filepath.Join(src, "cluster-class"), dst); err != nil {
		return fmt.Errorf("failed to create package for ClusterClass: %w", err)
	}

	kubernetesVerion, err := config.ParseKubernetesVersion()
	if err != nil {
		return fmt.Errorf("failed to parse kubernetes version: %w", err)
	}

	if newType {
		clusterAddonDst := filepath.Join(dst, fmt.Sprintf("%s-%s-%s-cluster-addon-%s.tgz", config.Config.Provider.Type, config.Config.ClusterStackName, kubernetesVerion.String(), metadata.Versions.Components.ClusterAddon))
		if err := createTarPackage(filepath.Join(src, "cluster-addon"), clusterAddonDst); err != nil {
			return fmt.Errorf("failed to create package for ClusterAddon: %w", err)
		}
	} else {
		fmt.Printf("path now: %q\n", filepath.Join(src, "cluster-addon"))
		if err := createHelmPackage(filepath.Join(src, "cluster-addon"), dst); err != nil {
			return fmt.Errorf("failed to create helm package for ClusterAddon: %w", err)
		}
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

func createTarPackage(src, dst string) error {
	outFile, err := os.Create(filepath.Clean(dst))
	if err != nil {
		return fmt.Errorf("failed to create tar output destination directory: %w", err)
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	if err := filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Ignore the root folder itself
		if path == src {
			return nil
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path of source: %q and destination: %q directory: %w", src, dst, err)
		}

		// Use filepath.ToSlash to convert path separators to '/'
		relPath = filepath.ToSlash(relPath)

		header, err := tar.FileInfoHeader(info, relPath)
		if err != nil {
			return fmt.Errorf("failed to get the tar info header: %w", err)
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to set the write header: %w", err)
		}

		if !info.IsDir() {
			file, err := os.Open(filepath.Clean(path))
			if err != nil {
				return fmt.Errorf("failed to open path: %w", err)
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return fmt.Errorf("failed to copy file: %w", err)
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to walk on the source directory: %w", err)
	}

	return nil
}
