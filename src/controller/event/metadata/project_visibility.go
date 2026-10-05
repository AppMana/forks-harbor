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
	"time"

	events "github.com/goharbor/harbor/src/controller/event"
	"github.com/goharbor/harbor/src/controller/event/operator"
	"github.com/goharbor/harbor/src/pkg/notification"
	"github.com/goharbor/harbor/src/pkg/notifier/event"
)

// ProjectVisibilityChangedMetadata retains the before/after values from the mutation.
type ProjectVisibilityChangedMetadata struct {
	ProjectID int64
	Project   string
	Operator  string
	OldPublic bool
	NewPublic bool
	OccurAt   time.Time
}

func (m *ProjectVisibilityChangedMetadata) Resolve(e *event.Event) error {
	e.Topic = events.TopicProjectVisibilityChanged
	e.Data = &events.ProjectVisibilityChangedEvent{
		ProjectID: m.ProjectID, Project: m.Project, Operator: m.Operator,
		OldPublic: m.OldPublic, NewPublic: m.NewPublic, OccurAt: m.OccurAt,
	}
	return nil
}

// QueueProjectVisibilityChange uses the existing post-transaction notification
// context. Failed/rolled-back API requests are discarded by that middleware.
func QueueProjectVisibilityChange(ctx context.Context, id int64, name string, oldPublic, newPublic bool) {
	if oldPublic == newPublic {
		return
	}
	notification.AddEvent(ctx, &ProjectVisibilityChangedMetadata{
		ProjectID: id, Project: name, Operator: operator.FromContext(ctx),
		OldPublic: oldPublic, NewPublic: newPublic, OccurAt: time.Now(),
	})
}
