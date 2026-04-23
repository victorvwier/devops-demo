package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type TinyLLMServiceSpec struct {
	Image         string                      `json:"image,omitempty"`
	Replicas      *int32                      `json:"replicas,omitempty"`
	Model         TinyLLMModelSpec            `json:"model,omitempty"`
	PromptPrefix  string                      `json:"promptPrefix,omitempty"`
	Resources     corev1.ResourceRequirements `json:"resources,omitempty"`
	Observability TinyLLMObservability        `json:"observability,omitempty"`
}

type TinyLLMModelSpec struct {
	Repository string `json:"repository,omitempty"`
	File       string `json:"file,omitempty"`
	Revision   string `json:"revision,omitempty"`
}

type TinyLLMObservability struct {
	BeylaEnabled bool `json:"beylaEnabled,omitempty"`
}

type TinyLLMServiceStatus struct {
	Phase             string      `json:"phase,omitempty"`
	ReadyReplicas     int32       `json:"readyReplicas,omitempty"`
	BackendMode       string      `json:"backendMode,omitempty"`
	BackendURL        string      `json:"backendURL,omitempty"`
	FrontendURL       string      `json:"frontendURL,omitempty"`
	LastReconcileTime metav1.Time `json:"lastReconcileTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type TinyLLMService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TinyLLMServiceSpec   `json:"spec,omitempty"`
	Status TinyLLMServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type TinyLLMServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TinyLLMService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TinyLLMService{}, &TinyLLMServiceList{})
}

func (in *TinyLLMService) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(TinyLLMService)
	*out = *in
	out.ObjectMeta = *in.ObjectMeta.DeepCopy()
	return out
}

func (in *TinyLLMServiceList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(TinyLLMServiceList)
	*out = *in
	if in.Items != nil {
		out.Items = make([]TinyLLMService, len(in.Items))
		copy(out.Items, in.Items)
	}
	return out
}
