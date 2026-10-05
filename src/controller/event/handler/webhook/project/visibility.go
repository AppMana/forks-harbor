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

// Handler delivers project permission changes through Harbor's existing job/retry pipeline.
type Handler struct{}

func (*Handler) Name() string     { return "ProjectVisibilityWebhook" }
func (*Handler) IsStateful() bool { return false }
func (*Handler) Handle(ctx context.Context, value any) error {
	changed, ok := value.(*event.ProjectVisibilityChangedEvent)
	if !ok || changed == nil {
		return fmt.Errorf("invalid project visibility event")
	}
	if !config.NotificationEnable(ctx) {
		return nil
	}
	policies, err := notification.PolicyMgr.GetRelatedPolices(ctx, changed.ProjectID, event.TopicProjectVisibilityChanged)
	if err != nil {
		return err
	}
	payload := &model.Payload{
		Type: event.TopicProjectVisibilityChanged, OccurAt: changed.OccurAt.Unix(),
		Operator: changed.Operator, EventData: &model.EventData{
			ProjectVisibility: &model.ProjectVisibilityChange{
				ProjectID: changed.ProjectID, ProjectName: changed.Project,
				OldPublic: changed.OldPublic, NewPublic: changed.NewPublic,
			},
		},
	}
	return util.SendHookWithPolicies(ctx, policies, payload, event.TopicProjectVisibilityChanged)
}
