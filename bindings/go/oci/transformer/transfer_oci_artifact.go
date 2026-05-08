package transformer

import (
	"context"
	"fmt"

	"ocm.software/open-component-model/bindings/go/credentials"
	descriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	accessv1 "ocm.software/open-component-model/bindings/go/oci/spec/access/v1"
	"ocm.software/open-component-model/bindings/go/oci/spec/transformation/v1alpha1"
	"ocm.software/open-component-model/bindings/go/repository"
	"ocm.software/open-component-model/bindings/go/runtime"
)

// OCIArtifactTransferer is implemented by repositories that can stream an OCI
// artifact directly from a source registry to a target registry without
// buffering the artifact through a temporary OCI layout tar file. The OCI
// resource repository implementation in
// ocm.software/open-component-model/bindings/go/oci/repository/resource
// satisfies this interface.
type OCIArtifactTransferer interface {
	repository.ResourceRepository
	// TransferOCIArtifact transfers an OCI artifact from the source registry
	// described by srcResource.Access to the target registry described by
	// targetAccess. It returns the access spec of the artifact at the
	// destination.
	TransferOCIArtifact(ctx context.Context, srcResource *descriptor.Resource, targetAccess *accessv1.OCIImage, srcCredentials, targetCredentials map[string]string) (*accessv1.OCIImage, error)
}

// TransferOCIArtifact is a fused get+add transformer that streams an OCI
// artifact directly from a source registry to a target registry, bypassing the
// temporary OCI layout tar file that the GetOCIArtifact / AddOCIArtifact split
// would otherwise produce. See [v1alpha1.TransferOCIArtifact] for the
// motivation.
type TransferOCIArtifact struct {
	Scheme             *runtime.Scheme
	Repository         OCIArtifactTransferer
	CredentialProvider credentials.Resolver
}

func (t *TransferOCIArtifact) Transform(ctx context.Context, step runtime.Typed) (runtime.Typed, error) {
	var transformation v1alpha1.TransferOCIArtifact
	if err := t.Scheme.Convert(step, &transformation); err != nil {
		return nil, fmt.Errorf("failed converting generic transformation to transfer oci artifact transformation: %w", err)
	}
	if transformation.Spec == nil {
		return nil, fmt.Errorf("spec is required for transfer oci artifact transformation")
	}
	if transformation.Spec.Resource == nil {
		return nil, fmt.Errorf("resource is required")
	}
	if transformation.Spec.TargetAccess == nil || transformation.Spec.TargetAccess.ImageReference == "" {
		return nil, fmt.Errorf("targetAccess.imageReference is required")
	}
	if transformation.Output == nil {
		transformation.Output = &v1alpha1.TransferOCIArtifactOutput{}
	}

	srcResource := descriptor.ConvertFromV2Resource(transformation.Spec.Resource)
	targetAccess := transformation.Spec.TargetAccess

	var srcCreds, targetCreds map[string]string
	if t.CredentialProvider != nil {
		if consumerID, err := t.Repository.GetResourceCredentialConsumerIdentity(ctx, srcResource); err == nil {
			if srcCreds, err = resolveCredentialsMap(ctx, t.CredentialProvider, consumerID); err != nil {
				return nil, fmt.Errorf("failed resolving source credentials: %w", err)
			}
		}

		// Resolve target credentials by synthesising a resource with the
		// target access spec, then asking the repository for that access's
		// consumer identity. This keeps the credential resolution logic in
		// one place (the OCI ResourceRepository).
		probe := srcResource.DeepCopy()
		probe.Access = targetAccess
		if consumerID, err := t.Repository.GetResourceCredentialConsumerIdentity(ctx, probe); err == nil {
			if targetCreds, err = resolveCredentialsMap(ctx, t.CredentialProvider, consumerID); err != nil {
				return nil, fmt.Errorf("failed resolving target credentials: %w", err)
			}
		}
	}

	updatedAccess, err := t.Repository.TransferOCIArtifact(ctx, srcResource, targetAccess, srcCreds, targetCreds)
	if err != nil {
		return nil, fmt.Errorf("failed transferring OCI artifact %v: %w", srcResource.ToIdentity(), err)
	}

	updatedResource := srcResource.DeepCopy()
	updatedResource.Access = updatedAccess

	v2UpdatedResource, err := descriptor.ConvertToV2Resource(t.Scheme, updatedResource)
	if err != nil {
		return nil, fmt.Errorf("failed converting resource to v2 format: %w", err)
	}

	transformation.Output.Resource = v2UpdatedResource

	return &transformation, nil
}
