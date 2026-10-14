// Copyright Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package notification

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRobotAndProjectDeletionEventsAreSupportedWebhookEvents(t *testing.T) {
	initSupportedNotifyType()
	for _, eventType := range []EventType{"CREATE_ROBOT", "UPDATE_ROBOT", "DELETE_ROBOT", "DELETE_PROJECT", "PROJECT_VISIBILITY_CHANGED"} {
		require.Contains(t, GetSupportedEventTypes(), eventType)
	}
}
