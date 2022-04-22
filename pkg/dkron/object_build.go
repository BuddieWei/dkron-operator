package dkron

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	operatorv1 "github.com/buddiewei/dkron-operator/api/v1"
)

var (
	DKRON_SINGLE_CONFIGMAP_NAME string             = "dkron-single-config"
	DKRON_SERVER_CONFIGMAP_NAME string             = "dkron-server-config"
	DKRON_INIT_STS_NAME         string             = "dkron-init"
	DKRON_SERVER_STS_NAME       string             = "dkron-server"
	DKRON_SERVER_SERVICE_NAME   string             = "dkron-server"
	PATH_TYPE_PREFIX            networkv1.PathType = networkv1.PathTypePrefix
)

func BuildStatefulSet(ctx context.Context, dkronOperator *operatorv1.DkronOperator, stsName string, replicas int32) (*appsv1.StatefulSet, error) {
	sts := &appsv1.StatefulSet{}
	sts.Name = stsName
	sts.Namespace = dkronOperator.Spec.Namespace
	sts.Spec.Replicas = &replicas
	sts.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"k8s-app": stsName,
		},
	}
	sts.Spec.Template.Labels = map[string]string{
		"k8s-app":      stsName,
		"k8s-app-role": "dkron-server",
	}
	sts.Spec.Template.Annotations = dkronOperator.Annotations
	sts.Spec.Template.Spec.NodeSelector = dkronOperator.Spec.NodeSelector
	sts.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name: stsName,
			Image: func() string {
				if dkronOperator.Spec.Registry != "" {
					return fmt.Sprintf("%s/%s:%s", dkronOperator.Spec.Registry, dkronOperator.Spec.ImageName, dkronOperator.Spec.ImageTag)
				}
				return fmt.Sprintf("%s:%s", dkronOperator.Spec.ImageName, dkronOperator.Spec.ImageTag)
			}(),
			ImagePullPolicy: dkronOperator.Spec.ImagePullPolicy,
			Command: []string{
				"/opt/local/dkron/dkron",
			},
			Args: []string{
				"agent",
			},
			Resources: *dkronOperator.Spec.Resources,
			VolumeMounts: func() []corev1.VolumeMount {
				mounts := []corev1.VolumeMount{
					{
						Name:      "dkron-config",
						MountPath: "/etc/dkron/dkron.yml",
						SubPath:   "dkron.yml",
					},
				}
				if dkronOperator.Spec.StorageClass != "" {
					mounts = append(mounts, corev1.VolumeMount{
						Name:      "pvc-dkron",
						MountPath: dkronOperator.Spec.DkronConfig.DataDir,
					})
				}
				return mounts
			}(),
			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/health",
						Port: intstr.FromInt(8080),
					},
				},
				InitialDelaySeconds: dkronOperator.Spec.LivenessInitialDelaySeconds,
				PeriodSeconds:       3,
				FailureThreshold:    3,
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/health",
						Port: intstr.FromInt(8080),
					},
				},
				InitialDelaySeconds: dkronOperator.Spec.ReadinessInitialDelaySeconds,
				PeriodSeconds:       3,
				FailureThreshold:    3,
			},
		},
	}
	sts.Spec.Template.Spec.Volumes = []corev1.Volume{
		{
			Name: "dkron-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: func() string {
							if dkronOperator.Spec.Replicas > 1 {
								return DKRON_SERVER_CONFIGMAP_NAME
							}
							return DKRON_SINGLE_CONFIGMAP_NAME
						}(),
					},
				},
			},
		},
	}
	sts.Spec.Template.Spec.Affinity = dkronOperator.Spec.Affinity
	sts.Spec.Template.Spec.Tolerations = dkronOperator.Spec.Tolerations
	sts.Spec.Template.Spec.HostAliases = dkronOperator.Spec.HostAliases
	sts.Spec.Template.Spec.ImagePullSecrets = func() []corev1.LocalObjectReference {
		secretNames := dkronOperator.Spec.ImagePullSecrets
		if len(secretNames) == 0 {
			return nil
		}
		secretRefs := make([]corev1.LocalObjectReference, len(secretNames))
		for i, secretName := range secretNames {
			secretRefs[i] = corev1.LocalObjectReference{Name: secretName}
		}
		return secretRefs
	}()
	sts.Spec.VolumeClaimTemplates = func() []corev1.PersistentVolumeClaim {
		if dkronOperator.Spec.StorageClass == "" {
			return nil
		}
		return []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pvc-dkron",
					Annotations: map[string]string{
						"volume.beta.kubernetes.io/storage-class": dkronOperator.Spec.StorageClass,
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse(dkronOperator.Spec.StorageSize),
						},
					},
				},
			},
		}
	}()
	return sts, nil
}

// BuildConfigMap builds a ConfigMap object
func BuildConfigMap(ctx context.Context, dkronOperator *operatorv1.DkronOperator, cmName string, bootstrapExpect int) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	cm.Name = cmName
	cm.Namespace = dkronOperator.Spec.Namespace

	dkronData := map[string]string{
		"server":              "true",
		"bootstrap-expect":    strconv.Itoa(bootstrapExpect),
		"data-dir":            dkronOperator.Spec.DkronConfig.DataDir,
		"disable-usage-stats": strconv.FormatBool(dkronOperator.Spec.DkronConfig.DisableUsageStats),
	}
	if bootstrapExpect > 1 {
		dkronData["retry-join"] = "[\"provider=k8s label_selector=\"k8s-app-role=dkron-server\" namespace={{ .Values.namespace }}\"]"
	}

	b, err := json.Marshal(dkronData)
	if err != nil {
		return cm, err
	}
	cm.Data = map[string]string{
		"dkron.yml": string(b),
	}
	return cm, nil
}

func BuildService(ctx context.Context, dkronOperator *operatorv1.DkronOperator) (*corev1.Service, error) {
	serv := &corev1.Service{}
	serv.Name = DKRON_SERVER_SERVICE_NAME
	serv.Namespace = dkronOperator.Spec.Namespace
	serv.Labels = map[string]string{
		"k8s-app": DKRON_SERVER_SERVICE_NAME,
	}
	serv.Spec.Selector = map[string]string{
		"k8s-app-role": DKRON_SERVER_STS_NAME,
	}
	serv.Spec.Ports = []corev1.ServicePort{
		{
			Port:       8080,
			TargetPort: intstr.FromInt(8080),
		},
	}
	return serv, nil
}

func BuildIngress(ctx context.Context, dkronOperator *operatorv1.DkronOperator) (*networkv1.Ingress, error) {
	ingress := &networkv1.Ingress{}
	ingress.Name = dkronOperator.Spec.IngressName
	ingress.Namespace = dkronOperator.Spec.Namespace
	ingress.Spec.Rules = []networkv1.IngressRule{
		{
			Host: fmt.Sprintf("%s.%s", dkronOperator.Spec.IngressName, dkronOperator.Spec.BaseDomain),
			IngressRuleValue: networkv1.IngressRuleValue{
				HTTP: &networkv1.HTTPIngressRuleValue{
					Paths: []networkv1.HTTPIngressPath{
						{
							Path:     "/",
							PathType: &PATH_TYPE_PREFIX,
							Backend: networkv1.IngressBackend{
								Service: &networkv1.IngressServiceBackend{
									Name: DKRON_SERVER_SERVICE_NAME,
									Port: networkv1.ServiceBackendPort{
										Number: 8080,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return ingress, nil
}
