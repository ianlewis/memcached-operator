// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	core "k8s.io/client-go/testing"
)

// ExpectedAction is an action that is expected to occur and and optional
// function with additional test code to be run for the action.
type ExpectedAction struct {
	action core.Action
	f      func(*testing.T, core.Action)
}

func (a *ExpectedAction) Check(t *testing.T, action core.Action) {
	assert.Condition(t, func() bool { return a.action.Matches(action.GetVerb(), action.GetResource().Resource) }, "action must match")
	assert.Equal(t, action.GetSubresource(), a.action.GetSubresource(), "action subresource must match")
	assert.Equal(t, action.GetNamespace(), a.action.GetNamespace(), "action namespace must match")

	if a.f != nil {
		a.f(t, action)
	}
}

// ExpectClientActions tests where actions on the core client match the expected actions
func (f *ClientFixture) ExpectClientActions(expected []ExpectedAction) {
	f.expectActions(f.Client.Actions(), expected)
}

// ExpectCRDClientActions tests where actions on the CRD client match the expected actions
func (f *ClientFixture) ExpectCRDClientActions(expected []ExpectedAction) {
	f.expectActions(f.Client.Actions(), expected)
}

// expectActions tests whether the actions given match the expected actions.
func (f *ClientFixture) expectActions(actions []core.Action, expected []ExpectedAction) {
	for i, action := range actions {
		if len(expected) < i+1 {
			assert.Failf(f.t, "unexpected actions", "%+v", actions[i:])
			break
		}

		expected[i].Check(f.t, action)
	}

	if len(expected) > len(actions) {
		assert.Failf(f.t, "additional expected actions", "%+v", expected[len(actions):])
	}
}
