package main

import (
	"sort"

	"github.com/aws/aws-sdk-go/service/ecs"
)

// ClusterSlice attaches the Sort interface to ecs.Cluster
type ClusterSlice []*ecs.Cluster

func (s ClusterSlice) Len() int {
	return len([]*ecs.Cluster(s))
}

func (s ClusterSlice) Less(i, j int) bool {
	return string(*s[i].ClusterName) < string(*s[j].ClusterName)
}

func (s ClusterSlice) Swap(i, j int) {
	s[j], s[i] = s[i], s[j]
}

// Sort is a convenience method for sorting clusters
func (s ClusterSlice) Sort() {
	sort.Sort(s)
}

// ServiceSlice attaches the Sort interface to ecs.Service
type ServiceSlice []*ecs.Service

func (s ServiceSlice) Len() int {
	return len([]*ecs.Service(s))
}

func (s ServiceSlice) Less(i, j int) bool {
	return string(*s[i].ServiceName) < string(*s[j].ServiceName)
}

func (s ServiceSlice) Swap(i, j int) {
	s[j], s[i] = s[i], s[j]
}

// Sort is a convenience method for sorting clusters
func (s ServiceSlice) Sort() {
	sort.Sort(s)
}

// ServiceEventSlice attaches the Sort interface to ecs.ServiceEvent
type ServiceEventSlice []*ecs.ServiceEvent

func (s ServiceEventSlice) Len() int {
	return len([]*ecs.ServiceEvent(s))
}

func (s ServiceEventSlice) Less(i, j int) bool {
	return s[i].CreatedAt.Before(*s[j].CreatedAt)
}

func (s ServiceEventSlice) Swap(i, j int) {
	s[j], s[i] = s[i], s[j]
}

// Sort is a convenience method for sorting clusters
func (s ServiceEventSlice) Sort() {
	sort.Sort(s)
}

// KeyValuePairSlice attaches the Sort interface to ecs.KeyValuePair
type KeyValuePairSlice []*ecs.KeyValuePair

func (s KeyValuePairSlice) Len() int {
	return len([]*ecs.KeyValuePair(s))
}

func (s KeyValuePairSlice) Less(i, j int) bool {
	return *s[i].Name < *s[j].Name
}

func (s KeyValuePairSlice) Swap(i, j int) {
	s[j], s[i] = s[i], s[j]
}

// Sort is a convenience method for sorting clusters
func (s KeyValuePairSlice) Sort() {
	sort.Sort(s)
}
