package oapi

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/util"
)

const (
	OpenShiftAPIServerPort  = 8443
	OpenShiftAPIServicePort = 443
)

var (
	openshiftAPIServerLabels = map[string]string{"app": "openshift-apiserver"}
	oauthAPIServerLabels     = map[string]string{"app": "openshift-oauth-apiserver"}
)

func (p *OpenShiftAPIServerServiceParams) ReconcileOpenShiftAPIService(svc *corev1.Service) error {
	return p.reconcileAPIService(svc, openshiftAPIServerLabels)
}

func (p *OpenShiftAPIServerServiceParams) ReconcileOAuthAPIService(svc *corev1.Service) error {
	return p.reconcileAPIService(svc, oauthAPIServerLabels)
}

func (p *OpenShiftAPIServerServiceParams) reconcileAPIService(svc *corev1.Service, labels map[string]string) error {
	util.EnsureOwnerRef(svc, p.OwnerReference)
	svc.Spec.Selector = labels
	var portSpec corev1.ServicePort
	if len(svc.Spec.Ports) > 0 {
		portSpec = svc.Spec.Ports[0]
	} else {
		svc.Spec.Ports = []corev1.ServicePort{portSpec}
	}
	portSpec.Name = "https"
	portSpec.Port = int32(OpenShiftAPIServicePort)
	portSpec.Protocol = corev1.ProtocolTCP
	portSpec.TargetPort = intstr.FromInt(OpenShiftAPIServerPort)
	svc.Spec.Type = corev1.ServiceTypeClusterIP
	svc.Spec.Ports[0] = portSpec
	return nil
}
