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

package model

import (
	"github.com/goharbor/harbor/src/controller/event/model"
	policy_model "github.com/goharbor/harbor/src/pkg/notification/policy/model"
)

// HookEvent is hook related event data to publish
type HookEvent struct {
	ProjectID int64
	PolicyID  int64
	EventType string
	Target    *policy_model.EventTarget
	Payload   *Payload
}

// Payload of notification event
type Payload struct {
	Type      string     `json:"type"`
	OccurAt   int64      `json:"occur_at"`
	Operator  string     `json:"operator"`
	EventData *EventData `json:"event_data,omitempty"`
}

// EventData of notification event payload
type EventData struct {
	ProjectVisibility *ProjectVisibilityChange `json:"project_visibility,omitempty"`
	Project           *ProjectRef              `json:"project,omitempty"`
	Robot             *RobotAccount            `json:"robot,omitempty"`
	Resources         []*Resource              `json:"resources,omitempty"`
	Repository        *Repository              `json:"repository,omitempty"`
	Replication       *model.Replication       `json:"replication,omitempty"`
	Retention         *model.Retention         `json:"retention,omitempty"`
	Scan              *model.Scan              `json:"scan,omitempty"`
	Custom            map[string]string        `json:"custom_attributes,omitempty"`
}

// Resource describe infos of resource triggered notification
type Resource struct {
	Digest       string         `json:"digest,omitempty"`
	Tag          string         `json:"tag,omitempty"`
	ResourceURL  string         `json:"resource_url,omitempty"`
	ScanOverview map[string]any `json:"scan_overview,omitempty"`
	SBOMOverview map[string]any `json:"sbom_overview,omitempty"`
}

// Repository info of notification event
type Repository struct {
	DateCreated  int64  `json:"date_created,omitempty"`
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	RepoFullName string `json:"repo_full_name"`
	RepoType     string `json:"repo_type"`
}

// ProjectVisibilityChange describes a project-wide publication permission change.
type ProjectVisibilityChange struct {
	ProjectID   int64  `json:"project_id"`
	ProjectName string `json:"project_name"`
	OldPublic   bool   `json:"old_public"`
	NewPublic   bool   `json:"new_public"`
}

// ProjectRef identifies a project, e.g. a deleted one.
type ProjectRef struct {
	ProjectID   int64  `json:"project_id"`
	ProjectName string `json:"project_name"`
}

// RobotAccount describes a created, updated or deleted robot account. The
// secret is never included.
type RobotAccount struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	// Level is "system" or "project".
	Level string `json:"level"`
	// ProjectID is set for project level robots.
	ProjectID int64 `json:"project_id,omitempty"`
	// ProjectIDs are the projects the robot's permissions name.
	ProjectIDs []int64 `json:"project_ids,omitempty"`
	Disabled   bool    `json:"disabled"`
	ExpiresAt  int64   `json:"expires_at"`
}
