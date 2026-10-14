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

	"github.com/stretchr/testify/require"

	"github.com/goharbor/harbor/src/common"
	"github.com/goharbor/harbor/src/common/security"
	events "github.com/goharbor/harbor/src/controller/event"
	"github.com/goharbor/harbor/src/lib/config"
	_ "github.com/goharbor/harbor/src/pkg/config/inmemory"
	"github.com/goharbor/harbor/src/pkg/notifier/event"
	"github.com/goharbor/harbor/src/pkg/robot/model"
	securitytesting "github.com/goharbor/harbor/src/testing/common/security"
)

func TestRobotEventsCarryTheProjectsOfTheirPermissions(t *testing.T) {
	config.InitWithSettings(map[string]any{common.RobotPrefix: "robot$"})
	sc := &securitytesting.Context{}
	sc.On("GetUsername").Return("admin")
	ctx := security.NewContext(context.Background(), sc)

	created := &event.Event{}
	require.NoError(t, (&CreateRobotEventMetadata{Ctx: ctx, Robot: &model.Robot{ID: 1, Name: "a"},
		ProjectIDs: []int64{3}}).Resolve(created))
	require.Equal(t, []int64{3}, created.Data.(*events.CreateRobotEvent).ProjectIDs)

	deleted := &event.Event{}
	require.NoError(t, (&DeleteRobotEventMetadata{Ctx: ctx, Robot: &model.Robot{ID: 1, Name: "a"},
		ProjectIDs: []int64{3, 4}}).Resolve(deleted))
	require.Equal(t, []int64{3, 4}, deleted.Data.(*events.DeleteRobotEvent).ProjectIDs)

	updated := &event.Event{}
	require.NoError(t, (&UpdateRobotEventMetadata{Ctx: ctx, Robot: &model.Robot{ID: 1, Name: "robot$a", Disabled: true},
		ProjectIDs: []int64{5}}).Resolve(updated))
	require.Equal(t, events.TopicUpdateRobot, updated.Topic)
	data := updated.Data.(*events.UpdateRobotEvent)
	require.Equal(t, events.TopicUpdateRobot, data.EventType)
	require.Equal(t, []int64{5}, data.ProjectIDs)
	require.Equal(t, "admin", data.Operator)
	// The controller updates a robot read through the API, whose name already has the prefix.
	require.Equal(t, "robot$a", data.Robot.Name)
	require.True(t, data.Robot.Disabled)
	require.False(t, data.OccurAt.IsZero())
}
