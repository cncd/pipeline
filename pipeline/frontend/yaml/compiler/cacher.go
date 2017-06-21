package compiler

import (
	"path"

	"github.com/cncd/pipeline/pipeline/frontend/yaml"

	libcompose "github.com/docker/libcompose/yaml"
)

// Cacher defines a compiler transform that can be used
// to implement default caching for a repository.
type Cacher interface {
	Restore(repo, branch string) *yaml.Container
	Rebuild(repo, branch string) *yaml.Container
}

type volumeCacher struct {
	base string
}

func (c *volumeCacher) Restore(repo, branch string) *yaml.Container {
	return &yaml.Container{
		Name:  "rebuild_cache",
		Image: "plugins/volume-cache:latest",
		Vargs: map[string]interface{}{
			"rebuild":  true,
			"fallback": path.Join("/cache", "master", "cache.tar.gz"),
			"path":     path.Join("/cache", branch, "cache.tar.gz"),
		},
		Volumes: libcompose.Volumes{
			Volumes: []*libcompose.Volume{
				{
					Source:      path.Join(c.base, repo),
					Destination: "/cache",
					// TODO add access mode
				},
			},
		},
	}
}

func (c *volumeCacher) Rebuild(repo, branch string) *yaml.Container {
	return &yaml.Container{
		Name:  "rebuild_cache",
		Image: "plugins/volume-cache:latest",
		Vargs: map[string]interface{}{
			"rebuild": true,
			"path":    path.Join("/cache", branch, "cache.tar.gz"),
		},
		Volumes: libcompose.Volumes{
			Volumes: []*libcompose.Volume{
				{
					Source:      path.Join(c.base, repo),
					Destination: "/cache",
					// TODO add access mode
				},
			},
		},
	}
}

type s3Cacher struct {
	bucket string
	access string
	secret string
}

func (c *s3Cacher) Restore(repo, branch string) *yaml.Container {
	return &yaml.Container{
		Name:  "rebuild_cache",
		Image: "plugins/s3-cache:latest",
		Vargs: map[string]interface{}{
			"access_key": c.access,
			"secret_key": c.secret,
			"bucket":     c.bucket,
			"rebuild":    true,
			"fallback":   path.Join(repo, "master", "cache.tar.gz"),
			"path":       path.Join(repo, branch, "cache.tar.gz"),
		},
	}
}

func (c *s3Cacher) Rebuild(repo, branch string) *yaml.Container {
	return &yaml.Container{
		Name:  "rebuild_cache",
		Image: "plugins/s3-cache:latest",
		Vargs: map[string]interface{}{
			"access_key": c.access,
			"secret_key": c.secret,
			"bucket":     c.bucket,
			"rebuild":    true,
			"path":       path.Join(repo, branch, "cache.tar.gz"),
		},
	}
}
