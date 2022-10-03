// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	conventions "go.opentelemetry.io/collector/semconv/v1.6.1"
)

type jobInstanceDefinition struct {
	job, instance, host, scheme, port string
}

type k8sResourceDefinition struct {
	podName, podUID, container, node, rs, ds, ss, job, cronjob, ns string
}

func makeK8sResource(jobInstance *jobInstanceDefinition, def *k8sResourceDefinition) pcommon.Resource {
	resource := makeResourceWithJobInstanceScheme(jobInstance, true)
	attrs := resource.Attributes()
	if def.podName != "" {
		attrs.UpsertString(conventions.AttributeK8SPodName, def.podName)
	}
	if def.podUID != "" {
		attrs.UpsertString(conventions.AttributeK8SPodUID, def.podUID)
	}
	if def.container != "" {
		attrs.UpsertString(conventions.AttributeK8SContainerName, def.container)
	}
	if def.node != "" {
		attrs.UpsertString(conventions.AttributeK8SNodeName, def.node)
	}
	if def.rs != "" {
		attrs.UpsertString(conventions.AttributeK8SReplicaSetName, def.rs)
	}
	if def.ds != "" {
		attrs.UpsertString(conventions.AttributeK8SDaemonSetName, def.ds)
	}
	if def.ss != "" {
		attrs.UpsertString(conventions.AttributeK8SStatefulSetName, def.ss)
	}
	if def.job != "" {
		attrs.UpsertString(conventions.AttributeK8SJobName, def.job)
	}
	if def.cronjob != "" {
		attrs.UpsertString(conventions.AttributeK8SCronJobName, def.cronjob)
	}
	if def.ns != "" {
		attrs.UpsertString(conventions.AttributeK8SNamespaceName, def.ns)
	}
	return resource
}

func makeResourceWithJobInstanceScheme(def *jobInstanceDefinition, hasHost bool) pcommon.Resource {
	resource := pcommon.NewResource()
	attrs := resource.Attributes()
	// Using hardcoded values to assert on outward expectations so that
	// when variables change, these tests will fail and we'll have reports.
	attrs.UpsertString("service.name", def.job)
	if hasHost {
		attrs.UpsertString("net.host.name", def.host)
	}
	attrs.UpsertString("service.instance.id", def.instance)
	attrs.UpsertString("net.host.port", def.port)
	attrs.UpsertString("http.scheme", def.scheme)
	return resource
}

func TestCreateNodeAndResourcePromToOTLP(t *testing.T) {
	tests := []struct {
		name, job string
		instance  string
		sdLabels  labels.Labels
		want      pcommon.Resource
	}{
		{
			name: "all attributes proper",
			job:  "job", instance: "hostname:8888", sdLabels: labels.New(labels.Label{Name: "__scheme__", Value: "http"}),
			want: makeResourceWithJobInstanceScheme(&jobInstanceDefinition{
				"job", "hostname:8888", "hostname", "http", "8888",
			}, true),
		},
		{
			name: "missing port",
			job:  "job", instance: "myinstance", sdLabels: labels.New(labels.Label{Name: "__scheme__", Value: "https"}),
			want: makeResourceWithJobInstanceScheme(&jobInstanceDefinition{
				"job", "myinstance", "myinstance", "https", "",
			}, true),
		},
		{
			name: "blank scheme",
			job:  "job", instance: "myinstance:443", sdLabels: labels.New(labels.Label{Name: "__scheme__", Value: ""}),
			want: makeResourceWithJobInstanceScheme(&jobInstanceDefinition{
				"job", "myinstance:443", "myinstance", "", "443",
			}, true),
		},
		{
			name: "blank instance, blank scheme",
			job:  "job", instance: "", sdLabels: labels.New(labels.Label{Name: "__scheme__", Value: ""}),
			want: makeResourceWithJobInstanceScheme(&jobInstanceDefinition{
				"job", "", "", "", "",
			}, true),
		},
		{
			name: "blank instance, non-blank scheme",
			job:  "job", instance: "", sdLabels: labels.New(labels.Label{Name: "__scheme__", Value: "http"}),
			want: makeResourceWithJobInstanceScheme(&jobInstanceDefinition{
				"job", "", "", "http", "",
			}, true),
		},
		{
			name: "0.0.0.0 address",
			job:  "job", instance: "0.0.0.0:8888", sdLabels: labels.New(labels.Label{Name: "__scheme__", Value: "http"}),
			want: makeResourceWithJobInstanceScheme(&jobInstanceDefinition{
				"job", "0.0.0.0:8888", "", "http", "8888",
			}, false),
		},
		{
			name: "localhost",
			job:  "job", instance: "localhost:8888", sdLabels: labels.New(labels.Label{Name: "__scheme__", Value: "http"}),
			want: makeResourceWithJobInstanceScheme(&jobInstanceDefinition{
				"job", "localhost:8888", "", "http", "8888",
			}, false),
		},
		{
			name: "kubernetes daemonset pod",
			job:  "job", instance: "hostname:8888", sdLabels: labels.New(
				labels.Label{Name: "__scheme__", Value: "http"},
				labels.Label{Name: "__meta_kubernetes_pod_name", Value: "my-pod-23491"},
				labels.Label{Name: "__meta_kubernetes_pod_uid", Value: "84279wretgu89dg489q2"},
				labels.Label{Name: "__meta_kubernetes_pod_container_name", Value: "my-container"},
				labels.Label{Name: "__meta_kubernetes_pod_node_name", Value: "k8s-node-123"},
				labels.Label{Name: "__meta_kubernetes_pod_controller_name", Value: "my-pod"},
				labels.Label{Name: "__meta_kubernetes_pod_controller_kind", Value: "DaemonSet"},
				labels.Label{Name: "__meta_kubernetes_namespace", Value: "kube-system"},
			),
			want: makeK8sResource(&jobInstanceDefinition{
				"job", "hostname:8888", "hostname", "http", "8888",
			}, &k8sResourceDefinition{
				podName:   "my-pod-23491",
				podUID:    "84279wretgu89dg489q2",
				container: "my-container",
				node:      "k8s-node-123",
				ds:        "my-pod",
				ns:        "kube-system",
			}),
		},
		{
			name: "kubernetes replicaset pod",
			job:  "job", instance: "hostname:8888", sdLabels: labels.New(
				labels.Label{Name: "__scheme__", Value: "http"},
				labels.Label{Name: "__meta_kubernetes_pod_name", Value: "my-pod-23491"},
				labels.Label{Name: "__meta_kubernetes_pod_uid", Value: "84279wretgu89dg489q2"},
				labels.Label{Name: "__meta_kubernetes_pod_container_name", Value: "my-container"},
				labels.Label{Name: "__meta_kubernetes_pod_node_name", Value: "k8s-node-123"},
				labels.Label{Name: "__meta_kubernetes_pod_controller_name", Value: "my-pod"},
				labels.Label{Name: "__meta_kubernetes_pod_controller_kind", Value: "ReplicaSet"},
				labels.Label{Name: "__meta_kubernetes_namespace", Value: "kube-system"},
			),
			want: makeK8sResource(&jobInstanceDefinition{
				"job", "hostname:8888", "hostname", "http", "8888",
			}, &k8sResourceDefinition{
				podName:   "my-pod-23491",
				podUID:    "84279wretgu89dg489q2",
				container: "my-container",
				node:      "k8s-node-123",
				rs:        "my-pod",
				ns:        "kube-system",
			}),
		},
		{
			name: "kubernetes statefulset pod",
			job:  "job", instance: "hostname:8888", sdLabels: labels.New(
				labels.Label{Name: "__scheme__", Value: "http"},
				labels.Label{Name: "__meta_kubernetes_pod_name", Value: "my-pod-23491"},
				labels.Label{Name: "__meta_kubernetes_pod_uid", Value: "84279wretgu89dg489q2"},
				labels.Label{Name: "__meta_kubernetes_pod_container_name", Value: "my-container"},
				labels.Label{Name: "__meta_kubernetes_pod_node_name", Value: "k8s-node-123"},
				labels.Label{Name: "__meta_kubernetes_pod_controller_name", Value: "my-pod"},
				labels.Label{Name: "__meta_kubernetes_pod_controller_kind", Value: "StatefulSet"},
				labels.Label{Name: "__meta_kubernetes_namespace", Value: "kube-system"},
			),
			want: makeK8sResource(&jobInstanceDefinition{
				"job", "hostname:8888", "hostname", "http", "8888",
			}, &k8sResourceDefinition{
				podName:   "my-pod-23491",
				podUID:    "84279wretgu89dg489q2",
				container: "my-container",
				node:      "k8s-node-123",
				ss:        "my-pod",
				ns:        "kube-system",
			}),
		},
		{
			name: "kubernetes job pod",
			job:  "job", instance: "hostname:8888", sdLabels: labels.New(
				labels.Label{Name: "__scheme__", Value: "http"},
				labels.Label{Name: "__meta_kubernetes_pod_name", Value: "my-pod-23491"},
				labels.Label{Name: "__meta_kubernetes_pod_uid", Value: "84279wretgu89dg489q2"},
				labels.Label{Name: "__meta_kubernetes_pod_container_name", Value: "my-container"},
				labels.Label{Name: "__meta_kubernetes_pod_node_name", Value: "k8s-node-123"},
				labels.Label{Name: "__meta_kubernetes_pod_controller_name", Value: "my-pod"},
				labels.Label{Name: "__meta_kubernetes_pod_controller_kind", Value: "Job"},
				labels.Label{Name: "__meta_kubernetes_namespace", Value: "kube-system"},
			),
			want: makeK8sResource(&jobInstanceDefinition{
				"job", "hostname:8888", "hostname", "http", "8888",
			}, &k8sResourceDefinition{
				podName:   "my-pod-23491",
				podUID:    "84279wretgu89dg489q2",
				container: "my-container",
				node:      "k8s-node-123",
				job:       "my-pod",
				ns:        "kube-system",
			}),
		},
		{
			name: "kubernetes cronjob pod",
			job:  "job", instance: "hostname:8888", sdLabels: labels.New(
				labels.Label{Name: "__scheme__", Value: "http"},
				labels.Label{Name: "__meta_kubernetes_pod_name", Value: "my-pod-23491"},
				labels.Label{Name: "__meta_kubernetes_pod_uid", Value: "84279wretgu89dg489q2"},
				labels.Label{Name: "__meta_kubernetes_pod_container_name", Value: "my-container"},
				labels.Label{Name: "__meta_kubernetes_pod_node_name", Value: "k8s-node-123"},
				labels.Label{Name: "__meta_kubernetes_pod_controller_name", Value: "my-pod"},
				labels.Label{Name: "__meta_kubernetes_pod_controller_kind", Value: "CronJob"},
				labels.Label{Name: "__meta_kubernetes_namespace", Value: "kube-system"},
			),
			want: makeK8sResource(&jobInstanceDefinition{
				"job", "hostname:8888", "hostname", "http", "8888",
			}, &k8sResourceDefinition{
				podName:   "my-pod-23491",
				podUID:    "84279wretgu89dg489q2",
				container: "my-container",
				node:      "k8s-node-123",
				cronjob:   "my-pod",
				ns:        "kube-system",
			}),
		},
		{
			name: "kubernetes node (e.g. kubelet)",
			job:  "job", instance: "hostname:8888", sdLabels: labels.New(
				labels.Label{Name: "__scheme__", Value: "http"},
				labels.Label{Name: "__meta_kubernetes_node_name", Value: "k8s-node-123"},
			),
			want: makeK8sResource(&jobInstanceDefinition{
				"job", "hostname:8888", "hostname", "http", "8888",
			}, &k8sResourceDefinition{
				node: "k8s-node-123",
			}),
		},
		{
			name: "kubernetes service endpoint",
			job:  "job", instance: "hostname:8888", sdLabels: labels.New(
				labels.Label{Name: "__scheme__", Value: "http"},
				labels.Label{Name: "__meta_kubernetes_endpoint_node_name", Value: "k8s-node-123"},
			),
			want: makeK8sResource(&jobInstanceDefinition{
				"job", "hostname:8888", "hostname", "http", "8888",
			}, &k8sResourceDefinition{
				node: "k8s-node-123",
			}),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := CreateResource(tt.job, tt.instance, tt.sdLabels)
			got.Attributes().Sort()
			tt.want.Attributes().Sort()
			require.Equal(t, tt.want, got)
		})
	}
}
