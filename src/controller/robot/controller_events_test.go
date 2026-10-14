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
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/goharbor/harbor/src/common"
	"github.com/goharbor/harbor/src/common/utils/test"
	"github.com/goharbor/harbor/src/controller/event/metadata"
	"github.com/goharbor/harbor/src/lib/config"
	"github.com/goharbor/harbor/src/pkg/notification"
	"github.com/goharbor/harbor/src/pkg/permission/types"
	proModels "github.com/goharbor/harbor/src/pkg/project/models"
	rbac_model "github.com/goharbor/harbor/src/pkg/rbac/model"
	"github.com/goharbor/harbor/src/pkg/robot/model"
	"github.com/goharbor/harbor/src/testing/mock"
	"github.com/goharbor/harbor/src/testing/pkg/project"
	"github.com/goharbor/harbor/src/testing/pkg/rbac"
	"github.com/goharbor/harbor/src/testing/pkg/robot"
)

func scopes(values ...string) []*rbac_model.UniversalRolePermission {
	permissions := make([]*rbac_model.UniversalRolePermission, 0, len(values))
	for _, scope := range values {
		permissions = append(permissions, &rbac_model.UniversalRolePermission{
			RoleType: ROBOTTYPE, RoleID: 7, Scope: scope, Resource: "repository", Action: "pull",
		})
	}
	return permissions
}

func eventQueue() (context.Context, *notification.EventCtx) {
	queue := notification.NewEventCtx()
	return notification.NewContext(context.Background(), queue), queue
}

func pullPermission(namespace string) []*Permission {
	return []*Permission{{Kind: LEVELPROJECT, Namespace: namespace,
		Access: []*types.Policy{{Resource: "repository", Action: "pull"}}}}
}

// A system robot's permissions name its projects; "/project/*" and "/system"
// name none. Project webhook policies can only observe the projects named.
func TestCreateQueuesRobotEventForEveryProjectInItsPermissions(t *testing.T) {
	keyPath := t.TempDir() + "/secretkey"
	_, err := test.GenerateKey(keyPath)
	require.NoError(t, err)
	defer os.Remove(keyPath)
	t.Setenv("KEY_PATH", keyPath)
	config.InitWithSettings(map[string]any{common.RobotTokenDuration: "30"})

	projectMgr, rbacMgr, robotMgr := &project.Manager{}, &rbac.Manager{}, &robot.Manager{}
	c := controller{robotMgr: robotMgr, rbacMgr: rbacMgr, proMgr: projectMgr}
	projectMgr.On("Get", mock.Anything, "first").Return(&proModels.Project{ProjectID: 3, Name: "first"}, nil)
	robotMgr.On("Create", mock.Anything, mock.Anything).Return(int64(7), nil)
	rbacMgr.On("CreateRbacPolicy", mock.Anything, mock.Anything).Return(int64(1), nil)
	rbacMgr.On("CreatePermission", mock.Anything, mock.Anything).Return(int64(1), nil)
	rbacMgr.On("GetPermissionsByRole", mock.Anything, ROBOTTYPE, int64(7)).
		Return(scopes("/project/3", "/project/3", "/project/*", "/system"), nil)

	ctx, queue := eventQueue()
	_, _, err = c.Create(ctx, &Robot{Robot: model.Robot{Name: "pusher", Duration: -1},
		Level: LEVELSYSTEM, Permissions: pullPermission("first")})
	require.NoError(t, err)
	require.Equal(t, 1, queue.Events.Len())
	created := queue.Events.Front().Value.(*metadata.CreateRobotEventMetadata)
	require.Equal(t, []int64{3}, created.ProjectIDs)
}

func TestUpdateQueuesRobotEventForProjectsBeforeAndAfterPermissionChange(t *testing.T) {
	projectMgr, rbacMgr, robotMgr := &project.Manager{}, &rbac.Manager{}, &robot.Manager{}
	c := controller{robotMgr: robotMgr, rbacMgr: rbacMgr, proMgr: projectMgr}
	projectMgr.On("Get", mock.Anything, "second").Return(&proModels.Project{ProjectID: 5, Name: "second"}, nil)
	robotMgr.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	rbacMgr.On("GetPermissionsByRole", mock.Anything, ROBOTTYPE, int64(7)).Return(scopes("/project/3"), nil).Once()
	rbacMgr.On("DeletePermissionsByRole", mock.Anything, ROBOTTYPE, int64(7)).Return(nil)
	rbacMgr.On("CreateRbacPolicy", mock.Anything, mock.Anything).Return(int64(1), nil)
	rbacMgr.On("CreatePermission", mock.Anything, mock.Anything).Return(int64(1), nil)
	rbacMgr.On("GetPermissionsByRole", mock.Anything, ROBOTTYPE, int64(7)).Return(scopes("/project/5"), nil).Once()

	ctx, queue := eventQueue()
	r := &Robot{Robot: model.Robot{ID: 7, Name: "robot$pusher", Disabled: true},
		Level: LEVELSYSTEM, Permissions: pullPermission("second")}
	require.NoError(t, c.Update(ctx, r, &Option{WithPermission: true}))
	require.Equal(t, 1, queue.Events.Len())
	updated := queue.Events.Front().Value.(*metadata.UpdateRobotEventMetadata)
	require.Equal(t, []int64{3, 5}, updated.ProjectIDs)
	require.Equal(t, "robot$pusher", updated.Robot.Name)
	require.True(t, updated.Robot.Disabled)
	rbacMgr.AssertExpectations(t)
}

// Secret refresh and enable/disable update the robot row only.
func TestUpdateWithoutPermissionsQueuesRobotEventForCurrentProjects(t *testing.T) {
	rbacMgr, robotMgr := &rbac.Manager{}, &robot.Manager{}
	c := controller{robotMgr: robotMgr, rbacMgr: rbacMgr, proMgr: &project.Manager{}}
	robotMgr.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	rbacMgr.On("GetPermissionsByRole", mock.Anything, ROBOTTYPE, int64(7)).Return(scopes("/project/4"), nil).Once()

	ctx, queue := eventQueue()
	require.NoError(t, c.Update(ctx, &Robot{Robot: model.Robot{ID: 7, Name: "robot$pusher", Secret: "rotated"}}, nil))
	require.Equal(t, 1, queue.Events.Len())
	require.Equal(t, []int64{4}, queue.Events.Front().Value.(*metadata.UpdateRobotEventMetadata).ProjectIDs)
	rbacMgr.AssertExpectations(t)
}

func TestFailedUpdateQueuesNoRobotEvent(t *testing.T) {
	rbacMgr, robotMgr := &rbac.Manager{}, &robot.Manager{}
	c := controller{robotMgr: robotMgr, rbacMgr: rbacMgr, proMgr: &project.Manager{}}
	robotMgr.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("update failed"))
	rbacMgr.On("GetPermissionsByRole", mock.Anything, ROBOTTYPE, int64(7)).Return(scopes("/project/4"), nil)

	ctx, queue := eventQueue()
	require.Error(t, c.Update(ctx, &Robot{Robot: model.Robot{ID: 7}}, nil))
	require.Zero(t, queue.Events.Len())
}

// Deletion removes the permissions, so the projects are read before it.
func TestDeleteQueuesRobotEventForProjectsReadBeforePermissionsAreRemoved(t *testing.T) {
	rbacMgr, robotMgr := &rbac.Manager{}, &robot.Manager{}
	c := controller{robotMgr: robotMgr, rbacMgr: rbacMgr, proMgr: &project.Manager{}}
	robotMgr.On("Get", mock.Anything, int64(7)).Return(&model.Robot{ID: 7, Name: "pusher"}, nil)
	robotMgr.On("Delete", mock.Anything, int64(7)).Return(nil)
	removed := false
	rbacMgr.On("GetPermissionsByRole", mock.Anything, ROBOTTYPE, int64(7)).
		Run(func(mock.Arguments) { require.False(t, removed, "permissions read after removal") }).
		Return(scopes("/project/2", "/project/9"), nil)
	rbacMgr.On("DeletePermissionsByRole", mock.Anything, ROBOTTYPE, int64(7)).
		Run(func(mock.Arguments) { removed = true }).Return(nil)

	ctx, queue := eventQueue()
	require.NoError(t, c.Delete(ctx, 7))
	require.Equal(t, 1, queue.Events.Len())
	require.Equal(t, []int64{2, 9}, queue.Events.Front().Value.(*metadata.DeleteRobotEventMetadata).ProjectIDs)
}
