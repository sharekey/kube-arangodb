//
// DISCLAIMER
//
// Copyright 2016-2022 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//

package kclient

import (
	"sync"

	api "github.com/arangodb/kube-arangodb/pkg/apis/deployment/v1"
	versionedFake "github.com/arangodb/kube-arangodb/pkg/generated/clientset/versioned/fake"
	monitoring "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringFake "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/fake"
	core "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	apiextensionsclientFake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesFake "k8s.io/client-go/kubernetes/fake"
)

func NewFakeClient() Client {
	return NewStaticClient(kubernetesFake.NewSimpleClientset(), apiextensionsclientFake.NewSimpleClientset(), versionedFake.NewSimpleClientset(), monitoringFake.NewSimpleClientset())
}

type FakeClientBuilder interface {
	Add(objects ...runtime.Object) FakeClientBuilder

	Client() Client
}

func NewFakeClientBuilder() FakeClientBuilder {
	return &fakeClientBuilder{}
}

type fakeClientBuilder struct {
	lock sync.Mutex

	all []runtime.Object
}

func (f *fakeClientBuilder) Add(objects ...runtime.Object) FakeClientBuilder {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.all = append(f.all, objects...)

	return f
}

func (f *fakeClientBuilder) filter(reg func(s *runtime.Scheme) error) []runtime.Object {
	s := runtime.NewScheme()

	r := make([]runtime.Object, 0, len(f.all))

	if err := reg(s); err != nil {
		panic(err)
	}

	for _, o := range f.all {
		if o == nil {
			continue
		}
		if _, _, err := s.ObjectKinds(o); err == nil {
			r = append(r, o)
		}
	}

	return r
}

func (f *fakeClientBuilder) Client() Client {
	return NewStaticClient(
		kubernetesFake.NewSimpleClientset(f.filter(kubernetesFake.AddToScheme)...),
		apiextensionsclientFake.NewSimpleClientset(f.filter(apiextensionsclientFake.AddToScheme)...),
		versionedFake.NewSimpleClientset(f.filter(versionedFake.AddToScheme)...),
		monitoringFake.NewSimpleClientset(f.filter(monitoringFake.AddToScheme)...))
}

type FakeDataInput struct {
	Namespace string

	Pods            map[string]*core.Pod
	Secrets         map[string]*core.Secret
	Services        map[string]*core.Service
	PVCS            map[string]*core.PersistentVolumeClaim
	ServiceAccounts map[string]*core.ServiceAccount
	PDBS            map[string]*policy.PodDisruptionBudget
	ServiceMonitors map[string]*monitoring.ServiceMonitor
	ArangoMembers   map[string]*api.ArangoMember
	Nodes           map[string]*core.Node
	ACS             map[string]*api.ArangoClusterSynchronization
	AT              map[string]*api.ArangoTask
}

func (f FakeDataInput) asList() []runtime.Object {
	var r []runtime.Object

	for k, v := range f.Pods {
		c := v.DeepCopy()
		c.SetName(k)
		if c.GetNamespace() == "" && f.Namespace != "" {
			c.SetNamespace(f.Namespace)
		}
		r = append(r, c)
	}
	for k, v := range f.Secrets {
		c := v.DeepCopy()
		c.SetName(k)
		if c.GetNamespace() == "" && f.Namespace != "" {
			c.SetNamespace(f.Namespace)
		}
		r = append(r, c)
	}
	for k, v := range f.Services {
		c := v.DeepCopy()
		c.SetName(k)
		if c.GetNamespace() == "" && f.Namespace != "" {
			c.SetNamespace(f.Namespace)
		}
		r = append(r, c)
	}
	for k, v := range f.PVCS {
		c := v.DeepCopy()
		c.SetName(k)
		if c.GetNamespace() == "" && f.Namespace != "" {
			c.SetNamespace(f.Namespace)
		}
		r = append(r, c)
	}
	for k, v := range f.ServiceAccounts {
		c := v.DeepCopy()
		c.SetName(k)
		if c.GetNamespace() == "" && f.Namespace != "" {
			c.SetNamespace(f.Namespace)
		}
		r = append(r, c)
	}
	for k, v := range f.PDBS {
		c := v.DeepCopy()
		c.SetName(k)
		if c.GetNamespace() == "" && f.Namespace != "" {
			c.SetNamespace(f.Namespace)
		}
		r = append(r, c)
	}
	for k, v := range f.ServiceMonitors {
		c := v.DeepCopy()
		c.SetName(k)
		if c.GetNamespace() == "" && f.Namespace != "" {
			c.SetNamespace(f.Namespace)
		}
		r = append(r, c)
	}
	for k, v := range f.ArangoMembers {
		c := v.DeepCopy()
		c.SetName(k)
		if c.GetNamespace() == "" && f.Namespace != "" {
			c.SetNamespace(f.Namespace)
		}
		r = append(r, c)
	}
	for k, v := range f.Nodes {
		c := v.DeepCopy()
		c.SetName(k)
		if c.GetNamespace() == "" && f.Namespace != "" {
			c.SetNamespace(f.Namespace)
		}
		r = append(r, c)
	}
	for k, v := range f.ACS {
		c := v.DeepCopy()
		c.SetName(k)
		if c.GetNamespace() == "" && f.Namespace != "" {
			c.SetNamespace(f.Namespace)
		}
		r = append(r, c)
	}
	for k, v := range f.AT {
		c := v.DeepCopy()
		c.SetName(k)
		if c.GetNamespace() == "" && f.Namespace != "" {
			c.SetNamespace(f.Namespace)
		}
		r = append(r, c)
	}

	for _, o := range r {
		if f.Namespace != "" {
			if m, ok := o.(meta.Object); ok {
				if m.GetName() == "" {
					panic("Invalid data")
				}
				if n := m.GetNamespace(); n == "" {
					m.SetNamespace(f.Namespace)
				}
			}
		}
	}

	return r
}

func (f FakeDataInput) Client() Client {
	return NewFakeClientBuilder().Add(f.asList()...).Client()
}
