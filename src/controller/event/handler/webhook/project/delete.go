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

package project

import (
	"context"
	"fmt"

	"github.com/goharbor/harbor/src/controller/event"
	"github.com/goharbor/harbor/src/controller/event/handler/util"
	"github.com/goharbor/harbor/src/lib/config"
	"github.com/goharbor/harbor/src/pkg/notification"
	"github.com/goharbor/harbor/src/pkg/notifier/model"
)

// DeleteHandler delivers project deletion to the deleted project's own
// webhook policies. Deletion only marks the project deleted, so they remain.
type DeleteHandler struct{}

func (*DeleteHandler) Name() string     { return "ProjectDeletionWebhook" }
func (*DeleteHandler) IsStateful() bool { return false }
func (*DeleteHandler) Handle(ctx context.Context, value any) error {
	deleted, ok := value.(*event.DeleteProjectEvent)
	if !ok || deleted == nil {
		return fmt.Errorf("invalid project deletion event")
	}
	if !config.NotificationEnable(ctx) {
		return nil
	}
	policies, err := notification.PolicyMgr.GetRelatedPolices(ctx, deleted.ProjectID, event.TopicDeleteProject)
	if err != nil {
		return err
	}
	if len(policies) == 0 {
		return nil
	}
	return util.SendHookWithPolicies(ctx, policies, deletionPayload(deleted), event.TopicDeleteProject)
}

func deletionPayload(deleted *event.DeleteProjectEvent) *model.Payload {
	return &model.Payload{
		Type: event.TopicDeleteProject, OccurAt: deleted.OccurAt.Unix(), Operator: deleted.Operator,
		EventData: &model.EventData{Project: &model.ProjectRef{
			ProjectID: deleted.ProjectID, ProjectName: deleted.Project,
		}},
	}
}
