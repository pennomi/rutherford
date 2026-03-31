package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/metadata/metadatainformer"
	"k8s.io/client-go/tools/cache"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// IMPORTANT: The backend's job is to forward the most unmodified data possible.
// All computation, aggregation, and breakdown logic belongs in the frontend.

type kubeletSummary struct {
	Node kubeletNodeStats  `json:"node"`
	Pods []kubeletPodStats `json:"pods"`
}

type kubeletNodeStats struct {
	Fs      kubeletFsStats      `json:"fs"`
	Runtime kubeletRuntimeStats `json:"runtime"`
}

type kubeletRuntimeStats struct {
	ImageFs kubeletFsStats `json:"imageFs"`
}

type kubeletFsStats struct {
	CapacityBytes int64 `json:"capacityBytes"`
	UsedBytes     int64 `json:"usedBytes"`
}

type kubeletPodStats struct {
	PodRef struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"podRef"`
	VolumeStats []kubeletVolumeStats `json:"volume"`
}

type kubeletVolumeStats struct {
	Name   string `json:"name"`
	PvcRef *struct {
		Name string `json:"name"`
	} `json:"pvcRef"`
	CapacityBytes int64 `json:"capacityBytes"`
	UsedBytes     int64 `json:"usedBytes"`
}

type NodeDiskStats struct {
	Kind     string        `json:"kind"`
	Metadata SimpleMeta    `json:"metadata"`
	NodeFs   DiskUsage     `json:"nodeFs"`
	ImageFs  DiskUsage     `json:"imageFs"`
	Volumes  []VolumeUsage `json:"volumes"`
}

type SimpleMeta struct {
	Name string `json:"name"`
}

type DiskUsage struct {
	CapacityBytes int64 `json:"capacityBytes"`
	UsedBytes     int64 `json:"usedBytes"`
}

type VolumeUsage struct {
	PodName       string `json:"podName"`
	PodNamespace  string `json:"podNamespace"`
	VolumeName    string `json:"volumeName"`
	PVCName       string `json:"pvcName"`
	CapacityBytes int64  `json:"capacityBytes"`
	UsedBytes     int64  `json:"usedBytes"`
}

type resourceDef struct {
	kind    string
	indexer cache.Indexer
}

type Watcher struct {
	clientset      *kubernetes.Clientset
	metricsClient  *metricsv.Clientset
	hub            *Hub
	resources      []resourceDef
	metricsHistory [][]byte
	metricsMu      sync.RWMutex
}

var secretGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}

func NewWatcher(clientset *kubernetes.Clientset, metricsClient *metricsv.Clientset, metadataClient metadata.Interface, hub *Hub) *Watcher {
	w := &Watcher{
		clientset:     clientset,
		metricsClient: metricsClient,
		hub:           hub,
	}

	factory := informers.NewSharedInformerFactory(clientset, 0)
	metaFactory := metadatainformer.NewSharedInformerFactory(metadataClient, 0)

	type informerSpec struct {
		kind     string
		informer cache.SharedIndexInformer
	}

	specs := []informerSpec{
		{"Namespace", factory.Core().V1().Namespaces().Informer()},
		{"Pod", factory.Core().V1().Pods().Informer()},
		{"Node", factory.Core().V1().Nodes().Informer()},
		{"PersistentVolumeClaim", factory.Core().V1().PersistentVolumeClaims().Informer()},
		{"PersistentVolume", factory.Core().V1().PersistentVolumes().Informer()},
		{"Event", factory.Core().V1().Events().Informer()},
		{"Service", factory.Core().V1().Services().Informer()},
		{"ConfigMap", factory.Core().V1().ConfigMaps().Informer()},
		{"ServiceAccount", factory.Core().V1().ServiceAccounts().Informer()},
		{"EndpointSlice", factory.Discovery().V1().EndpointSlices().Informer()},
		{"Deployment", factory.Apps().V1().Deployments().Informer()},
		{"ReplicaSet", factory.Apps().V1().ReplicaSets().Informer()},
		{"StatefulSet", factory.Apps().V1().StatefulSets().Informer()},
		{"DaemonSet", factory.Apps().V1().DaemonSets().Informer()},
		{"Job", factory.Batch().V1().Jobs().Informer()},
		{"CronJob", factory.Batch().V1().CronJobs().Informer()},
		{"Ingress", factory.Networking().V1().Ingresses().Informer()},
		{"IngressClass", factory.Networking().V1().IngressClasses().Informer()},
		{"NetworkPolicy", factory.Networking().V1().NetworkPolicies().Informer()},
		{"StorageClass", factory.Storage().V1().StorageClasses().Informer()},
		{"ClusterRole", factory.Rbac().V1().ClusterRoles().Informer()},
		{"ClusterRoleBinding", factory.Rbac().V1().ClusterRoleBindings().Informer()},
		{"Role", factory.Rbac().V1().Roles().Informer()},
		{"RoleBinding", factory.Rbac().V1().RoleBindings().Informer()},
		{"Secret", metaFactory.ForResource(secretGVR).Informer()},
	}

	for _, s := range specs {
		kind := s.kind
		s.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				w.broadcast("ADDED", kind, obj)
			},
			UpdateFunc: func(_, obj interface{}) {
				w.broadcast("MODIFIED", kind, obj)
			},
			DeleteFunc: func(obj interface{}) {
				if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
					obj = tombstone.Obj
				}
				w.broadcast("DELETED", kind, obj)
			},
		})
		w.resources = append(w.resources, resourceDef{kind: kind, indexer: s.informer.GetIndexer()})
	}

	factory.Start(nil)
	metaFactory.Start(nil)
	factory.WaitForCacheSync(nil)
	metaFactory.WaitForCacheSync(nil)

	return w
}

func (w *Watcher) Start(ctx context.Context) {
	go w.pollMetrics(ctx)
}

func prepareObj(kind string, obj interface{}) {
	if ro, ok := obj.(runtime.Object); ok {
		ro.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Kind: kind})
	}
	if accessor, ok := obj.(metav1.ObjectMetaAccessor); ok {
		accessor.GetObjectMeta().SetManagedFields(nil)
	}
}

func (w *Watcher) broadcast(eventType, kind string, obj interface{}) {
	prepareObj(kind, obj)
	raw, err := json.Marshal(watchEvent{
		Type:   eventType,
		Object: marshalRaw(obj),
	})
	if err != nil {
		return
	}
	w.hub.Broadcast(raw)
}

func (w *Watcher) Snapshot(ctx context.Context) ([][]byte, error) {
	start := time.Now()
	var messages [][]byte

	for _, r := range w.resources {
		t := time.Now()
		items := r.indexer.List()
		for _, obj := range items {
			prepareObj(r.kind, obj)
			messages = appendEvent(messages, obj)
		}
		if d := time.Since(t); d > 10*time.Millisecond {
			log.Printf("[snapshot] %s: %v (%d items)", r.kind, d, len(items))
		}
	}

	w.metricsMu.RLock()
	messages = append(messages, w.metricsHistory...)
	w.metricsMu.RUnlock()

	log.Printf("[snapshot] TOTAL: %v (%d messages)", time.Since(start), len(messages))
	return messages, nil
}

type MetricsSnapshot struct {
	Kind      string          `json:"kind"`
	Timestamp string          `json:"timestamp"`
	Nodes     json.RawMessage `json:"nodes"`
	Pods      json.RawMessage `json:"pods"`
	DiskStats []NodeDiskStats `json:"diskStats"`
}

const metricsHistorySize = 10

func (w *Watcher) pollMetrics(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	w.fetchAndBroadcastMetrics(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.fetchAndBroadcastMetrics(ctx)
		}
	}
}

func (w *Watcher) fetchAndBroadcastMetrics(ctx context.Context) {
	snapshot := MetricsSnapshot{
		Kind:      "MetricsSnapshot",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	nodeMetrics, err := w.metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}
	snapshot.Nodes = marshalRaw(&nodeMetrics.Items)

	podMetrics, err := w.metricsClient.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}
	snapshot.Pods = marshalRaw(&podMetrics.Items)

	nodes, err := w.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}
	for _, node := range nodes.Items {
		path := fmt.Sprintf("/api/v1/nodes/%s/proxy/stats/summary", node.Name)
		rawBytes, err := w.clientset.RESTClient().Get().AbsPath(path).DoRaw(ctx)
		if err != nil {
			continue
		}
		var summary kubeletSummary
		if json.Unmarshal(rawBytes, &summary) != nil {
			continue
		}
		snapshot.DiskStats = append(snapshot.DiskStats, buildDiskStats(node.Name, summary))
	}

	raw, err := json.Marshal(watchEvent{
		Type:   "ADDED",
		Object: marshalRaw(&snapshot),
	})
	if err != nil {
		return
	}

	w.metricsMu.Lock()
	w.metricsHistory = append(w.metricsHistory, raw)
	if len(w.metricsHistory) > metricsHistorySize {
		w.metricsHistory = w.metricsHistory[len(w.metricsHistory)-metricsHistorySize:]
	}
	w.metricsMu.Unlock()

	w.hub.Broadcast(raw)
}

func buildDiskStats(nodeName string, summary kubeletSummary) NodeDiskStats {
	stats := NodeDiskStats{
		Kind:     "NodeDiskStats",
		Metadata: SimpleMeta{Name: nodeName},
		NodeFs: DiskUsage{
			CapacityBytes: summary.Node.Fs.CapacityBytes,
			UsedBytes:     summary.Node.Fs.UsedBytes,
		},
		ImageFs: DiskUsage{
			CapacityBytes: summary.Node.Runtime.ImageFs.CapacityBytes,
			UsedBytes:     summary.Node.Runtime.ImageFs.UsedBytes,
		},
	}

	for _, pod := range summary.Pods {
		for _, vol := range pod.VolumeStats {
			if vol.PvcRef == nil {
				continue
			}
			stats.Volumes = append(stats.Volumes, VolumeUsage{
				PodName:       pod.PodRef.Name,
				PodNamespace:  pod.PodRef.Namespace,
				VolumeName:    vol.Name,
				PVCName:       vol.PvcRef.Name,
				CapacityBytes: vol.CapacityBytes,
				UsedBytes:     vol.UsedBytes,
			})
		}
	}

	return stats
}

func appendEvent(messages [][]byte, obj interface{}) [][]byte {
	raw, err := json.Marshal(watchEvent{
		Type:   "ADDED",
		Object: marshalRaw(obj),
	})
	if err != nil {
		return messages
	}
	return append(messages, raw)
}

func marshalRaw(obj interface{}) json.RawMessage {
	data, err := json.Marshal(obj)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return data
}
