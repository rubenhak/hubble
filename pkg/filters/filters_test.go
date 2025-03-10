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

package filters

import (
	"reflect"
	"testing"

	"github.com/cilium/cilium/pkg/monitor/api"
	pb "github.com/cilium/hubble/api/v1/flow"
	v1 "github.com/cilium/hubble/pkg/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestApply(t *testing.T) {
	ffyes := FilterFuncs{func(_ *v1.Event) bool {
		return true
	}}
	ffno := FilterFuncs{func(_ *v1.Event) bool {
		return false
	}}

	type args struct {
		whitelist FilterFuncs
		blacklist FilterFuncs
		ev        *v1.Event
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{args: args{whitelist: ffyes}, want: true},
		{args: args{whitelist: ffno}, want: false},
		{args: args{blacklist: ffno}, want: true},
		{args: args{blacklist: ffyes}, want: false},
		{args: args{whitelist: ffyes, blacklist: ffyes}, want: false},
		{args: args{whitelist: ffyes, blacklist: ffno}, want: true},
		{args: args{whitelist: ffno, blacklist: ffyes}, want: false},
		{args: args{whitelist: ffno, blacklist: ffno}, want: false},
		{args: args{}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Apply(tt.args.whitelist, tt.args.blacklist, tt.args.ev); got != tt.want {
				t.Errorf("Apply() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatch(t *testing.T) {
	fyes := func(_ *v1.Event) bool {
		return true
	}
	fno := func(_ *v1.Event) bool {
		return false
	}
	fs := FilterFuncs{fyes, fno}
	assert.False(t, fs.MatchAll(nil))
	assert.True(t, fs.MatchOne(nil))
	assert.False(t, fs.MatchNone(nil))

	// When no filter is specified, MatchAll(), MatchOne() and MatchNone() must
	// all return true
	fs = FilterFuncs{}
	assert.True(t, fs.MatchAll(nil))
	assert.True(t, fs.MatchOne(nil))
	assert.True(t, fs.MatchNone(nil))
}

func TestIPFilter(t *testing.T) {
	type args struct {
		f  []*pb.FlowFilter
		ev []*v1.Event
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []bool
	}{
		{
			name: "source ip",
			args: args{
				f: []*pb.FlowFilter{
					{SourceIp: []string{"1.1.1.1", "f00d::a10:0:0:9195"}},
				},
				ev: []*v1.Event{
					{Flow: &pb.Flow{IP: &pb.IP{Source: "1.1.1.1", Destination: "2.2.2.2"}}},
					{Flow: &pb.Flow{IP: &pb.IP{Source: "2.2.2.2", Destination: "1.1.1.1"}}},
					{Flow: &pb.Flow{IP: &pb.IP{Source: "f00d::a10:0:0:9195", Destination: "ff02::1:ff00:b3e5"}}},
					{Flow: &pb.Flow{IP: &pb.IP{Source: "ff02::1:ff00:b3e5", Destination: "f00d::a10:0:0:9195"}}},
				},
			},
			want: []bool{
				true,
				false,
				true,
				false,
			},
		},
		{
			name: "destination ip",
			args: args{
				f: []*pb.FlowFilter{
					{DestinationIp: []string{"1.1.1.1", "f00d::a10:0:0:9195"}},
				},
				ev: []*v1.Event{
					{Flow: &pb.Flow{IP: &pb.IP{Source: "1.1.1.1", Destination: "2.2.2.2"}}},
					{Flow: &pb.Flow{IP: &pb.IP{Source: "2.2.2.2", Destination: "1.1.1.1"}}},
					{Flow: &pb.Flow{IP: &pb.IP{Source: "f00d::a10:0:0:9195", Destination: "ff02::1:ff00:b3e5"}}},
					{Flow: &pb.Flow{IP: &pb.IP{Source: "ff02::1:ff00:b3e5", Destination: "f00d::a10:0:0:9195"}}},
				},
			},
			want: []bool{
				false,
				true,
				false,
				true,
			},
		},
		{
			name: "source and destination ip",
			args: args{
				f: []*pb.FlowFilter{
					{
						SourceIp:      []string{"1.1.1.1", "f00d::a10:0:0:9195"},
						DestinationIp: []string{"2.2.2.2", "ff02::1:ff00:b3e5"},
					},
				},
				ev: []*v1.Event{
					{Flow: &pb.Flow{IP: &pb.IP{Source: "1.1.1.1", Destination: "2.2.2.2"}}},
					{Flow: &pb.Flow{IP: &pb.IP{Source: "2.2.2.2", Destination: "1.1.1.1"}}},
					{Flow: &pb.Flow{IP: &pb.IP{Source: "f00d::a10:0:0:9195", Destination: "ff02::1:ff00:b3e5"}}},
					{Flow: &pb.Flow{IP: &pb.IP{Source: "ff02::1:ff00:b3e5", Destination: "f00d::a10:0:0:9195"}}},
				},
			},
			want: []bool{
				true,
				false,
				true,
				false,
			},
		},
		{
			name: "source or destination ip",
			args: args{
				f: []*pb.FlowFilter{
					{SourceIp: []string{"1.1.1.1"}},
					{DestinationIp: []string{"2.2.2.2"}},
				},
				ev: []*v1.Event{
					{Flow: &pb.Flow{IP: &pb.IP{Source: "1.1.1.1", Destination: "2.2.2.2"}}},
					{Flow: &pb.Flow{IP: &pb.IP{Source: "2.2.2.2", Destination: "1.1.1.1"}}},
					{Flow: &pb.Flow{IP: &pb.IP{Source: "1.1.1.1", Destination: "1.1.1.1"}}},
					{Flow: &pb.Flow{IP: &pb.IP{Source: "2.2.2.2", Destination: "2.2.2.2"}}},
				},
			},
			want: []bool{
				true,
				false,
				true,
				true,
			},
		},
		{
			name: "invalid data",
			args: args{
				f: []*pb.FlowFilter{
					{SourceIp: []string{"1.1.1.1"}},
				},
				ev: []*v1.Event{
					nil,
					{},
					{Flow: &pb.Flow{}},
					{Flow: &pb.Flow{IP: &pb.IP{Source: ""}}},
				},
			},
			want: []bool{
				false,
				false,
				false,
				false,
			},
		},
		{
			name: "invalid source ip filter",
			args: args{
				f: []*pb.FlowFilter{
					{SourceIp: []string{"320.320.320.320"}},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid destination ip filter",
			args: args{
				f: []*pb.FlowFilter{
					{DestinationIp: []string{""}},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fl, err := BuildFilterList(tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildFilterList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for i, ev := range tt.args.ev {
				if filterResult := fl.MatchOne(ev); filterResult != tt.want[i] {
					t.Errorf("\"%s\" filterResult %d = %v, want %v", tt.name, i, filterResult, tt.want[i])
				}
			}
		})
	}
}

func TestPodFilter(t *testing.T) {
	type args struct {
		f  []*pb.FlowFilter
		ev []*v1.Event
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []bool
	}{
		{
			name: "source pod",
			args: args{
				f: []*pb.FlowFilter{
					{SourcePod: []string{"xwing", "default/tiefighter"}},
				},
				ev: []*v1.Event{
					{Flow: &pb.Flow{Source: &pb.Endpoint{Namespace: "default", PodName: "xwing"}}},
					{Flow: &pb.Flow{Source: &pb.Endpoint{Namespace: "default", PodName: "tiefighter"}}},
					{Flow: &pb.Flow{Source: &pb.Endpoint{Namespace: "kube-system", PodName: "xwing"}}},
					{Flow: &pb.Flow{Destination: &pb.Endpoint{Namespace: "default", PodName: "xwing"}}},
				},
			},
			want: []bool{
				true,
				true,
				false,
				false,
			},
		},
		{
			name: "destination pod",
			args: args{
				f: []*pb.FlowFilter{
					{DestinationPod: []string{"xwing", "default/tiefighter"}},
				},
				ev: []*v1.Event{
					{Flow: &pb.Flow{Destination: &pb.Endpoint{Namespace: "default", PodName: "xwing"}}},
					{Flow: &pb.Flow{Destination: &pb.Endpoint{Namespace: "default", PodName: "tiefighter"}}},
					{Flow: &pb.Flow{Destination: &pb.Endpoint{Namespace: "kube-system", PodName: "xwing"}}},
					{Flow: &pb.Flow{Source: &pb.Endpoint{Namespace: "default", PodName: "xwing"}}},
				},
			},
			want: []bool{
				true,
				true,
				false,
				false,
			},
		},
		{
			name: "source and destination pod",
			args: args{
				f: []*pb.FlowFilter{
					{
						SourcePod:      []string{"xwing", "tiefighter"},
						DestinationPod: []string{"deathstar"},
					},
				},
				ev: []*v1.Event{
					{Flow: &pb.Flow{
						Source:      &pb.Endpoint{Namespace: "default", PodName: "xwing"},
						Destination: &pb.Endpoint{Namespace: "default", PodName: "deathstar"},
					}},
					{Flow: &pb.Flow{
						Source:      &pb.Endpoint{Namespace: "default", PodName: "tiefighter"},
						Destination: &pb.Endpoint{Namespace: "default", PodName: "deathstar"},
					}},
					{Flow: &pb.Flow{
						Source:      &pb.Endpoint{Namespace: "default", PodName: "deathstar"},
						Destination: &pb.Endpoint{Namespace: "default", PodName: "xwing"},
					}},
					{Flow: &pb.Flow{
						Source:      &pb.Endpoint{Namespace: "default", PodName: "tiefighter"},
						Destination: &pb.Endpoint{Namespace: "default", PodName: "xwing"},
					}},
				},
			},
			want: []bool{
				true,
				true,
				false,
				false,
			},
		},
		{
			name: "source or destination pod",
			args: args{
				f: []*pb.FlowFilter{
					{SourcePod: []string{"xwing"}},
					{DestinationPod: []string{"deathstar"}},
				},
				ev: []*v1.Event{
					{Flow: &pb.Flow{
						Source:      &pb.Endpoint{Namespace: "default", PodName: "xwing"},
						Destination: &pb.Endpoint{Namespace: "default", PodName: "deathstar"},
					}},
					{Flow: &pb.Flow{
						Source:      &pb.Endpoint{Namespace: "default", PodName: "tiefighter"},
						Destination: &pb.Endpoint{Namespace: "default", PodName: "deathstar"},
					}},
					{Flow: &pb.Flow{
						Source:      &pb.Endpoint{Namespace: "default", PodName: "deathstar"},
						Destination: &pb.Endpoint{Namespace: "default", PodName: "xwing"},
					}},
					{Flow: &pb.Flow{
						Source:      &pb.Endpoint{Namespace: "default", PodName: "tiefighter"},
						Destination: &pb.Endpoint{Namespace: "default", PodName: "xwing"},
					}},
				},
			},
			want: []bool{
				true,
				true,
				false,
				false,
			},
		},
		{
			name: "namespace filter",
			args: args{
				f: []*pb.FlowFilter{
					{SourcePod: []string{"kube-system/"}},
					{DestinationPod: []string{"kube-system/"}},
				},
				ev: []*v1.Event{
					{Flow: &pb.Flow{
						Source:      &pb.Endpoint{Namespace: "kube-system", PodName: "coredns"},
						Destination: &pb.Endpoint{Namespace: "kube-system", PodName: "kube-proxy"},
					}},
					{Flow: &pb.Flow{
						Source:      &pb.Endpoint{Namespace: "default", PodName: "tiefighter"},
						Destination: &pb.Endpoint{Namespace: "kube-system", PodName: "coredns"},
					}},
					{Flow: &pb.Flow{
						Source:      &pb.Endpoint{Namespace: "kube-system", PodName: "coredns"},
						Destination: &pb.Endpoint{Namespace: "default", PodName: "xwing"},
					}},
					{Flow: &pb.Flow{
						Source:      &pb.Endpoint{Namespace: "default", PodName: "tiefighter"},
						Destination: &pb.Endpoint{Namespace: "default", PodName: "xwing"},
					}},
				},
			},
			want: []bool{
				true,
				true,
				true,
				false,
			},
		},
		{
			name: "prefix filter",
			args: args{
				f: []*pb.FlowFilter{
					{SourcePod: []string{"xwing", "kube-system/coredns-"}},
				},
				ev: []*v1.Event{
					{Flow: &pb.Flow{
						Source: &pb.Endpoint{Namespace: "default", PodName: "xwing"},
					}},
					{Flow: &pb.Flow{
						Source: &pb.Endpoint{Namespace: "default", PodName: "xwing-t-65b"},
					}},
					{Flow: &pb.Flow{
						Source: &pb.Endpoint{Namespace: "kube-system", PodName: "coredns-12345"},
					}},
					{Flow: &pb.Flow{
						Source: &pb.Endpoint{Namespace: "kube-system", PodName: "-coredns-12345"},
					}},
					{Flow: &pb.Flow{
						Source: &pb.Endpoint{Namespace: "default", PodName: "tiefighter"},
					}},
				},
			},
			want: []bool{
				true,
				true,
				true,
				false,
				false,
			},
		},
		{
			name: "invalid data",
			args: args{
				f: []*pb.FlowFilter{
					{SourcePod: []string{"xwing"}},
				},
				ev: []*v1.Event{
					nil,
					{},
					{Flow: &pb.Flow{}},
					{Flow: &pb.Flow{Source: &pb.Endpoint{Namespace: "", PodName: "xwing"}}},
				},
			},
			want: []bool{
				false,
				false,
				false,
				false,
			},
		},
		{
			name: "invalid source pod filter",
			args: args{
				f: []*pb.FlowFilter{
					{SourcePod: []string{""}},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid destination pod filter",
			args: args{
				f: []*pb.FlowFilter{
					{DestinationIp: []string{"/podname"}},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fl, err := BuildFilterList(tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildFilterList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for i, ev := range tt.args.ev {
				if filterResult := fl.MatchOne(ev); filterResult != tt.want[i] {
					t.Errorf("\"%s\" filterResult %d = %v, want %v", tt.name, i, filterResult, tt.want[i])
				}
			}
		})
	}
}

func TestFQDNFilter(t *testing.T) {
	type args struct {
		f  []*pb.FlowFilter
		ev []*v1.Event
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []bool
	}{
		{
			name: "source fqdn",
			args: args{
				f: []*pb.FlowFilter{
					{SourceFqdn: []string{"cilium.io", "ebpf.io"}},
				},
				ev: []*v1.Event{
					{Flow: &pb.Flow{SourceNames: []string{"cilium.io"}}},
					{Flow: &pb.Flow{SourceNames: []string{"ebpf.io"}}},
					{Flow: &pb.Flow{DestinationNames: []string{"cilium.io"}}},
					{Flow: &pb.Flow{DestinationNames: []string{"ebpf.io"}}},
				},
			},
			want: []bool{
				true,
				true,
				false,
				false,
			},
		},
		{
			name: "destination fqdn",
			args: args{
				f: []*pb.FlowFilter{
					{DestinationFqdn: []string{"cilium.io", "ebpf.io"}},
				},
				ev: []*v1.Event{
					{Flow: &pb.Flow{SourceNames: []string{"cilium.io"}}},
					{Flow: &pb.Flow{SourceNames: []string{"ebpf.io"}}},
					{Flow: &pb.Flow{DestinationNames: []string{"cilium.io"}}},
					{Flow: &pb.Flow{DestinationNames: []string{"ebpf.io"}}},
				},
			},
			want: []bool{
				false,
				false,
				true,
				true,
			},
		},
		{
			name: "source and destination fqdn",
			args: args{
				f: []*pb.FlowFilter{
					{
						SourceFqdn:      []string{"cilium.io", "docs.cilium.io"},
						DestinationFqdn: []string{"ebpf.io"},
					},
				},
				ev: []*v1.Event{
					{Flow: &pb.Flow{
						SourceNames:      []string{"cilium.io"},
						DestinationNames: []string{"ebpf.io"},
					}},
					{Flow: &pb.Flow{
						SourceNames:      []string{"ebpf.io"},
						DestinationNames: []string{"cilium.io"},
					}},
					{Flow: &pb.Flow{
						SourceNames:      []string{"deathstar.empire.svc.cluster.local", "docs.cilium.io"},
						DestinationNames: []string{"ebpf.io"},
					}},
				},
			},
			want: []bool{
				true,
				false,
				true,
			},
		},
		{
			name: "source or destination fqdn",
			args: args{
				f: []*pb.FlowFilter{
					{SourceFqdn: []string{"cilium.io", "docs.cilium.io"}},
					{DestinationFqdn: []string{"ebpf.io"}},
				},
				ev: []*v1.Event{
					{Flow: &pb.Flow{
						SourceNames:      []string{"cilium.io"},
						DestinationNames: []string{"ebpf.io"},
					}},
					{Flow: &pb.Flow{
						SourceNames:      []string{"ebpf.io"},
						DestinationNames: []string{"cilium.io"},
					}},
					{Flow: &pb.Flow{
						SourceNames: []string{"deathstar.empire.svc.cluster.local", "docs.cilium.io"},
					}},
					{Flow: &pb.Flow{
						DestinationNames: []string{"ebpf.io"},
					}},
					{Flow: &pb.Flow{
						SourceNames:      []string{"deathstar.empire.svc.cluster.local", "docs.cilium.io"},
						DestinationNames: []string{"ebpf.io"},
					}},
				},
			},
			want: []bool{
				true,
				false,
				true,
				true,
				true,
			},
		},
		{
			name: "invalid data",
			args: args{
				f: []*pb.FlowFilter{
					{SourceFqdn: []string{"cilium.io."}},
				},
				ev: []*v1.Event{
					nil,
					{},
					{Flow: &pb.Flow{}},
					{Flow: &pb.Flow{SourceNames: []string{"cilium.io."}}}, // should not have trailing dot
					{Flow: &pb.Flow{SourceNames: []string{"www.cilium.io"}}},
					{Flow: &pb.Flow{SourceNames: []string{""}}},
				},
			},
			want: []bool{
				false,
				false,
				false,
				false,
				false,
				false,
			},
		},
		{
			name: "invalid source fqdn filter",
			args: args{
				f: []*pb.FlowFilter{
					{SourceFqdn: []string{""}},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid destination fqdn filter",
			args: args{
				f: []*pb.FlowFilter{
					{DestinationFqdn: []string{"."}},
				},
			},
			wantErr: true,
		},
		{
			name: "wildcard filters",
			args: args{
				f: []*pb.FlowFilter{
					{SourceFqdn: []string{"*.cilium.io", "*.org."}},
					{DestinationFqdn: []string{"*"}},
				},
				ev: []*v1.Event{
					{Flow: &pb.Flow{SourceNames: []string{"www.cilium.io"}}},
					{Flow: &pb.Flow{SourceNames: []string{"multiple.domains.org"}}},
					{Flow: &pb.Flow{SourceNames: []string{"cilium.io"}}},
					{Flow: &pb.Flow{SourceNames: []string{"tiefighter", "empire.org"}}},
					{Flow: &pb.Flow{DestinationNames: []string{}}},
					{Flow: &pb.Flow{DestinationNames: []string{"anything.really"}}},
					{Flow: &pb.Flow{DestinationNames: []string{""}}},
				},
			},
			want: []bool{
				true,
				true,
				false,
				true,
				false,
				true,
				true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fl, err := BuildFilterList(tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildFilterList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for i, ev := range tt.args.ev {
				if filterResult := fl.MatchOne(ev); filterResult != tt.want[i] {
					t.Errorf("\"%s\" filterResult %d = %v, want %v", tt.name, i, filterResult, tt.want[i])
				}
			}
		})
	}
}

func TestVerdictFilter(t *testing.T) {
	ev := &v1.Event{
		Flow: &pb.Flow{
			Verdict: pb.Verdict_FORWARDED,
		},
	}
	assert.True(t, filterByVerdicts([]pb.Verdict{pb.Verdict_FORWARDED})(ev))
	assert.False(t, filterByVerdicts([]pb.Verdict{pb.Verdict_DROPPED})(ev))
}

func TestHttpStatusCodeFilter(t *testing.T) {
	httpFlow := func(http *pb.HTTP) *v1.Event {
		return &v1.Event{
			Flow: &pb.Flow{
				EventType: &pb.CiliumEventType{
					Type: api.MessageTypeAccessLog,
				},
				L7: &pb.Layer7{
					Record: &pb.Layer7_Http{
						Http: http,
					},
				}},
		}
	}

	type args struct {
		f  []*pb.FlowFilter
		ev []*v1.Event
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []bool
	}{
		{
			name: "status code full",
			args: args{
				f: []*pb.FlowFilter{
					{
						HttpStatusCode: []string{"200", "302"},
						EventType:      []*pb.EventTypeFilter{{Type: api.MessageTypeAccessLog}},
					},
				},
				ev: []*v1.Event{
					httpFlow(&pb.HTTP{Code: 200}),
					httpFlow(&pb.HTTP{Code: 302}),
					httpFlow(&pb.HTTP{Code: 404}),
					httpFlow(&pb.HTTP{Code: 500}),
				},
			},
			want: []bool{
				true,
				true,
				false,
				false,
			},
			wantErr: false,
		},
		{
			name: "status code prefix",
			args: args{
				f: []*pb.FlowFilter{
					{
						HttpStatusCode: []string{"40+", "5+"},
						EventType:      []*pb.EventTypeFilter{{Type: api.MessageTypeAccessLog}},
					},
				},
				ev: []*v1.Event{
					httpFlow(&pb.HTTP{Code: 302}),
					httpFlow(&pb.HTTP{Code: 400}),
					httpFlow(&pb.HTTP{Code: 404}),
					httpFlow(&pb.HTTP{Code: 410}),
					httpFlow(&pb.HTTP{Code: 004}),
					httpFlow(&pb.HTTP{Code: 500}),
					httpFlow(&pb.HTTP{Code: 501}),
					httpFlow(&pb.HTTP{Code: 510}),
					httpFlow(&pb.HTTP{Code: 050}),
				},
			},
			want: []bool{
				false,
				true,
				true,
				false,
				false,
				true,
				true,
				true,
				false,
			},
			wantErr: false,
		},
		{
			name: "invalid data",
			args: args{
				f: []*pb.FlowFilter{
					{
						HttpStatusCode: []string{"200"},
						EventType:      []*pb.EventTypeFilter{{Type: api.MessageTypeAccessLog}},
					},
				},
				ev: []*v1.Event{
					{Payload: &pb.Payload{Data: nil}},
					{Payload: &pb.Payload{Data: []byte{byte(api.MessageTypeAccessLog)}}},
					{Flow: &pb.Flow{}},
					httpFlow(&pb.HTTP{}),
					httpFlow(&pb.HTTP{Code: 666}),
				},
			},
			want: []bool{
				false,
				false,
				false,
				false,
				false,
			},
			wantErr: false,
		},
		{
			name: "invalid empty filter",
			args: args{
				f: []*pb.FlowFilter{
					{
						HttpStatusCode: []string{""},
						EventType:      []*pb.EventTypeFilter{{Type: api.MessageTypeAccessLog}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid catch-all prefix",
			args: args{
				f: []*pb.FlowFilter{
					{
						HttpStatusCode: []string{"+"},
						EventType:      []*pb.EventTypeFilter{{Type: api.MessageTypeAccessLog}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid status code",
			args: args{
				f: []*pb.FlowFilter{
					{
						HttpStatusCode: []string{"909"},
						EventType:      []*pb.EventTypeFilter{{Type: api.MessageTypeAccessLog}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid status code prefix",
			args: args{
				f: []*pb.FlowFilter{
					{
						HttpStatusCode: []string{"3++"},
						EventType:      []*pb.EventTypeFilter{{Type: api.MessageTypeAccessLog}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid status code prefix",
			args: args{
				f: []*pb.FlowFilter{
					{
						HttpStatusCode: []string{"3+0"},
						EventType:      []*pb.EventTypeFilter{{Type: api.MessageTypeAccessLog}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty event type filter",
			args: args{
				f: []*pb.FlowFilter{
					{
						HttpStatusCode: []string{"200"},
						EventType:      []*pb.EventTypeFilter{},
					},
				},
				ev: []*v1.Event{
					httpFlow(&pb.HTTP{Code: 200}),
				},
			},
			want: []bool{
				true,
			},
			wantErr: false,
		},
		{
			name: "compatible event type filter",
			args: args{
				f: []*pb.FlowFilter{
					{
						HttpStatusCode: []string{"200"},
						EventType: []*pb.EventTypeFilter{
							{Type: api.MessageTypeAccessLog},
							{Type: api.MessageTypeTrace},
						},
					},
				},
				ev: []*v1.Event{
					httpFlow(&pb.HTTP{Code: 200}),
				},
			},
			want: []bool{
				true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fl, err := BuildFilterList(tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("\"%s\" error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			for i, ev := range tt.args.ev {
				if got := fl.MatchOne(ev); got != tt.want[i] {
					t.Errorf("\"%s\" got %d = %v, want %v", tt.name, i, got, tt.want[i])
				}
			}
		})
	}
}

func TestLabelSelectorFilter(t *testing.T) {
	type args struct {
		f  []*pb.FlowFilter
		ev []*v1.Event
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []bool
	}{
		{
			name: "label filter without value",
			args: args{
				f: []*pb.FlowFilter{{SourceLabel: []string{"label1", "label2"}}},
				ev: []*v1.Event{
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"label1"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"label1=val1"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"label2", "label3", "label4=val4"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"label3"},
							},
						},
					},
				},
			},
			want: []bool{
				true,
				true,
				true,
				false,
			},
		},
		{
			name: "label filter with value",
			args: args{
				f: []*pb.FlowFilter{{SourceLabel: []string{"label1=val1", "label2=val2"}}},
				ev: []*v1.Event{
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"label1"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"label1=val1"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"label1=val2", "label2=val1", "label3"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"label2=val2", "label3"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"label3=val1"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{""},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: nil,
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"label1=val1=toomuch"},
							},
						},
					},
				},
			},
			want: []bool{
				false,
				true,
				false,
				true,
				false,
				false,
				false,
				false,
			},
		},
		{
			name: "complex label label filter",
			args: args{
				f: []*pb.FlowFilter{{SourceLabel: []string{"label1 in (val1, val2), label3 notin ()"}}},
				ev: []*v1.Event{
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"label1"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"label1=val1"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"label1=val2", "label2=val1", "label3=val3"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"label2=val2", "label3"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"label1=val1", "label3=val3"},
							},
						},
					},
				},
			},
			want: []bool{
				false,
				true,
				true,
				false,
				true,
			},
		},
		{
			name: "source and destination label filter",
			args: args{
				f: []*pb.FlowFilter{
					{
						SourceLabel:      []string{"src1, src2=val2"},
						DestinationLabel: []string{"dst1, dst2=val2"},
					},
				},
				ev: []*v1.Event{
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"src1", "src2=val2"},
							},
							Destination: &pb.Endpoint{
								Labels: []string{"dst1", "dst2=val2"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"label1=val1"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Destination: &pb.Endpoint{
								Labels: []string{"dst1", "dst2=val2"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"dst1", "dst2=val2"},
							},
							Destination: &pb.Endpoint{
								Labels: []string{"src1", "src2=val2"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"src1"},
							},
							Destination: &pb.Endpoint{
								Labels: []string{"dst1"},
							},
						},
					},
				},
			},
			want: []bool{
				true,
				false,
				false,
				false,
				false,
			},
		},
		{
			name: "matchall filter",
			args: args{
				f: []*pb.FlowFilter{
					{
						SourceLabel: []string{""},
					},
				},
				ev: []*v1.Event{
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"src1", "src2=val2"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: nil,
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{""},
							},
						},
					},
				},
			},
			want: []bool{
				true,
				true,
				true,
			},
		},
		{
			name: "cilium fixed prefix filters",
			args: args{
				f: []*pb.FlowFilter{
					{
						SourceLabel: []string{"k8s:app=bar", "foo", "reserved:host"},
					},
				},
				ev: []*v1.Event{
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"k8s:app=bar"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"k8s:foo=baz"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"k8s.app=bar"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"container:foo=bar", "reserved:host"},
							},
						},
					},
				},
			},
			want: []bool{
				true,
				true,
				false,
				true,
			},
		},
		{
			name: "cilium any prefix filters",
			args: args{
				f: []*pb.FlowFilter{
					{
						SourceLabel: []string{"any:key"},
					},
				},
				ev: []*v1.Event{
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"key"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"reserved:key"},
							},
						},
					},
					{
						Flow: &pb.Flow{
							Source: &pb.Endpoint{
								Labels: []string{"any.key"},
							},
						},
					},
				},
			},
			want: []bool{
				true,
				true,
				false,
			},
		},
		{
			name: "invalid source filter",
			args: args{
				f: []*pb.FlowFilter{
					{
						SourceLabel: []string{"()"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid destination filter",
			args: args{
				f: []*pb.FlowFilter{
					{
						DestinationLabel: []string{"="},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fl, err := BuildFilterList(tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("\"%s\" error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			for i, ev := range tt.args.ev {
				if got := fl.MatchOne(ev); got != tt.want[i] {
					t.Errorf("\"%s\" got %d = %v, want %v", tt.name, i, got, tt.want[i])
				}
			}
		})
	}
}

func TestFlowProtocolFilter(t *testing.T) {
	type args struct {
		f  []*pb.FlowFilter
		ev *v1.Event
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    bool
	}{
		{
			name: "udp",
			args: args{
				f: []*pb.FlowFilter{{Protocol: []string{"udp"}}},
				ev: &v1.Event{Flow: &pb.Flow{
					L4: &pb.Layer4{Protocol: &pb.Layer4_UDP{UDP: &pb.UDP{}}},
				}},
			},
			want: true,
		},
		{
			name: "http",
			args: args{
				f: []*pb.FlowFilter{{Protocol: []string{"http"}}},
				ev: &v1.Event{Flow: &pb.Flow{
					L4: &pb.Layer4{Protocol: &pb.Layer4_TCP{TCP: &pb.TCP{}}},
					L7: &pb.Layer7{Record: &pb.Layer7_Http{Http: &pb.HTTP{}}},
				}},
			},
			want: true,
		},
		{
			name: "icmp (v4)",
			args: args{
				f: []*pb.FlowFilter{{Protocol: []string{"icmp"}}},
				ev: &v1.Event{Flow: &pb.Flow{
					L4: &pb.Layer4{Protocol: &pb.Layer4_ICMPv4{ICMPv4: &pb.ICMPv4{}}},
				}},
			},
			want: true,
		},
		{
			name: "icmp (v6)",
			args: args{
				f: []*pb.FlowFilter{{Protocol: []string{"icmp"}}},
				ev: &v1.Event{Flow: &pb.Flow{
					L4: &pb.Layer4{Protocol: &pb.Layer4_ICMPv6{ICMPv6: &pb.ICMPv6{}}},
				}},
			},
			want: true,
		},
		{
			name: "multiple protocols",
			args: args{
				f: []*pb.FlowFilter{{Protocol: []string{"tcp", "kafka"}}},
				ev: &v1.Event{Flow: &pb.Flow{
					L4: &pb.Layer4{Protocol: &pb.Layer4_TCP{TCP: &pb.TCP{}}},
				}},
			},
			want: true,
		},
		{
			name: "invalid protocols",
			args: args{
				f: []*pb.FlowFilter{{Protocol: []string{"not a protocol"}}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fl, err := BuildFilterList(tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("\"%s\" error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if got := fl.MatchOne(tt.args.ev); got != tt.want {
				t.Errorf("\"%s\" got %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestPortFilter(t *testing.T) {
	type args struct {
		f  []*pb.FlowFilter
		ev *v1.Event
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    bool
	}{
		{
			name: "udp",
			args: args{
				f: []*pb.FlowFilter{{
					SourcePort:      []string{"12345"},
					DestinationPort: []string{"53"},
				}},
				ev: &v1.Event{Flow: &pb.Flow{
					L4: &pb.Layer4{Protocol: &pb.Layer4_UDP{UDP: &pb.UDP{
						SourcePort:      12345,
						DestinationPort: 53,
					}}},
				}},
			},
			want: true,
		},
		{
			name: "tcp",
			args: args{
				f: []*pb.FlowFilter{{
					SourcePort:      []string{"32320"},
					DestinationPort: []string{"80"},
				}},
				ev: &v1.Event{Flow: &pb.Flow{
					L4: &pb.Layer4{Protocol: &pb.Layer4_TCP{TCP: &pb.TCP{
						SourcePort:      32320,
						DestinationPort: 80,
					}}},
				}},
			},
			want: true,
		},
		{
			name: "wrong direction",
			args: args{
				f: []*pb.FlowFilter{{
					DestinationPort: []string{"80"},
				}},
				ev: &v1.Event{Flow: &pb.Flow{
					L4: &pb.Layer4{Protocol: &pb.Layer4_TCP{TCP: &pb.TCP{
						SourcePort:      80,
						DestinationPort: 32320,
					}}},
				}},
			},
			want: false,
		},
		{
			name: "no port",
			args: args{
				f: []*pb.FlowFilter{{
					DestinationPort: []string{"0"},
				}},
				ev: &v1.Event{Flow: &pb.Flow{
					L4: &pb.Layer4{Protocol: &pb.Layer4_ICMPv4{ICMPv4: &pb.ICMPv4{}}},
				}},
			},
			want: false,
		},
		{
			name: "invalid port",
			args: args{
				f: []*pb.FlowFilter{{SourcePort: []string{"999999"}}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fl, err := BuildFilterList(tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("\"%s\" error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if got := fl.MatchOne(tt.args.ev); got != tt.want {
				t.Errorf("\"%s\" got %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func Test_parseSelector(t *testing.T) {
	type args struct {
		selector string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "simple labels",
			args: args{
				selector: "bar=baz,k8s:app=hubble,reserved:world",
			},
			want: "bar=baz,k8s.app=hubble,reserved.world",
		},
		{
			name: "complex labels",
			args: args{
				selector: "any:dash-label.com,k8s:io.cilium in (is-awesome,rocks)",
			},
			want: "any.dash-label.com,k8s.io.cilium in (is-awesome,rocks)",
		},
		{
			name: "too many colons",
			args: args{
				selector: "any:k8s:bla",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSelector(tt.args.selector)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSelector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got.String(), tt.want) {
				t.Errorf("parseSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}
