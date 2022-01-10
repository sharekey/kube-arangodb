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
// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v2alpha1 "github.com/arangodb/kube-arangodb/pkg/apis/deployment/v2alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeArangoClusterSynchronizations implements ArangoClusterSynchronizationInterface
type FakeArangoClusterSynchronizations struct {
	Fake *FakeDatabaseV2alpha1
	ns   string
}

var arangoclustersynchronizationsResource = schema.GroupVersionResource{Group: "database.arangodb.com", Version: "v2alpha1", Resource: "arangoclustersynchronizations"}

var arangoclustersynchronizationsKind = schema.GroupVersionKind{Group: "database.arangodb.com", Version: "v2alpha1", Kind: "ArangoClusterSynchronization"}

// Get takes name of the arangoClusterSynchronization, and returns the corresponding arangoClusterSynchronization object, and an error if there is any.
func (c *FakeArangoClusterSynchronizations) Get(ctx context.Context, name string, options v1.GetOptions) (result *v2alpha1.ArangoClusterSynchronization, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(arangoclustersynchronizationsResource, c.ns, name), &v2alpha1.ArangoClusterSynchronization{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v2alpha1.ArangoClusterSynchronization), err
}

// List takes label and field selectors, and returns the list of ArangoClusterSynchronizations that match those selectors.
func (c *FakeArangoClusterSynchronizations) List(ctx context.Context, opts v1.ListOptions) (result *v2alpha1.ArangoClusterSynchronizationList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(arangoclustersynchronizationsResource, arangoclustersynchronizationsKind, c.ns, opts), &v2alpha1.ArangoClusterSynchronizationList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v2alpha1.ArangoClusterSynchronizationList{ListMeta: obj.(*v2alpha1.ArangoClusterSynchronizationList).ListMeta}
	for _, item := range obj.(*v2alpha1.ArangoClusterSynchronizationList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested arangoClusterSynchronizations.
func (c *FakeArangoClusterSynchronizations) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(arangoclustersynchronizationsResource, c.ns, opts))

}

// Create takes the representation of a arangoClusterSynchronization and creates it.  Returns the server's representation of the arangoClusterSynchronization, and an error, if there is any.
func (c *FakeArangoClusterSynchronizations) Create(ctx context.Context, arangoClusterSynchronization *v2alpha1.ArangoClusterSynchronization, opts v1.CreateOptions) (result *v2alpha1.ArangoClusterSynchronization, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(arangoclustersynchronizationsResource, c.ns, arangoClusterSynchronization), &v2alpha1.ArangoClusterSynchronization{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v2alpha1.ArangoClusterSynchronization), err
}

// Update takes the representation of a arangoClusterSynchronization and updates it. Returns the server's representation of the arangoClusterSynchronization, and an error, if there is any.
func (c *FakeArangoClusterSynchronizations) Update(ctx context.Context, arangoClusterSynchronization *v2alpha1.ArangoClusterSynchronization, opts v1.UpdateOptions) (result *v2alpha1.ArangoClusterSynchronization, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(arangoclustersynchronizationsResource, c.ns, arangoClusterSynchronization), &v2alpha1.ArangoClusterSynchronization{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v2alpha1.ArangoClusterSynchronization), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeArangoClusterSynchronizations) UpdateStatus(ctx context.Context, arangoClusterSynchronization *v2alpha1.ArangoClusterSynchronization, opts v1.UpdateOptions) (*v2alpha1.ArangoClusterSynchronization, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(arangoclustersynchronizationsResource, "status", c.ns, arangoClusterSynchronization), &v2alpha1.ArangoClusterSynchronization{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v2alpha1.ArangoClusterSynchronization), err
}

// Delete takes name of the arangoClusterSynchronization and deletes it. Returns an error if one occurs.
func (c *FakeArangoClusterSynchronizations) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(arangoclustersynchronizationsResource, c.ns, name), &v2alpha1.ArangoClusterSynchronization{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeArangoClusterSynchronizations) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(arangoclustersynchronizationsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v2alpha1.ArangoClusterSynchronizationList{})
	return err
}

// Patch applies the patch and returns the patched arangoClusterSynchronization.
func (c *FakeArangoClusterSynchronizations) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v2alpha1.ArangoClusterSynchronization, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(arangoclustersynchronizationsResource, c.ns, name, pt, data, subresources...), &v2alpha1.ArangoClusterSynchronization{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v2alpha1.ArangoClusterSynchronization), err
}
