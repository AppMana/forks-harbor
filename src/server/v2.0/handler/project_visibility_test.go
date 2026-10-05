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

package handler

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/goharbor/harbor/src/common/security"
	eventmetadata "github.com/goharbor/harbor/src/controller/event/metadata"
	"github.com/goharbor/harbor/src/controller/project"
	metadatactl "github.com/goharbor/harbor/src/controller/project/metadata"
	"github.com/goharbor/harbor/src/pkg/notification"
	projectmodels "github.com/goharbor/harbor/src/pkg/project/models"
	"github.com/goharbor/harbor/src/server/v2.0/models"
	operations "github.com/goharbor/harbor/src/server/v2.0/restapi/operations/project"
	metaops "github.com/goharbor/harbor/src/server/v2.0/restapi/operations/project_metadata"
	securitytesting "github.com/goharbor/harbor/src/testing/common/security"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type visibilityProjectController struct {
	project.Controller
	project *projectmodels.Project
	err     error
}

func (c visibilityProjectController) Get(context.Context, any, ...project.Option) (*projectmodels.Project, error) {
	return c.project, nil
}
func (c visibilityProjectController) Update(context.Context, *projectmodels.Project) error {
	return c.err
}

type visibilityMetadataController struct {
	metadatactl.Controller
	err error
}

func (c visibilityMetadataController) Add(context.Context, int64, map[string]string) error {
	return c.err
}
func (c visibilityMetadataController) Update(context.Context, int64, map[string]string) error {
	return c.err
}
func (c visibilityMetadataController) Delete(context.Context, int64, ...string) error { return c.err }

func TestVisibilityEventsCoverEveryMutationAPI(t *testing.T) {
	for _, route := range []string{"project-update", "metadata-add", "metadata-update", "metadata-delete"} {
		for _, old := range []bool{false, true} {
			for _, failure := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/old=%t/failure=%t", route, old, failure), func(t *testing.T) {
					sc := &securitytesting.Context{}
					sc.On("Can", mock.Anything, mock.Anything, mock.Anything).Return(true)
					sc.On("GetUsername").Return("admin")
					queue := notification.NewEventCtx()
					ctx := notification.NewContext(security.NewContext(context.Background(), sc), queue)
					var mutationErr error
					if failure {
						mutationErr = errors.New("mutation rolled back")
					}
					p := &projectmodels.Project{ProjectID: 42, Name: "workspace-public-release",
						Metadata: map[string]string{"public": fmt.Sprint(old)}}
					ctl := visibilityProjectController{project: p, err: mutationErr}
					metadata := visibilityMetadataController{err: mutationErr}
					api := &projectMetadataAPI{ctl: metadata, proCtl: ctl}
					next := false
					switch route {
					case "project-update":
						(&projectAPI{projectCtl: ctl}).UpdateProject(ctx, operations.UpdateProjectParams{
							ProjectNameOrID: "42", Project: &models.ProjectReq{
								Metadata: &models.ProjectMetadata{Public: "false"}}})
					case "metadata-add":
						next = true
						api.AddProjectMetadatas(ctx, metaops.AddProjectMetadatasParams{
							ProjectNameOrID: "42", Metadata: map[string]string{"public": "true"}})
					case "metadata-update":
						api.UpdateProjectMetadata(ctx, metaops.UpdateProjectMetadataParams{
							ProjectNameOrID: "42", MetaName: "public", Metadata: map[string]string{"public": "false"}})
					case "metadata-delete":
						api.DeleteProjectMetadata(ctx, metaops.DeleteProjectMetadataParams{
							ProjectNameOrID: "42", MetaName: "public"})
					}
					if failure || old == next {
						require.Zero(t, queue.Events.Len())
					} else {
						require.Equal(t, 1, queue.Events.Len())
						event := queue.Events.Front().Value.(*eventmetadata.ProjectVisibilityChangedMetadata)
						require.Equal(t, old, event.OldPublic)
						require.Equal(t, next, event.NewPublic)
						require.Equal(t, int64(42), event.ProjectID)
					}
				})
			}
		}
	}
}
