package provider

import (
	"time"

	"k8s.io/apimachinery/pkg/types"
)

// ResourceKind is an unique name for each resource. It can used for API discovery and generic
// code that does things based on the kind. For example, there may be a generic "deleter"
// that based on resource kind, name and namespace deletes it.
type ResourceKind string

type DataPoints []DataPoint

type DataPoint struct {
	X int64 `json:"x"`
	Y int64 `json:"y"`
}

type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     uint64    `json:"value"`
}

// ResourceSelector is a structure used to quickly and uniquely identify given resource.
// This struct can be later used for heapster data download etc.
type ResourceSelector struct {
	// Namespace of this resource.
	Namespace string
	// Type of this resource
	ResourceType ResourceKind
	// Name of this resource.
	ResourceName string
	// Selector used to identify this resource (should be used only for Deployments!).
	Selector map[string]string
	// UID is resource unique identifier.
	UID types.UID
}

// SidecarMetric is a format of data used by our sidecar. This is also the format of data that is being sent by backend API.
type SidecarMetric struct {
	// DataPoints is a list of X, Y int64 data points, sorted by X.
	DataPoints []DataPoint `json:"dataPoints"` // modified by acornsoft-dashboard
	// MetricPoints is a list of value, timestamp metrics used for sparklines on a pod list page.
	MetricPoints []MetricPoint `json:"metricPoints"`
	// MetricName is the name of metric stored in this struct.
	MetricName string `json:"metricName"`
	// Label stores information about identity of resources (UIDS) described by this metric.
	UIDs []types.UID `json:"uids"`
}

func (metric *SidecarMetric) AddMetricPoint(item MetricPoint) []MetricPoint {
	metric.MetricPoints = append(metric.MetricPoints, item)
	return metric.MetricPoints
}

type SidecarMetricResultList struct {
	Items []SidecarMetric `json:"items"`
}
