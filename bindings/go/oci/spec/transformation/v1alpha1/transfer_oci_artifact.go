package v1alpha1

import (
	v2 "ocm.software/open-component-model/bindings/go/descriptor/v2"
	accessv1 "ocm.software/open-component-model/bindings/go/oci/spec/access/v1"
	"ocm.software/open-component-model/bindings/go/runtime"
)

// TransferOCIArtifactType is the runtime type name of the
// TransferOCIArtifact transformation.
const TransferOCIArtifactType = "TransferOCIArtifact"

// TransferOCIArtifact is a fused get+add transformation that streams an OCI
// artifact directly from a source registry to a target registry without
// buffering it through a temporary OCI layout tar file.
//
// The transformation graph emits this single node when both endpoints of a
// resource transfer are OCI registries. The streaming path:
//
//   - removes the multi-GiB temp file under the agent's tmpdir that the
//     download / upload split would otherwise materialise (no OOM on small
//     hosts, no /tmp pressure when /tmp is tmpfs),
//   - skips fetching layers from the source for blobs the destination already
//     has (oras.CopyGraph performs HEAD checks at the destination),
//   - skips pushing layers to the destination for blobs that already exist
//     there (same HEAD check).
//
// +k8s:deepcopy-gen:interfaces=ocm.software/open-component-model/bindings/go/runtime.Typed
// +k8s:deepcopy-gen=true
// +ocm:typegen=true
type TransferOCIArtifact struct {
	// +ocm:jsonschema-gen:enum=TransferOCIArtifact/v1alpha1
	Type   runtime.Type              `json:"type"`
	ID     string                    `json:"id"`
	Spec   *TransferOCIArtifactSpec  `json:"spec"`
	Output *TransferOCIArtifactOutput `json:"output,omitempty"`
}

// TransferOCIArtifactSpec is the input specification for the
// TransferOCIArtifact transformation.
//
// +k8s:deepcopy-gen=true
type TransferOCIArtifactSpec struct {
	// Resource is the source resource descriptor whose access points at the
	// OCI artifact to transfer.
	Resource *v2.Resource `json:"resource"`
	// TargetAccess is the OCI image access spec describing where the artifact
	// should be transferred to. The image reference must be tagged.
	TargetAccess *accessv1.OCIImage `json:"targetAccess"`
}

// TransferOCIArtifactOutput is the output specification for the
// TransferOCIArtifact transformation.
//
// +k8s:deepcopy-gen=true
type TransferOCIArtifactOutput struct {
	// Resource is the updated resource descriptor with the target OCI image
	// access populated.
	Resource *v2.Resource `json:"resource"`
}
