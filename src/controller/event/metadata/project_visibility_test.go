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

package metadata

import (
	"context"
	"testing"

	events "github.com/goharbor/harbor/src/controller/event"
	"github.com/goharbor/harbor/src/controller/event/operator"
	"github.com/goharbor/harbor/src/pkg/notification"
	"github.com/goharbor/harbor/src/pkg/notifier/event"
	"github.com/stretchr/testify/require"
)

func TestProjectVisibilityChangeSuppressesNoopsAndPreservesBeforeAfter(t *testing.T) {
	for _, old := range []bool{false, true} {
		for _, next := range []bool{false, true} {
			queue := notification.NewEventCtx()
			ctx := notification.NewContext(context.WithValue(context.Background(), operator.ContextKey{}, "admin"), queue)
			QueueProjectVisibilityChange(ctx, 42, "workspace-public-release", old, next)
			if old == next {
				require.Zero(t, queue.Events.Len())
				continue
			}
			require.Equal(t, 1, queue.Events.Len())
			require.False(t, queue.MustNotify, "visibility changes must not bypass rollback suppression")
			resolved := &event.Event{}
			require.NoError(t, queue.Events.Front().Value.(*ProjectVisibilityChangedMetadata).Resolve(resolved))
			require.Equal(t, events.TopicProjectVisibilityChanged, resolved.Topic)
			changed := resolved.Data.(*events.ProjectVisibilityChangedEvent)
			require.Equal(t, int64(42), changed.ProjectID)
			require.Equal(t, "workspace-public-release", changed.Project)
			require.Equal(t, "admin", changed.Operator)
			require.Equal(t, old, changed.OldPublic)
			require.Equal(t, next, changed.NewPublic)
			require.False(t, changed.OccurAt.IsZero())
		}
	}
}
