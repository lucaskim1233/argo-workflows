package executor

import (
	"context"
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/artifactory"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/gcs"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/git"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/hdfs"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/http"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/oss"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/raw"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/resource"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/s3"
)

// ArtifactDriver is the interface for loading and saving of artifacts
type ArtifactDriver interface {
	// Load accepts an artifact source URL and places it at specified path
	Load(inputArtifact *wfv1.Artifact, path string) error

	// Save uploads the path to artifact destination
	Save(path string, outputArtifact *wfv1.Artifact) error
}

var ErrUnsupportedDriver = fmt.Errorf("unsupported artifact driver")

type NewDriverFunc func(ctx context.Context, art *wfv1.Artifact, ri resource.Interface) (ArtifactDriver, error)

// NewDriver initializes an instance of an artifact driver
func NewDriver(ctx context.Context, art *wfv1.Artifact, ri resource.Interface) (ArtifactDriver, error) {
	if art.S3 != nil {
		var accessKey string
		var secretKey string

		if art.S3.AccessKeySecret.Name != "" {
			accessKeyBytes, err := ri.GetSecret(ctx, art.S3.AccessKeySecret.Name, art.S3.AccessKeySecret.Key)
			if err != nil {
				return nil, err
			}
			accessKey = accessKeyBytes
			secretKeyBytes, err := ri.GetSecret(ctx, art.S3.SecretKeySecret.Name, art.S3.SecretKeySecret.Key)
			if err != nil {
				return nil, err
			}
			secretKey = secretKeyBytes
		}

		driver := s3.S3ArtifactDriver{
			Endpoint:    art.S3.Endpoint,
			AccessKey:   accessKey,
			SecretKey:   secretKey,
			Secure:      art.S3.Insecure == nil || !*art.S3.Insecure,
			Region:      art.S3.Region,
			RoleARN:     art.S3.RoleARN,
			UseSDKCreds: art.S3.UseSDKCreds,
		}
		return &driver, nil
	}
	if art.HTTP != nil {
		return &http.HTTPArtifactDriver{}, nil
	}
	if art.Git != nil {
		gitDriver := git.GitArtifactDriver{
			InsecureIgnoreHostKey: art.Git.InsecureIgnoreHostKey,
		}
		if art.Git.UsernameSecret != nil {
			usernameBytes, err := ri.GetSecret(ctx, art.Git.UsernameSecret.Name, art.Git.UsernameSecret.Key)
			if err != nil {
				return nil, err
			}
			gitDriver.Username = usernameBytes
		}
		if art.Git.PasswordSecret != nil {
			passwordBytes, err := ri.GetSecret(ctx, art.Git.PasswordSecret.Name, art.Git.PasswordSecret.Key)
			if err != nil {
				return nil, err
			}
			gitDriver.Password = passwordBytes
		}
		if art.Git.SSHPrivateKeySecret != nil {
			sshPrivateKeyBytes, err := ri.GetSecret(ctx, art.Git.SSHPrivateKeySecret.Name, art.Git.SSHPrivateKeySecret.Key)
			if err != nil {
				return nil, err
			}
			gitDriver.SSHPrivateKey = sshPrivateKeyBytes
		}

		return &gitDriver, nil
	}
	if art.Artifactory != nil {
		usernameBytes, err := ri.GetSecret(ctx, art.Artifactory.UsernameSecret.Name, art.Artifactory.UsernameSecret.Key)
		if err != nil {
			return nil, err
		}
		passwordBytes, err := ri.GetSecret(ctx, art.Artifactory.PasswordSecret.Name, art.Artifactory.PasswordSecret.Key)
		if err != nil {
			return nil, err
		}
		driver := artifactory.ArtifactoryArtifactDriver{
			Username: usernameBytes,
			Password: passwordBytes,
		}
		return &driver, nil

	}
	if art.HDFS != nil {
		return hdfs.CreateDriver(ctx, ri, art.HDFS)
	}
	if art.Raw != nil {
		return &raw.RawArtifactDriver{}, nil
	}

	if art.OSS != nil {
		var accessKey string
		var secretKey string

		if art.OSS.AccessKeySecret.Name != "" {
			accessKeyBytes, err := ri.GetSecret(ctx, art.OSS.AccessKeySecret.Name, art.OSS.AccessKeySecret.Key)
			if err != nil {
				return nil, err
			}
			accessKey = string(accessKeyBytes)
			secretKeyBytes, err := ri.GetSecret(ctx, art.OSS.SecretKeySecret.Name, art.OSS.SecretKeySecret.Key)
			if err != nil {
				return nil, err
			}
			secretKey = string(secretKeyBytes)
		}

		driver := oss.OSSArtifactDriver{
			Endpoint:  art.OSS.Endpoint,
			AccessKey: accessKey,
			SecretKey: secretKey,
		}
		return &driver, nil
	}

	if art.GCS != nil {
		driver := gcs.ArtifactDriver{}
		if art.GCS.ServiceAccountKeySecret.Name != "" {
			serviceAccountKeyBytes, err := ri.GetSecret(ctx, art.GCS.ServiceAccountKeySecret.Name, art.GCS.ServiceAccountKeySecret.Key)
			if err != nil {
				return nil, err
			}
			serviceAccountKey := string(serviceAccountKeyBytes)
			driver.ServiceAccountKey = serviceAccountKey
		}
		// key is not set, assume it is using Workload Idendity
		return &driver, nil
	}

	return nil, ErrUnsupportedDriver
}
