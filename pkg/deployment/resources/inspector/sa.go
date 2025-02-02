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

package inspector

import (
	"context"
	"time"

	"github.com/arangodb/kube-arangodb/pkg/util/errors"
	"github.com/arangodb/kube-arangodb/pkg/util/globals"
	"github.com/arangodb/kube-arangodb/pkg/util/k8sutil/inspector/throttle"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	requireRegisterInspectorLoader(serviceAccountsInspectorLoaderObj)
}

var serviceAccountsInspectorLoaderObj = serviceAccountsInspectorLoader{}

type serviceAccountsInspectorLoader struct {
}

func (p serviceAccountsInspectorLoader) Component() throttle.Component {
	return throttle.ServiceAccount
}

func (p serviceAccountsInspectorLoader) Load(ctx context.Context, i *inspectorState) {
	var q serviceAccountsInspector
	p.loadV1(ctx, i, &q)
	i.serviceAccounts = &q
	q.state = i
	q.last = time.Now()
}

func (p serviceAccountsInspectorLoader) loadV1(ctx context.Context, i *inspectorState, q *serviceAccountsInspector) {
	var z serviceAccountsInspectorV1

	z.serviceAccountInspector = q

	z.serviceAccounts, z.err = p.getV1ServiceAccounts(ctx, i)

	q.v1 = &z
}

func (p serviceAccountsInspectorLoader) getV1ServiceAccounts(ctx context.Context, i *inspectorState) (map[string]*core.ServiceAccount, error) {
	objs, err := p.getV1ServiceAccountsList(ctx, i)
	if err != nil {
		return nil, err
	}

	r := make(map[string]*core.ServiceAccount, len(objs))

	for id := range objs {
		r[objs[id].GetName()] = objs[id]
	}

	return r, nil
}

func (p serviceAccountsInspectorLoader) getV1ServiceAccountsList(ctx context.Context, i *inspectorState) ([]*core.ServiceAccount, error) {
	ctxChild, cancel := globals.GetGlobalTimeouts().Kubernetes().WithTimeout(ctx)
	defer cancel()
	obj, err := i.client.Kubernetes().CoreV1().ServiceAccounts(i.namespace).List(ctxChild, meta.ListOptions{
		Limit: globals.GetGlobals().Kubernetes().RequestBatchSize().Get(),
	})

	if err != nil {
		return nil, err
	}

	items := obj.Items
	cont := obj.Continue
	var s = int64(len(items))

	if z := obj.RemainingItemCount; z != nil {
		s += *z
	}

	ptrs := make([]*core.ServiceAccount, 0, s)

	for {
		for id := range items {
			ptrs = append(ptrs, &items[id])
		}

		if cont == "" {
			break
		}

		items, cont, err = p.getV1ServiceAccountsListRequest(ctx, i, cont)

		if err != nil {
			return nil, err
		}
	}

	return ptrs, nil
}

func (p serviceAccountsInspectorLoader) getV1ServiceAccountsListRequest(ctx context.Context, i *inspectorState, cont string) ([]core.ServiceAccount, string, error) {
	ctxChild, cancel := globals.GetGlobalTimeouts().Kubernetes().WithTimeout(ctx)
	defer cancel()
	obj, err := i.client.Kubernetes().CoreV1().ServiceAccounts(i.namespace).List(ctxChild, meta.ListOptions{
		Limit:    globals.GetGlobals().Kubernetes().RequestBatchSize().Get(),
		Continue: cont,
	})

	if err != nil {
		return nil, "", err
	}

	return obj.Items, obj.Continue, err
}

func (p serviceAccountsInspectorLoader) Verify(i *inspectorState) error {
	if err := i.serviceAccounts.v1.err; err != nil {
		return err
	}

	return nil
}

func (p serviceAccountsInspectorLoader) Copy(from, to *inspectorState, override bool) {
	if to.serviceAccounts != nil {
		if !override {
			return
		}
	}

	to.serviceAccounts = from.serviceAccounts
	to.serviceAccounts.state = to
}

func (p serviceAccountsInspectorLoader) Name() string {
	return "serviceAccounts"
}

type serviceAccountsInspector struct {
	state *inspectorState

	last time.Time

	v1 *serviceAccountsInspectorV1
}

func (p *serviceAccountsInspector) LastRefresh() time.Time {
	return p.last
}

func (p *serviceAccountsInspector) Refresh(ctx context.Context) error {
	p.Throttle(p.state.throttles).Invalidate()
	return p.state.refresh(ctx, serviceAccountsInspectorLoaderObj)
}

func (p serviceAccountsInspector) Throttle(c throttle.Components) throttle.Throttle {
	return c.ServiceAccount()
}

func (p *serviceAccountsInspector) validate() error {
	if p == nil {
		return errors.Newf("ServiceAccountInspector is nil")
	}

	if p.state == nil {
		return errors.Newf("Parent is nil")
	}

	return p.v1.validate()
}
