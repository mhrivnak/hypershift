package kcm

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	configv1 "github.com/openshift/api/config/v1"
	hyperv1 "github.com/openshift/hypershift/api/v1alpha1"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/cloud/aws"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/config"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/manifests"
)

type KubeControllerManagerParams struct {
	FeatureGate         configv1.FeatureGate        `json:"featureGate"`
	Network             configv1.Network            `json:"network"`
	ServiceCA           []byte                      `json:"serviceCA"`
	CloudProvider       string                      `json:"cloudProvider"`
	CloudProviderConfig corev1.LocalObjectReference `json:"cloudProviderConfig"`
	CloudProviderCreds  corev1.LocalObjectReference `json:"cloudProviderCreds"`
	Port                int32                       `json:"port"`

	Replicas         int32 `json:"replicas"`
	Scheduling       config.Scheduling
	AdditionalLabels map[string]string          `json:"additionalLabels"`
	SecurityContexts config.SecurityContextSpec `json:"securityContexts"`
	LivenessProbes   config.LivenessProbes      `json:"livenessProbes"`
	ReadinessProbes  config.ReadinessProbes     `json:"readinessProbes"`
	Resources        config.ResourcesSpec       `json:"resources"`
	OwnerReference   *metav1.OwnerReference     `json:"ownerReference"`

	HyperkubeImage string `json:"hyperkubeImage"`
}

const (
	DefaultPriorityClass = "system-node-critical"
	DefaultPort          = 10257
)

func NewKubeControllerManagerParams(hcp *hyperv1.HostedControlPlane, images map[string]string) *KubeControllerManagerParams {
	params := &KubeControllerManagerParams{
		FeatureGate: configv1.FeatureGate{
			Spec: configv1.FeatureGateSpec{
				FeatureGateSelection: configv1.FeatureGateSelection{
					FeatureSet: configv1.Default,
				},
			},
		},
		Network: config.Network(hcp),
		// TODO: Come up with sane defaults for scheduling APIServer pods
		// Expose configuration
		AdditionalLabels: map[string]string{},
		Scheduling: config.Scheduling{
			PriorityClass: DefaultPriorityClass,
		},
		HyperkubeImage: images["hyperkube"],
		Port:           DefaultPort,
	}
	params.LivenessProbes = config.LivenessProbes{
		kcmContainerMain().Name: {
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Scheme: corev1.URISchemeHTTPS,
					Port:   intstr.FromInt(int(params.Port)),
					Path:   "healthz",
				},
			},
			InitialDelaySeconds: 45,
			TimeoutSeconds:      10,
			PeriodSeconds:       10,
			FailureThreshold:    3,
			SuccessThreshold:    1,
		},
	}
	params.ReadinessProbes = config.ReadinessProbes{
		kcmContainerMain().Name: {
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Scheme: corev1.URISchemeHTTPS,
					Port:   intstr.FromInt(int(params.Port)),
					Path:   "healthz",
				},
			},
			InitialDelaySeconds: 10,
			TimeoutSeconds:      10,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			FailureThreshold:    3,
		},
	}
	params.Resources = map[string]corev1.ResourceRequirements{
		kcmContainerMain().Name: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("200Mi"),
				corev1.ResourceCPU:    resource.MustParse("60m"),
			},
		},
	}
	switch hcp.Spec.Platform.Type {
	case hyperv1.AWSPlatform:
		params.CloudProvider = aws.Provider
		params.CloudProviderConfig.Name = manifests.AWSProviderConfig("").Name
		params.CloudProviderCreds.Name = hcp.Spec.Platform.AWS.KubeCloudControllerCreds.Name
	}

	switch hcp.Spec.ControllerAvailabilityPolicy {
	case hyperv1.HighlyAvailable:
		params.Replicas = 3
	default:
		params.Replicas = 1
	}
	params.OwnerReference = config.ControllerOwnerRef(hcp)

	return params
}

func externalAddress(endpoint hyperv1.APIEndpoint) string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}
