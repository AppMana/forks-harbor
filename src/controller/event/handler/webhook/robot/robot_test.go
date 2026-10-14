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

package robot

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
	robotModel "github.com/goharbor/harbor/src/pkg/robot/model"
	"github.com/goharbor/harbor/src/testing/mock"
	testing_notification "github.com/goharbor/harbor/src/testing/pkg/notification/policy"
)

func withPolicies(t *testing.T) *testing_notification.Manager {
	config.InitWithSettings(map[string]any{common.NotificationEnable: true})
	previous := notification.PolicyMgr
	t.Cleanup(func() { notification.PolicyMgr = previous })
	policies := &testing_notification.Manager{}
	notification.PolicyMgr = policies
	return policies
}

func TestRobotEventsReachThePoliciesOfEveryProjectTheRobotNamed(t *testing.T) {
	occurred := time.Unix(1678082923, 0)
	robot := &robotModel.Robot{ID: 7, Name: "robot$pusher", Disabled: true, ExpiresAt: -1}
	for topic, value := range map[string]any{
		event.TopicCreateRobot: &event.CreateRobotEvent{EventType: event.TopicCreateRobot, Robot: robot, Operator: "admin", OccurAt: occurred, ProjectIDs: []int64{3, 5}},
		event.TopicUpdateRobot: &event.UpdateRobotEvent{EventType: event.TopicUpdateRobot, Robot: robot, Operator: "admin", OccurAt: occurred, ProjectIDs: []int64{3, 5}},
		event.TopicDeleteRobot: &event.DeleteRobotEvent{EventType: event.TopicDeleteRobot, Robot: robot, Operator: "admin", OccurAt: occurred, ProjectIDs: []int64{3, 5}},
	} {
		t.Run(topic, func(t *testing.T) {
			policies := withPolicies(t)
			policies.On("GetRelatedPolices", mock.Anything, int64(3), topic).Return([]*policy_model.Policy{}, nil).Once()
			policies.On("GetRelatedPolices", mock.Anything, int64(5), topic).Return([]*policy_model.Policy{}, nil).Once()
			require.NoError(t, (&Handler{}).Handle(context.Background(), value))
			policies.AssertExpectations(t)

			payload, projects, err := robotPayload(value)
			require.NoError(t, err)
			require.Equal(t, []int64{3, 5}, projects)
			require.Equal(t, &notifyModel.Payload{Type: topic, OccurAt: 1678082923, Operator: "admin",
				EventData: &notifyModel.EventData{Robot: &notifyModel.RobotAccount{
					ID: 7, Name: "robot$pusher", Level: "system", ProjectIDs: []int64{3, 5}, Disabled: true, ExpiresAt: -1,
				}}}, payload)
		})
	}
}

func TestProjectRobotIsReportedAtProjectLevel(t *testing.T) {
	payload, _, err := robotPayload(&event.DeleteRobotEvent{EventType: event.TopicDeleteRobot,
		Robot: &robotModel.Robot{ID: 8, Name: "robot$library+scanner", ProjectID: 1}, ProjectIDs: []int64{1}})
	require.NoError(t, err)
	require.Equal(t, "project", payload.EventData.Robot.Level)
	require.Equal(t, int64(1), payload.EventData.Robot.ProjectID)
}

func TestRobotWithoutNamedProjectsLooksUpNoPolicies(t *testing.T) {
	policies := withPolicies(t)
	require.NoError(t, (&Handler{}).Handle(context.Background(), &event.UpdateRobotEvent{
		EventType: event.TopicUpdateRobot, Robot: &robotModel.Robot{ID: 7}}))
	policies.AssertNotCalled(t, "GetRelatedPolices", mock.Anything, mock.Anything, mock.Anything)
}

func TestInvalidRobotEventIsRejected(t *testing.T) {
	withPolicies(t)
	require.Error(t, (&Handler{}).Handle(context.Background(), &event.CreateProjectEvent{}))
	require.Error(t, (&Handler{}).Handle(context.Background(), &event.UpdateRobotEvent{}))
}
