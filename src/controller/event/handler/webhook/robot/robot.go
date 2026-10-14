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
	"errors"
	"fmt"
	"time"

	"github.com/goharbor/harbor/src/controller/event"
	"github.com/goharbor/harbor/src/controller/event/handler/util"
	"github.com/goharbor/harbor/src/lib/config"
	"github.com/goharbor/harbor/src/pkg/notification"
	"github.com/goharbor/harbor/src/pkg/notifier/model"
	robotModel "github.com/goharbor/harbor/src/pkg/robot/model"
)

// Handler delivers robot account changes to the webhook policies of every
// project the robot's permissions name, through Harbor's job/retry pipeline.
// A system robot belongs to no project, so its projects are those it can act in.
type Handler struct{}

func (*Handler) Name() string     { return "RobotWebhook" }
func (*Handler) IsStateful() bool { return false }

func (*Handler) Handle(ctx context.Context, value any) error {
	payload, projectIDs, err := robotPayload(value)
	if err != nil {
		return err
	}
	if !config.NotificationEnable(ctx) {
		return nil
	}
	var errs []error
	for _, projectID := range projectIDs {
		policies, err := notification.PolicyMgr.GetRelatedPolices(ctx, projectID, payload.Type)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if len(policies) == 0 {
			continue
		}
		if err := util.SendHookWithPolicies(ctx, policies, payload, payload.Type); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func robotPayload(value any) (*model.Payload, []int64, error) {
	var (
		topic      string
		robot      *robotModel.Robot
		operator   string
		occurAt    time.Time
		projectIDs []int64
	)
	switch e := value.(type) {
	case *event.CreateRobotEvent:
		topic, robot, operator, occurAt, projectIDs = event.TopicCreateRobot, e.Robot, e.Operator, e.OccurAt, e.ProjectIDs
	case *event.UpdateRobotEvent:
		topic, robot, operator, occurAt, projectIDs = event.TopicUpdateRobot, e.Robot, e.Operator, e.OccurAt, e.ProjectIDs
	case *event.DeleteRobotEvent:
		topic, robot, operator, occurAt, projectIDs = event.TopicDeleteRobot, e.Robot, e.Operator, e.OccurAt, e.ProjectIDs
	default:
		return nil, nil, fmt.Errorf("invalid robot event type %T", value)
	}
	if robot == nil {
		return nil, nil, fmt.Errorf("robot event %s without robot", topic)
	}
	level := "system"
	if robot.ProjectID > 0 {
		level = "project"
	}
	return &model.Payload{
		Type: topic, OccurAt: occurAt.Unix(), Operator: operator,
		EventData: &model.EventData{Robot: &model.RobotAccount{
			ID: robot.ID, Name: robot.Name, Level: level, ProjectID: robot.ProjectID,
			ProjectIDs: projectIDs, Disabled: robot.Disabled, ExpiresAt: robot.ExpiresAt,
		}},
	}, projectIDs, nil
}
