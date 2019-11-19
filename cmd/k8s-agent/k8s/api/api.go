package api

import (
	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubernetesAPI interface for communicating with kubernetes cluster
//go:generate counterfeiter . KubernetesAPI
type KubernetesAPI interface {
	// CreateClusterServiceBroker creates cluster-wide visible service broker
	CreateClusterServiceBroker(broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error)
	// DeleteClusterServiceBroker deletes cluster-wide visible service broker
	DeleteClusterServiceBroker(name string, options *v1.DeleteOptions) error
	// RetrieveClusterServiceBrokers gets all cluster-wide visible service brokers
	RetrieveClusterServiceBrokers() (*v1beta1.ClusterServiceBrokerList, error)
	// RetrieveClusterServiceBrokerByName gets cluster-wide visible service broker
	RetrieveClusterServiceBrokerByName(name string) (*v1beta1.ClusterServiceBroker, error)
	// UpdateClusterServiceBroker gets cluster-wide visible service broker
	UpdateClusterServiceBroker(broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error)
	// SyncClusterServiceBroker synchronize a cluster-wide visible service broker
	SyncClusterServiceBroker(name string, retries int) error
}
