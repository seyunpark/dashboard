package model

import (
	"context"
	"fmt"
	"strings"

	resty "github.com/go-resty/resty/v2"

	log "github.com/sirupsen/logrus"

	"github.com/acornsoftlab/dashboard/pkg/config"
	"github.com/acornsoftlab/dashboard/pkg/lang"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"
	metricsV1beta1api "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

type Dashboard struct {
	context   string
	Summary   map[string]*dashboardUsage    `json:"summary"`
	Nodes     map[string]dashboardNode      `json:"nodes"`
	Workloads map[string]dashboardAvailable `json:"workloads"`
	Metrics   map[string]interface{}        `json:"metrics"`
}
type dashboardNode struct {
	Address string                     `json:"address"`
	Status  string                     `json:"status"`
	Roles   string                     `json:"role"`
	Usage   map[string]*dashboardUsage `json:"usage"`
}
type dashboardAvailable struct {
	Ready     int `json:"ready"`
	Available int `json:"available"`
}

type dashboardUsage struct {
	Allocatable int64   `json:"allocatable"`
	Usage       int64   `json:"usage"`
	Percent     float32 `json:"percent"`
}

type dashboardMetrics struct {
	Allocatable int64         `json:"allocatable"`
	DataPoints  []interface{} `json:"dataPoints"`
}

type resource struct {
	cpu     int64
	memory  int64
	storage int64
	pods    int64
}

func NewDashboard(contextName string) Dashboard {
	return Dashboard{
		context: contextName,
		Nodes:   make(map[string]dashboardNode),
	}
}

func (self *Dashboard) Get() error {

	conf := config.Value.KubeConfigs[self.context]

	apiClient, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return err
	}

	metricsClient, err := metricsclientset.NewForConfig(conf)
	if err != nil {
		return err
	}

	nodeList, err := apiClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	podList, err := apiClient.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	// init variables
	self.Workloads = make(map[string]dashboardAvailable)

	// self.Workloads.DaemonSet
	dsList, err := apiClient.AppsV1().DaemonSets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("Unable to get daemonsets (cause=%v)", err)
		return err
	}
	ready := 0
	for _, m := range dsList.Items {
		if m.Status.NumberAvailable == m.Status.NumberAvailable {
			ready += 1
		}
	}
	self.Workloads["daemonset"] = dashboardAvailable{Available: len(dsList.Items), Ready: ready}

	// self.Workloads.Deployment
	deployList, err := apiClient.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	ready = 0
	for _, m := range deployList.Items {
		if m.Status.AvailableReplicas == m.Status.ReadyReplicas {
			ready += 1
		}
	}
	self.Workloads["deployment"] = dashboardAvailable{Available: len(deployList.Items), Ready: ready}

	// self.Workloads.ReplicaSet
	rsList, err := apiClient.AppsV1().ReplicaSets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("Unable to get replicasets (cause=%v)", err)
		return err
	}
	ready = 0
	for _, m := range rsList.Items {
		if m.Status.AvailableReplicas == m.Status.ReadyReplicas {
			ready += 1
		}
	}
	self.Workloads["replicaset"] = dashboardAvailable{Available: len(rsList.Items), Ready: ready}

	// self.Workloads.StatefulSet
	sfsList, err := apiClient.AppsV1().StatefulSets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("Unable to get statefulsets (cause=%v)", err)
		return err
	}
	ready = 0
	for _, m := range sfsList.Items {
		if m.Status.Replicas == m.Status.ReadyReplicas {
			ready += 1
		}
	}
	self.Workloads["statefulset"] = dashboardAvailable{Available: len(sfsList.Items), Ready: ready}

	// self.Nodes.Address/Status/Roles 외 리소스 allocatable
	allocateTotal := resource{}

	for _, m := range nodeList.Items {
		self.Nodes[m.Name] = dashboardNode{
			Address: m.Status.Addresses[0].Address,
			Status:  findNodeStatus(m),
			Roles:   findNodeRoles(m),
			Usage: map[string]*dashboardUsage{
				"cpu":     {Allocatable: m.Status.Allocatable.Cpu().MilliValue()},
				"memory":  {Allocatable: m.Status.Allocatable.Memory().Value()},
				"storage": {Allocatable: m.Status.Allocatable.Storage().Value()},
				"pod":     {Allocatable: m.Status.Allocatable.Pods().Value()},
			},
		}

		allocateTotal.cpu = +m.Status.Allocatable.Cpu().MilliValue()
		allocateTotal.memory += m.Status.Allocatable.Memory().Value()
		allocateTotal.storage += m.Status.Allocatable.Storage().Value()
		allocateTotal.pods += m.Status.Allocatable.Pods().Value()
	}

	versionedNodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("Unable to get node metries (cause=%v)", err)
		return err
	}

	nodeMetrics := &metricsapi.NodeMetricsList{}
	err = metricsV1beta1api.Convert_v1beta1_NodeMetricsList_To_metrics_NodeMetricsList(versionedNodeMetrics, nodeMetrics, nil)
	if err != nil {
		log.Errorf("Unable to convert node metries (cause=%v)", err)
		return err
	}

	// self.Nodes.cpu/memory (리소스 Usage 입력,  Percent 계산)
	usageTotal := resource{}
	for _, m := range nodeMetrics.Items {
		nd := self.Nodes[m.Name]
		parsePercentUsage(m.Usage.Cpu().MilliValue(), nd.Usage["cpu"])
		parsePercentUsage(m.Usage.Memory().Value(), nd.Usage["memory"])
		parsePercentUsage(m.Usage.Storage().Value(), nd.Usage["storage"])

		usageTotal.cpu += m.Usage.Cpu().MilliValue()
		usageTotal.memory += m.Usage.Memory().Value()
		usageTotal.storage += m.Usage.Storage().Value()
	}

	// self.Workloads.Pods (노드별 파드 수 & running 파드 수)
	usagePods := map[string]int64{}
	ready = 0
	for _, m := range podList.Items {
		if m.Spec.NodeName != "" {
			usagePods[m.Spec.NodeName] = usagePods[m.Spec.NodeName] + 1
			if m.Status.Phase == coreV1.PodRunning {
				ready += 1
			}
		}
	}
	self.Workloads["pod"] = dashboardAvailable{Available: len(podList.Items), Ready: ready}

	// self.Nodes.Pods
	for n, p := range usagePods {
		parsePercentUsage(p, self.Nodes[n].Usage["pod"])
		usageTotal.pods += p
	}

	// self.Summary (usage percent)
	self.Summary = map[string]*dashboardUsage{
		"nodes":   {Allocatable: int64(len(nodeList.Items)), Usage: int64(len(nodeList.Items))},
		"cpu":     {Allocatable: allocateTotal.cpu, Usage: usageTotal.cpu},
		"memory":  {Allocatable: allocateTotal.memory, Usage: usageTotal.memory},
		"storage": {Allocatable: allocateTotal.storage, Usage: usageTotal.storage},
	}
	parsePercentUsages(self.Summary)

	// self.Metrics
	client := resty.New()
	_, err = client.R().
		SetHeader("Content-Type", "application/json").
		SetResult(&self.Metrics).
		Get(fmt.Sprintf("%s/api/v1/clusters/%s", config.Value.MetricsScraperUrl, self.context))

	if err != nil {
		log.Errorf("Unable to get scrapping metrics (cause=%v)", err)
	}

	return nil

}

func parsePercentUsages(usages map[string]*dashboardUsage) {
	for _, usage := range usages {
		usage.Percent = percent(usage.Usage, usage.Allocatable)
	}
}

func parsePercentUsage(v int64, usage *dashboardUsage) {
	usage.Usage = v
	usage.Percent = percent(v, usage.Allocatable)
}

func percent(a int64, b int64) float32 {
	return float32(lang.DivideRound(a, b, 4) * 100)
}

const (
	LabelNodeRolePrefix = "node-role.kubernetes.io/"
	NodeLabelRole       = "kubernetes.io/role"
)

func findNodeStatus(node coreV1.Node) string {

	for _, c := range node.Status.Conditions {
		if c.Type == coreV1.NodeReady {
			if c.Status == coreV1.ConditionTrue {
				return "Ready"
			} else {
				return "NotReady"
			}
		}
	}
	return "Unknown"
}

// findNodeRoles returns the roles of a given node.
// The roles are determined by looking for:
// * a node-role.kubernetes.io/<role>="" label
// * a kubernetes.io/role="<role>" label
func findNodeRoles(node coreV1.Node) string {
	roleList := sets.NewString()
	for k, v := range node.Labels {
		switch {
		case strings.HasPrefix(k, LabelNodeRolePrefix):
			if role := strings.TrimPrefix(k, LabelNodeRolePrefix); len(role) > 0 {
				roleList.Insert(role)
			}

		case k == NodeLabelRole && v != "":
			roleList.Insert(v)
		}
	}

	if len(roleList) > 0 {
		return strings.Join(roleList.List(), ",")
	} else {
		return "<none>"
	}

}
