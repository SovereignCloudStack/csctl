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

// Package assetsclient contains interface for talking to assets repositories.
package assetsclient

import (
	"context"
)

// Client contains functions to talk to list and download assets.
type Client interface {
	DownloadReleaseAssets(ctx context.Context, tag, path string) error
	ListRelease(ctx context.Context) ([]string, error)
}

// Factory is a factory to generate assets clients.
type Factory interface {
	NewClient(ctx context.Context) (Client, error)
}

// Pusher contains function to push the release assets to the registry.
type Pusher interface {
	PushReleaseAssets(ctx context.Context, releaseAssets []ReleaseAsset, tag, dir, artifactType string, metadata map[string]string) error
}

// ReleaseAsset represents a release asset that would together make up the artifact.
type ReleaseAsset struct {
	FileName  string
	MediaType string
}
