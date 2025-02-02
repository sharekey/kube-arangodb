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
	policy "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// PodDisruptionBudget
const (
	PodDisruptionBudgetGroup          = policy.GroupName
	PodDisruptionBudgetResource       = "poddisruptionbudgets"
	PodDisruptionBudgetKind           = "PodDisruptionBudget"
	PodDisruptionBudgetVersionV1Beta1 = "v1beta1"
	PodDisruptionBudgetVersionV1      = "v1"
)

func PodDisruptionBudgetGK() schema.GroupKind {
	return schema.GroupKind{
		Group: PodDisruptionBudgetGroup,
		Kind:  PodDisruptionBudgetKind,
	}
}

func PodDisruptionBudgetGKv1Beta1() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   PodDisruptionBudgetGroup,
		Kind:    PodDisruptionBudgetKind,
		Version: PodDisruptionBudgetVersionV1Beta1,
	}
}

func PodDisruptionBudgetGKv1() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   PodDisruptionBudgetGroup,
		Kind:    PodDisruptionBudgetKind,
		Version: PodDisruptionBudgetVersionV1,
	}
}

func PodDisruptionBudgetGR() schema.GroupResource {
	return schema.GroupResource{
		Group:    PodDisruptionBudgetGroup,
		Resource: PodDisruptionBudgetResource,
	}
}

func PodDisruptionBudgetGRv1() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    PodDisruptionBudgetGroup,
		Resource: PodDisruptionBudgetResource,
		Version:  PodDisruptionBudgetVersionV1,
	}
}

func PodDisruptionBudgetGRv1Beta1() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    PodDisruptionBudgetGroup,
		Resource: PodDisruptionBudgetResource,
		Version:  PodDisruptionBudgetVersionV1Beta1,
	}
}

func (p *podDisruptionBudgetsInspectorV1) GroupVersionKind() schema.GroupVersionKind {
	return PodDisruptionBudgetGKv1()
}

func (p *podDisruptionBudgetsInspectorV1) GroupVersionResource() schema.GroupVersionResource {
	return PodDisruptionBudgetGRv1()
}

func (p *podDisruptionBudgetsInspectorV1Beta1) GroupVersionKind() schema.GroupVersionKind {
	return PodDisruptionBudgetGKv1Beta1()
}

func (p *podDisruptionBudgetsInspectorV1Beta1) GroupVersionResource() schema.GroupVersionResource {
	return PodDisruptionBudgetGRv1Beta1()
}

func (p *podDisruptionBudgetsInspector) GroupKind() schema.GroupKind {
	return PodDisruptionBudgetGK()
}

func (p *podDisruptionBudgetsInspector) GroupResource() schema.GroupResource {
	return PodDisruptionBudgetGR()
}
