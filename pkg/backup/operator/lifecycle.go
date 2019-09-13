//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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
// Author Adam Janikowski
//

package operator

// LifecyclePreStart interface executed before operator starts
type LifecyclePreStart interface {
	Handler

	LifecyclePreStart() error
}

// ExecLifecyclePreStart execute PreStart step on handler if interface is implemented
func ExecLifecyclePreStart(handler Handler) error {
	if l, ok := handler.(LifecyclePreStart); ok {
		return l.LifecyclePreStart()
	}
	return nil
}
