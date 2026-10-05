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

package notification

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/goharbor/harbor/src/controller/event/metadata"
	"github.com/goharbor/harbor/src/pkg/artifact"
	"github.com/goharbor/harbor/src/pkg/notification"
	notifierEvent "github.com/goharbor/harbor/src/pkg/notifier/event"
)

type NotificationMiddlewareTestSuite struct {
	suite.Suite
}

func (suite *NotificationMiddlewareTestSuite) TestMiddleware() {
	next := func() http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			notification.AddEvent(r.Context(), &metadata.DeleteArtifactEventMetadata{
				Ctx: context.Background(),
				Artifact: &artifact.Artifact{
					ProjectID:      1,
					RepositoryID:   2,
					RepositoryName: "library/hello-world",
				},
				Tags: []string{"latest"},
			})
		})
	}
	path := fmt.Sprintf("/v2/library/photon/manifests/latest")
	req := httptest.NewRequest(http.MethodPatch, path, nil)
	res := httptest.NewRecorder()
	Middleware()(next()).ServeHTTP(res, req)
	suite.Equal(http.StatusAccepted, res.Code)
}

func (suite *NotificationMiddlewareTestSuite) TestMiddlewareMustNotify() {
	next := func() http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			notification.AddEvent(r.Context(), &metadata.DeleteArtifactEventMetadata{
				Ctx: context.Background(),
				Artifact: &artifact.Artifact{
					ProjectID:      1,
					RepositoryID:   2,
					RepositoryName: "library/hello-world",
				},
				Tags: []string{"latest"},
			}, true)
		})
	}
	path := fmt.Sprintf("/v2/library/photon/manifests/latest")
	req := httptest.NewRequest(http.MethodPatch, path, nil)
	res := httptest.NewRecorder()
	Middleware()(next()).ServeHTTP(res, req)
	suite.Equal(http.StatusInternalServerError, res.Code)
}

func TestNotificationMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, &NotificationMiddlewareTestSuite{})
}

// The transaction middleware runs inside notification middleware and replaces
// a buffered success with 500 if commit fails. Such responses must not publish.
func TestVisibilityEventIsNotResolvedAfterFailedTransaction(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusInternalServerError} {
		resolved := make(chan struct{}, 1)
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			notification.AddEvent(r.Context(), &visibilityResolutionProbe{resolved: resolved})
			w.WriteHeader(status)
		})
		Middleware()(next).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPut, "/api/v2.0/projects/42", nil))
		if status == http.StatusOK {
			select {
			case <-resolved:
			case <-time.After(time.Second):
				t.Fatal("committed event was not resolved")
			}
		} else {
			select {
			case <-resolved:
				t.Fatal("rolled-back event was resolved")
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
}

type visibilityResolutionProbe struct{ resolved chan struct{} }

func (p *visibilityResolutionProbe) Resolve(e *notifierEvent.Event) error {
	p.resolved <- struct{}{}
	e.Topic = "PROJECT_VISIBILITY_CHANGED"
	return nil
}
