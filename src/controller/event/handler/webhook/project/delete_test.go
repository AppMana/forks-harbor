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
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/goharbor/harbor/src/common"
	"github.com/goharbor/harbor/src/controller/event"
	"github.com/goharbor/harbor/src/lib/config"
	_ "github.com/goharbor/harbor/src/pkg/config/inmemory"
	"github.com/goharbor/harbor/src/pkg/notification"
	policy_model "github.com/goharbor/harbor/src/pkg/notification/policy/model"
	notifyModel "github.com/goharbor/harbor/src/pkg/notifier/model"
	"github.com/goharbor/harbor/src/testing/mock"
	testing_notification "github.com/goharbor/harbor/src/testing/pkg/notification/policy"
)

// Project deletion is a soft delete: the project's policies survive it, so
// they can observe the deletion itself.
func TestProjectDeletionReachesThePoliciesOfTheDeletedProject(t *testing.T) {
	config.InitWithSettings(map[string]any{common.NotificationEnable: true})
	previous := notification.PolicyMgr
	t.Cleanup(func() { notification.PolicyMgr = previous })
	policies := &testing_notification.Manager{}
	notification.PolicyMgr = policies
	policies.On("GetRelatedPolices", mock.Anything, int64(42), event.TopicDeleteProject).Return([]*policy_model.Policy{}, nil).Once()

	deleted := &event.DeleteProjectEvent{EventType: event.TopicDeleteProject, ProjectID: 42, Project: "library",
		Operator: "admin", OccurAt: time.Unix(1678082923, 0)}
	require.NoError(t, (&DeleteHandler{}).Handle(context.Background(), deleted))
	policies.AssertExpectations(t)
	require.Equal(t, &notifyModel.Payload{Type: event.TopicDeleteProject, OccurAt: 1678082923, Operator: "admin",
		EventData: &notifyModel.EventData{Project: &notifyModel.ProjectRef{ProjectID: 42, ProjectName: "library"}}},
		deletionPayload(deleted))
	require.Error(t, (&DeleteHandler{}).Handle(context.Background(), &event.CreateProjectEvent{}))
}
