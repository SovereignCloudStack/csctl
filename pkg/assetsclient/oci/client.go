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

// Package oci provides utilities to work with oci registries.
package oci

import (
	"context"
	"errors"
	"fmt"

	"github.com/SovereignCloudStack/csctl/pkg/assetsclient"
	imagev1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// Client represents the client for oci repository.
type Client struct {
	Repository *remote.Repository
}

type factory struct{}

// NewFactory returns a new factory for OCI clients.
func NewFactory() assetsclient.Factory {
	return &factory{}
}

var _ = assetsclient.Factory(&factory{})

var _ = assetsclient.Client(&Client{})

// NewClient creates a new ociClient.
func NewClient() (*Client, error) {
	config, err := newOCIConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI config: %w", err)
	}

	client := auth.Client{
		Credential: auth.StaticCredential(config.registry, auth.Credential{
			AccessToken: config.accessToken,
			Username:    config.username,
			Password:    config.password,
		}),
	}

	repository, err := remote.NewRepository(config.repository)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client to remote repository %s: %w", config.repository, err)
	}

	repository.Client = &client
	return &Client{Repository: repository}, nil
}

// NewClientForRepository creates a new ociClient for the provided repository.
func NewClientForRepository(repo string) (*Client, error) {
	config, err := newOCIConfigWithoutRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI config: %w", err)
	}

	client := auth.Client{
		Credential: auth.StaticCredential(config.registry, auth.Credential{
			AccessToken: config.accessToken,
			Username:    config.username,
			Password:    config.password,
		}),
	}

	repository, err := remote.NewRepository(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client to remote repository %s: %w", config.repository, err)
	}

	repository.Client = &client
	return &Client{Repository: repository}, nil
}

func (*factory) NewClient(ctx context.Context) (assetsclient.Client, error) {
	_ = ctx
	config, err := newOCIConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI config: %w", err)
	}

	client := auth.Client{
		Credential: auth.StaticCredential(config.registry, auth.Credential{
			AccessToken: config.accessToken,
			Username:    config.username,
			Password:    config.password,
		}),
	}

	repository, err := remote.NewRepository(config.repository)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client to remote repository %s: %w", config.repository, err)
	}

	repository.Client = &client
	return &Client{Repository: repository}, nil
}

// ListRelease returns a list of releases in the repository.
func (c *Client) ListRelease(ctx context.Context) ([]string, error) {
	tags, err := registry.Tags(ctx, c.Repository)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	return tags, nil
}

// FoundRelease checks if the specified release exists in the repository.
func (c *Client) FoundRelease(ctx context.Context, tag string) bool {
	if _, err := c.Repository.Resolve(ctx, tag); err != nil {
		return false
	}

	return true
}

// CopyRelease copies the release artifact to target repository.
func (c *Client) CopyRelease(ctx context.Context, sourceTag, targetRepository, targetTag string) error {
	destinationRepository, err := remote.NewRepository(targetRepository)
	if err != nil {
		return fmt.Errorf("failed to create OCI client to remote repository %s: %w", targetRepository, err)
	}

	destinationRepository.Client = c.Repository.Client

	if _, err := oras.Copy(ctx, c.Repository, sourceTag, destinationRepository, targetTag, oras.DefaultCopyOptions); err != nil {
		return fmt.Errorf("failed to copy release from source repository %q to destination repository %q: %w", c.Repository.Reference, targetRepository, err)
	}

	return nil
}

// DownloadReleaseAssets downloads the specified release artifact at the provided path.
func (c *Client) DownloadReleaseAssets(ctx context.Context, tag, path string) (reterr error) {
	dest, err := file.New(path)
	if err != nil {
		return fmt.Errorf("failed to create file store: %w", err)
	}

	defer func() {
		err := dest.Close()
		if err != nil {
			reterr = errors.Join(reterr, err)
		}
	}()

	_, err = oras.Copy(ctx, c.Repository, tag, dest, tag, oras.DefaultCopyOptions)
	if err != nil {
		return fmt.Errorf("failed to copy repository artifacts to path %s: %w", path, err)
	}

	return nil
}

// PushReleaseAssets pushes the provided release assets as an artifact into the repository.
func (c *Client) PushReleaseAssets(ctx context.Context, releaseAssets []assetsclient.ReleaseAsset, tag, dir, artifactType string, annotations map[string]string) error {
	filestore, err := file.New(dir)
	if err != nil {
		return fmt.Errorf("failed to create new file store: %w", err)
	}

	defer filestore.Close()

	descriptors := []imagev1.Descriptor{}
	for _, releaseAsset := range releaseAssets {
		fileDescriptor, err := filestore.Add(ctx, releaseAsset.FileName, releaseAsset.MediaType, "")
		if err != nil {
			return fmt.Errorf("failed to add file asset %s to filestore: %w", releaseAsset.FileName, err)
		}

		descriptors = append(descriptors, fileDescriptor)
	}

	manifestDesc, err := oras.PackManifest(ctx, filestore, oras.PackManifestVersion1_1, artifactType, oras.PackManifestOptions{
		Layers:              descriptors,
		ManifestAnnotations: annotations,
	})
	if err != nil {
		return fmt.Errorf("failed to generate manifest descriptor: %w", err)
	}

	if err := filestore.Tag(ctx, manifestDesc, tag); err != nil {
		return fmt.Errorf("failed to tag the manifest descriptor: %w", err)
	}

	if _, err := oras.Copy(ctx, filestore, tag, c.Repository, tag, oras.DefaultCopyOptions); err != nil {
		return fmt.Errorf("failed to copy release assets to remote repository: %w", err)
	}

	return nil
}
