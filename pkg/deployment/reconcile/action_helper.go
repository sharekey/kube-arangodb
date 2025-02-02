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

package reconcile

import (
	"context"

	"github.com/rs/zerolog"

	api "github.com/arangodb/kube-arangodb/pkg/apis/deployment/v1"
)

type actionEmpty struct {
	actionImpl
	actionEmptyStart
	actionEmptyCheckProgress
}

type actionEmptyCheckProgress struct {
}

// CheckProgress define optional check progress for action
// Returns: ready, abort, error.
func (e actionEmptyCheckProgress) CheckProgress(_ context.Context) (bool, bool, error) {
	return true, false, nil
}

type actionEmptyStart struct {
}

func (e actionEmptyStart) Start(_ context.Context) (bool, error) {
	return false, nil
}

func newActionImplDefRef(log zerolog.Logger, action api.Action, actionCtx ActionContext) actionImpl {
	return newActionImpl(log, action, actionCtx, &action.MemberID)
}

func newActionImpl(log zerolog.Logger, action api.Action, actionCtx ActionContext, memberIDRef *string) actionImpl {
	if memberIDRef == nil {
		panic("Action cannot have nil reference to member!")
	}

	return newBaseActionImpl(log, action, actionCtx, memberIDRef)
}

func newBaseActionImplDefRef(log zerolog.Logger, action api.Action, actionCtx ActionContext) actionImpl {
	return newBaseActionImpl(log, action, actionCtx, &action.MemberID)
}

func newBaseActionImpl(log zerolog.Logger, action api.Action, actionCtx ActionContext, memberIDRef *string) actionImpl {
	if memberIDRef == nil {
		panic("Action cannot have nil reference to member!")
	}

	return actionImpl{
		log:         log,
		action:      action,
		actionCtx:   actionCtx,
		memberIDRef: memberIDRef,
	}
}

type actionImpl struct {
	log       zerolog.Logger
	action    api.Action
	actionCtx ActionContext

	memberIDRef *string
}

// MemberID returns the member ID used / created in the current action.
func (a actionImpl) MemberID() string {
	return *a.memberIDRef
}
