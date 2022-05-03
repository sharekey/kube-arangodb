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

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type serviceMonitorsInspectorAnonymousV1 struct {
	i *serviceMonitorsInspectorV1
}

func (e *serviceMonitorsInspectorAnonymousV1) Get(ctx context.Context, name string, opts meta.GetOptions) (meta.Object, error) {
	return e.i.Get(ctx, name, opts)
}
