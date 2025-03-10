// Copyright 2019 Authors of Hubble
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cilium/cilium/api/v1/models"
	monitorAPI "github.com/cilium/cilium/pkg/monitor/api"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	v1 "github.com/cilium/hubble/pkg/api/v1"
)

type fakeCiliumClient struct {
	fakeEndpointList func() ([]*models.Endpoint, error)
	fakeGetEndpoint  func(uint64) (*models.Endpoint, error)
	fakeGetIdentity  func(uint64) (*models.Identity, error)
	fakeGetFqdnCache func() ([]*models.DNSLookup, error)
	fakeGetIPCache   func() ([]*models.IPListEntry, error)
}

func (c *fakeCiliumClient) EndpointList() ([]*models.Endpoint, error) {
	if c.fakeEndpointList != nil {
		return c.fakeEndpointList()
	}
	panic("EndpointList() should not have been called since it was not defined")
}

func (c *fakeCiliumClient) GetEndpoint(id uint64) (*models.Endpoint, error) {
	if c.fakeGetEndpoint != nil {
		return c.fakeGetEndpoint(id)
	}
	panic("GetEndpoint(uint64) should not have been called since it was not defined")
}

func (c *fakeCiliumClient) GetIdentity(id uint64) (*models.Identity, error) {
	if c.fakeGetIdentity != nil {
		return c.fakeGetIdentity(id)
	}
	panic("GetIdentity(uint64) should not have been called since it was not defined")
}

func (c *fakeCiliumClient) GetFqdnCache() ([]*models.DNSLookup, error) {
	if c.fakeGetFqdnCache != nil {
		return c.fakeGetFqdnCache()
	}
	panic("GetFqdnCache() should not have been called since it was not defined")
}

func (c *fakeCiliumClient) GetIPCache() ([]*models.IPListEntry, error) {
	if c.fakeGetIPCache != nil {
		return c.fakeGetIPCache()
	}
	panic("GetIPCache() should not have been called since it was not defined")
}

var fakeDummyCiliumClient = &fakeCiliumClient{
	fakeEndpointList: func() (endpoints []*models.Endpoint, e error) {
		return nil, nil
	},
	fakeGetEndpoint: func(u uint64) (endpoint *models.Endpoint, e error) {
		return nil, nil
	},
	fakeGetIdentity: func(u uint64) (endpoint *models.Identity, e error) {
		return &models.Identity{}, nil
	},
	fakeGetFqdnCache: func() ([]*models.DNSLookup, error) {
		return nil, nil
	},
}

type fakeEndpointsHandler struct {
	fakeSyncEndpoints  func([]*v1.Endpoint)
	fakeUpdateEndpoint func(*v1.Endpoint)
	fakeMarkDeleted    func(*v1.Endpoint)
	fakeFindEPs        func(epID uint64, ns, pod string) []v1.Endpoint
	fakeGetEndpoint    func(ip net.IP) (endpoint *v1.Endpoint, ok bool)
}

func (f *fakeEndpointsHandler) SyncEndpoints(eps []*v1.Endpoint) {
	if f.fakeSyncEndpoints != nil {
		f.fakeSyncEndpoints(eps)
		return
	}
	panic("SyncEndpoints([]*v1.Endpoint) should not have been called since it was not defined")
}

func (f *fakeEndpointsHandler) UpdateEndpoint(ep *v1.Endpoint) {
	if f.fakeUpdateEndpoint != nil {
		f.fakeUpdateEndpoint(ep)
		return
	}
	panic("UpdateEndpoint(*v1.Endpoint) should not have been called since it was not defined")
}

func (f *fakeEndpointsHandler) MarkDeleted(ep *v1.Endpoint) {
	if f.fakeMarkDeleted != nil {
		f.fakeMarkDeleted(ep)
		return
	}
	panic("MarkDeleted(ep *v1.Endpoint) should not have been called since it was not defined")
}

func (f *fakeEndpointsHandler) FindEPs(epID uint64, ns, pod string) []v1.Endpoint {
	if f.fakeFindEPs != nil {
		return f.fakeFindEPs(epID, ns, pod)
	}
	panic(" FindEPs(epID uint64, ns, pod string) should not have been called since it was not defined")
}

func (f *fakeEndpointsHandler) GetEndpoint(ip net.IP) (ep *v1.Endpoint, ok bool) {
	if f.fakeGetEndpoint != nil {
		return f.fakeGetEndpoint(ip)
	}
	panic("GetEndpoint(ip net.IP) (ep *v1.Endpoint, ok bool) should not have been called since it was not defined")
}

func TestObserverServer_syncAllEndpoints(t *testing.T) {
	refreshEndpointList = 50 * time.Millisecond
	var (
		returnEmptyEndpoints int32
		endpointsMutex       sync.RWMutex
		endpoints            []*v1.Endpoint
	)

	fakeClient := &fakeCiliumClient{
		fakeEndpointList: func() ([]*models.Endpoint, error) {
			if atomic.LoadInt32(&returnEmptyEndpoints) != 0 {
				return []*models.Endpoint{}, nil
			}
			eps := []*models.Endpoint{
				{
					ID: 1,
					Status: &models.EndpointStatus{
						ExternalIdentifiers: &models.EndpointIdentifiers{
							ContainerID: "313c63b8b164a19ec0fe42cd86c4159f3276ba8a415d77f340817fcfee2cb479",
							PodName:     "default/foo",
						},
						Networking: &models.EndpointNetworking{
							Addressing: []*models.AddressPair{
								{
									IPV4: "1.1.1.1",
									IPV6: "fd00::1",
								},
							},
						},
					},
				},
				{
					ID: 2,
					Status: &models.EndpointStatus{
						ExternalIdentifiers: &models.EndpointIdentifiers{
							ContainerID: "313c63b8b164a19ec0fe42cd86c4159f3276ba8a415d77f340817fcfee2cb471",
							PodName:     "default/bar",
						},
						Networking: &models.EndpointNetworking{
							Addressing: []*models.AddressPair{
								{
									IPV4: "1.1.1.2",
									IPV6: "fd00::2",
								},
							},
						},
					},
				},
			}
			return eps, nil
		},
	}

	fakeHandler := &fakeEndpointsHandler{
		fakeSyncEndpoints: func(newEndpoint []*v1.Endpoint) {
			if len(newEndpoint) == 0 {
				now := time.Now()
				endpointsMutex.Lock()
				for _, ep := range endpoints {
					ep.Deleted = &now
				}
				endpointsMutex.Unlock()
			}
		},
		fakeUpdateEndpoint: func(ep *v1.Endpoint) {
			endpointsMutex.Lock()
			endpoints = append(endpoints, ep)
			endpointsMutex.Unlock()
		},
	}
	s := &ObserverServer{
		ciliumClient: fakeClient,
		endpoints:    fakeHandler,
		log:          zap.L(),
	}
	go s.syncEndpoints()

	time.Sleep(2 * refreshEndpointList)

	endpointsWanted := []*v1.Endpoint{
		{
			ContainerIDs: []string{"313c63b8b164a19ec0fe42cd86c4159f3276ba8a415d77f340817fcfee2cb479"},
			ID:           1,
			Created:      time.Unix(0, 0),
			IPv4:         net.ParseIP("1.1.1.1").To4(),
			IPv6:         net.ParseIP("fd00::1").To16(),
			PodName:      "foo",
			PodNamespace: "default",
		},
		{
			ContainerIDs: []string{"313c63b8b164a19ec0fe42cd86c4159f3276ba8a415d77f340817fcfee2cb471"},
			ID:           2,
			Created:      time.Unix(0, 0),
			IPv4:         net.ParseIP("1.1.1.2").To4(),
			IPv6:         net.ParseIP("fd00::2").To16(),
			PodName:      "bar",
			PodNamespace: "default",
		},
	}
	endpointsMutex.Lock()
	for _, ep := range endpoints {
		ep.Created = time.Unix(0, 0)
	}
	assert.EqualValues(t, endpoints, endpointsWanted)
	endpointsMutex.Unlock()

	// stop returning any endpoints so all of them will be marked as deleted
	atomic.StoreInt32(&returnEmptyEndpoints, 1)
	time.Sleep(2 * refreshEndpointList)
	endpointsWanted = []*v1.Endpoint{
		{
			ContainerIDs: []string{"313c63b8b164a19ec0fe42cd86c4159f3276ba8a415d77f340817fcfee2cb479"},
			ID:           1,
			Created:      time.Unix(0, 0),
			IPv4:         net.ParseIP("1.1.1.1").To4(),
			IPv6:         net.ParseIP("fd00::1").To16(),
			PodName:      "foo",
			PodNamespace: "default",
		},
		{
			ContainerIDs: []string{"313c63b8b164a19ec0fe42cd86c4159f3276ba8a415d77f340817fcfee2cb471"},
			ID:           2,
			Created:      time.Unix(0, 0),
			IPv4:         net.ParseIP("1.1.1.2").To4(),
			IPv6:         net.ParseIP("fd00::2").To16(),
			PodName:      "bar",
			PodNamespace: "default",
		},
	}
	endpointsMutex.Lock()
	for _, ep := range endpoints {
		assert.NotNil(t, ep.Deleted)
		ep.Created = time.Unix(0, 0)
		ep.Deleted = nil
	}
	assert.EqualValues(t, endpoints, endpointsWanted)
	endpointsMutex.Unlock()
}

func TestObserverServer_consumeEpAddEvents(t *testing.T) {
	once := sync.Once{}
	wg := sync.WaitGroup{}
	wg.Add(2)
	ecn := &monitorAPI.EndpointCreateNotification{
		EndpointRegenNotification: monitorAPI.EndpointRegenNotification{
			ID: 13,
		},
	}
	ecnMarshal, err := json.Marshal(ecn)
	assert.Nil(t, err)
	fakeClient := &fakeCiliumClient{
		fakeGetEndpoint: func(epID uint64) (*models.Endpoint, error) {
			defer wg.Done()
			assert.Equal(t, uint64(13), epID)
			return &models.Endpoint{
				ID: 13,
				Status: &models.EndpointStatus{
					ExternalIdentifiers: &models.EndpointIdentifiers{
						ContainerID: "123",
						PodName:     "default/bar",
					},
					Networking: &models.EndpointNetworking{
						Addressing: []*models.AddressPair{
							{
								IPV4: "10.0.0.1",
								IPV6: "fd00::1",
							},
						},
					},
				},
			}, nil
		},
	}
	fakeHandler := &fakeEndpointsHandler{
		fakeUpdateEndpoint: func(ep *v1.Endpoint) {
			once.Do(func() {
				defer wg.Done()
				wanted := &v1.Endpoint{
					ContainerIDs: []string{"123"},
					Created:      time.Unix(12, 34),
					ID:           13,
					IPv4:         net.ParseIP("10.0.0.1").To4(),
					IPv6:         net.ParseIP("fd00::1").To16(),
					PodName:      "bar",
					PodNamespace: "default",
				}
				ep.Created = time.Unix(12, 34)
				assert.Equal(t, wanted, ep)
			})
		},
	}
	epAddCh := make(chan string, 1)
	s := &ObserverServer{
		ciliumClient: fakeClient,
		endpoints:    fakeHandler,
		epAdd:        epAddCh,
		log:          zap.L(),
	}
	go s.consumeEpAddEvents()

	s.GetEpAddChannel() <- string(ecnMarshal)
	wg.Wait()

	// Endpoint is not found so we don't even add it to the list of endpoints
	wg = sync.WaitGroup{}
	fakeClient = &fakeCiliumClient{
		fakeGetEndpoint: func(epID uint64) (*models.Endpoint, error) {
			defer wg.Done()
			assert.Equal(t, uint64(13), epID)
			return nil, nil
		},
	}
	wg.Add(1)
	s = &ObserverServer{
		ciliumClient: fakeClient,
		endpoints:    fakeHandler,
		epAdd:        epAddCh,
		log:          zap.L(),
	}
	go s.consumeEpAddEvents()

	s.GetEpAddChannel() <- string(ecnMarshal)
	wg.Wait()
}

func TestObserverServer_consumeEpDelEvents(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	edn := &monitorAPI.EndpointDeleteNotification{
		EndpointRegenNotification: monitorAPI.EndpointRegenNotification{
			ID: 13,
		},
		PodName:   "bar",
		Namespace: "default",
	}
	ednMarshal, err := json.Marshal(edn)
	assert.Nil(t, err)
	fakeHandler := &fakeEndpointsHandler{
		fakeMarkDeleted: func(ep *v1.Endpoint) {
			defer wg.Done()
			assert.NotNil(t, ep.Deleted)
			wanted := &v1.Endpoint{
				ID:           13,
				Created:      time.Unix(0, 0),
				PodName:      "bar",
				PodNamespace: "default",
			}
			ep.Deleted = nil
			assert.Equal(t, wanted, ep)
		},
	}
	epDelCh := make(chan string, 1)
	s := &ObserverServer{
		endpoints: fakeHandler,
		epDel:     epDelCh,
		log:       zap.L(),
	}
	go s.consumeEpDelEvents()

	s.GetEpDelChannel() <- string(ednMarshal)
	wg.Wait()
}

func TestGetNamespace(t *testing.T) {
	ep := models.Endpoint{
		ID:   0,
		Spec: nil,
		Status: &models.EndpointStatus{
			Identity: &models.Identity{
				Labels: []string{"a=b", "c=d", "e=f"},
			},
		},
	}
	assert.Empty(t, GetNamespace(&ep))
	ns := "mynamespace"
	ep.Status.Identity.Labels = []string{"a=b", "c=d", fmt.Sprintf("%s=%s", v1.K8sNamespaceTag, ns)}
	assert.Equal(t, GetNamespace(&ep), ns)
}
