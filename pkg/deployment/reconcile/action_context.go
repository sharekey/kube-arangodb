//
// DISCLAIMER
//
// Copyright 2016-2022 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//

package reconcile

import (
	"context"
	"time"

	"github.com/arangodb/arangosync-client/client"
	"github.com/arangodb/go-driver/agency"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	core "k8s.io/api/core/v1"

	"github.com/arangodb/go-driver"
	backupApi "github.com/arangodb/kube-arangodb/pkg/apis/backup/v1"
	api "github.com/arangodb/kube-arangodb/pkg/apis/deployment/v1"
	"github.com/arangodb/kube-arangodb/pkg/deployment/acs/sutil"
	agencyCache "github.com/arangodb/kube-arangodb/pkg/deployment/agency"
	"github.com/arangodb/kube-arangodb/pkg/deployment/member"
	"github.com/arangodb/kube-arangodb/pkg/deployment/reconciler"
	"github.com/arangodb/kube-arangodb/pkg/util/errors"
	"github.com/arangodb/kube-arangodb/pkg/util/k8sutil"
	inspectorInterface "github.com/arangodb/kube-arangodb/pkg/util/k8sutil/inspector"
	persistentvolumeclaimv1 "github.com/arangodb/kube-arangodb/pkg/util/k8sutil/inspector/persistentvolumeclaim/v1"
	podv1 "github.com/arangodb/kube-arangodb/pkg/util/k8sutil/inspector/pod/v1"
	poddisruptionbudgetv1beta1 "github.com/arangodb/kube-arangodb/pkg/util/k8sutil/inspector/poddisruptionbudget/v1beta1"
	secretv1 "github.com/arangodb/kube-arangodb/pkg/util/k8sutil/inspector/secret/v1"
	servicev1 "github.com/arangodb/kube-arangodb/pkg/util/k8sutil/inspector/service/v1"
	serviceaccountv1 "github.com/arangodb/kube-arangodb/pkg/util/k8sutil/inspector/serviceaccount/v1"
	servicemonitorv1 "github.com/arangodb/kube-arangodb/pkg/util/k8sutil/inspector/servicemonitor/v1"
)

// ActionContext provides methods to the Action implementations
// to control their context.
type ActionContext interface {
	reconciler.DeploymentStatusUpdate
	reconciler.DeploymentAgencyMaintenance
	reconciler.ArangoMemberContext
	reconciler.DeploymentPodRenderer
	reconciler.DeploymentModInterfaces
	reconciler.ArangoAgencyGet
	reconciler.DeploymentInfoGetter
	reconciler.DeploymentClient
	reconciler.DeploymentSyncClient

	member.StateInspectorGetter

	sutil.ACSGetter

	ActionLocalsContext

	// GetMemberStatusByID returns the current member status
	// for the member with given id.
	// Returns member status, true when found, or false
	// when no such member is found.
	GetMemberStatusByID(id string) (api.MemberStatus, bool)
	// GetMemberStatusAndGroupByID returns the current member status and group
	// for the member with given id.
	// Returns member status, true when found, or false
	// when no such member is found.
	GetMemberStatusAndGroupByID(id string) (api.MemberStatus, api.ServerGroup, bool)
	// CreateMember adds a new member to the given group.
	// If ID is non-empty, it will be used, otherwise a new ID is created.
	CreateMember(ctx context.Context, group api.ServerGroup, id string, mods ...CreateMemberMod) (string, error)
	// UpdateMember updates the deployment status wrt the given member.
	UpdateMember(ctx context.Context, member api.MemberStatus) error
	// RemoveMemberByID removes a member with given id.
	RemoveMemberByID(ctx context.Context, id string) error
	// GetImageInfo returns the image info for an image with given name.
	// Returns: (info, infoFound)
	GetImageInfo(imageName string) (api.ImageInfo, bool)
	// GetCurrentImageInfo returns the image info for an current image.
	// Returns: (info, infoFound)
	GetCurrentImageInfo() (api.ImageInfo, bool)
	// SetCurrentImage changes the CurrentImage field in the deployment
	// status to the given image.
	SetCurrentImage(ctx context.Context, imageInfo api.ImageInfo) error
	// DisableScalingCluster disables scaling DBservers and coordinators
	DisableScalingCluster(ctx context.Context) error
	// EnableScalingCluster enables scaling DBservers and coordinators
	EnableScalingCluster(ctx context.Context) error
	// UpdateClusterCondition update status of ArangoDeployment with defined modifier. If action returns True action is taken
	UpdateClusterCondition(ctx context.Context, conditionType api.ConditionType, status bool, reason, message string) error
	// GetBackup receives information about a backup resource
	GetBackup(ctx context.Context, backup string) (*backupApi.ArangoBackup, error)
	// GetName receives information about a deployment name
	GetName() string
	// SelectImage select currently used image by pod
	SelectImage(spec api.DeploymentSpec, status api.DeploymentStatus) (api.ImageInfo, bool)
}

type ActionLocalsContext interface {
	CurrentLocals() api.PlanLocals

	Get(action api.Action, key api.PlanLocalKey) (string, bool)
	Add(key api.PlanLocalKey, value string, override bool) bool
}

// newActionContext creates a new ActionContext implementation.
func newActionContext(log zerolog.Logger, context Context) ActionContext {
	return &actionContext{
		log:     log,
		context: context,
	}
}

// actionContext implements ActionContext
type actionContext struct {
	context      Context
	log          zerolog.Logger
	cachedStatus inspectorInterface.Inspector
	locals       api.PlanLocals
}

func (ac *actionContext) ACS() sutil.ACS {
	return ac.context.ACS()
}

func (ac *actionContext) GetDatabaseAsyncClient(ctx context.Context) (driver.Client, error) {
	return ac.context.GetDatabaseAsyncClient(ctx)
}

func (ac *actionContext) CurrentLocals() api.PlanLocals {
	return ac.locals
}

func (ac *actionContext) Get(action api.Action, key api.PlanLocalKey) (string, bool) {
	return ac.locals.GetWithParent(action.Locals, key)
}

func (ac *actionContext) Add(key api.PlanLocalKey, value string, override bool) bool {
	return ac.locals.Add(key, value, override)
}

func (ac *actionContext) WithArangoMember(cache inspectorInterface.Inspector, timeout time.Duration, name string) reconciler.ArangoMemberModContext {
	return ac.context.WithArangoMember(cache, timeout, name)
}

func (ac *actionContext) WithCurrentArangoMember(name string) reconciler.ArangoMemberModContext {
	return ac.context.WithCurrentArangoMember(name)
}

func (ac *actionContext) GetMembersState() member.StateInspector {
	return ac.context.GetMembersState()
}

func (ac *actionContext) UpdateStatus(ctx context.Context, status api.DeploymentStatus, lastVersion int32, force ...bool) error {
	return ac.context.UpdateStatus(ctx, status, lastVersion, force...)
}

func (ac *actionContext) GetNamespace() string {
	return ac.context.GetNamespace()
}

func (ac *actionContext) GetAgencyClientsWithPredicate(ctx context.Context, predicate func(id string) bool) ([]driver.Connection, error) {
	return ac.context.GetAgencyClientsWithPredicate(ctx, predicate)
}

func (ac *actionContext) GetStatus() (api.DeploymentStatus, int32) {
	return ac.context.GetStatus()
}

func (ac *actionContext) GetStatusSnapshot() api.DeploymentStatus {
	return ac.context.GetStatusSnapshot()
}

func (ac *actionContext) GenerateMemberEndpoint(group api.ServerGroup, member api.MemberStatus) (string, error) {
	return ac.context.GenerateMemberEndpoint(group, member)
}

func (ac *actionContext) GetAgencyHealth() (agencyCache.Health, bool) {
	return ac.context.GetAgencyHealth()
}

func (ac *actionContext) GetAgencyCache() (agencyCache.State, bool) {
	return ac.context.GetAgencyCache()
}

func (ac *actionContext) RenderPodForMemberFromCurrent(ctx context.Context, acs sutil.ACS, memberID string) (*core.Pod, error) {
	return ac.context.RenderPodForMemberFromCurrent(ctx, acs, memberID)
}

func (ac *actionContext) RenderPodTemplateForMemberFromCurrent(ctx context.Context, acs sutil.ACS, memberID string) (*core.PodTemplateSpec, error) {
	return ac.context.RenderPodTemplateForMemberFromCurrent(ctx, acs, memberID)
}

func (ac *actionContext) SetAgencyMaintenanceMode(ctx context.Context, enabled bool) error {
	return ac.context.SetAgencyMaintenanceMode(ctx, enabled)
}

func (ac *actionContext) RenderPodForMember(ctx context.Context, acs sutil.ACS, spec api.DeploymentSpec, status api.DeploymentStatus, memberID string, imageInfo api.ImageInfo) (*core.Pod, error) {
	return ac.context.RenderPodForMember(ctx, acs, spec, status, memberID, imageInfo)
}

func (ac *actionContext) RenderPodTemplateForMember(ctx context.Context, acs sutil.ACS, spec api.DeploymentSpec, status api.DeploymentStatus, memberID string, imageInfo api.ImageInfo) (*core.PodTemplateSpec, error) {
	return ac.context.RenderPodTemplateForMember(ctx, acs, spec, status, memberID, imageInfo)
}

func (ac *actionContext) SelectImage(spec api.DeploymentSpec, status api.DeploymentStatus) (api.ImageInfo, bool) {
	return ac.context.SelectImage(spec, status)
}

func (ac *actionContext) GetCachedStatus() inspectorInterface.Inspector {
	return ac.cachedStatus
}

func (ac *actionContext) GetName() string {
	return ac.context.GetName()
}

func (ac *actionContext) GetBackup(ctx context.Context, backup string) (*backupApi.ArangoBackup, error) {
	return ac.context.GetBackup(ctx, backup)
}

func (ac *actionContext) WithStatusUpdateErr(ctx context.Context, action reconciler.DeploymentStatusUpdateErrFunc, force ...bool) error {
	return ac.context.WithStatusUpdateErr(ctx, action, force...)
}

func (ac *actionContext) WithStatusUpdate(ctx context.Context, action reconciler.DeploymentStatusUpdateFunc, force ...bool) error {
	return ac.context.WithStatusUpdate(ctx, action, force...)
}

func (ac *actionContext) SecretsModInterface() secretv1.ModInterface {
	return ac.context.SecretsModInterface()
}

func (ac *actionContext) PodsModInterface() podv1.ModInterface {
	return ac.context.PodsModInterface()
}

func (ac *actionContext) ServiceAccountsModInterface() serviceaccountv1.ModInterface {
	return ac.context.ServiceAccountsModInterface()
}

func (ac *actionContext) ServicesModInterface() servicev1.ModInterface {
	return ac.context.ServicesModInterface()
}

func (ac *actionContext) PersistentVolumeClaimsModInterface() persistentvolumeclaimv1.ModInterface {
	return ac.context.PersistentVolumeClaimsModInterface()
}

func (ac *actionContext) PodDisruptionBudgetsModInterface() poddisruptionbudgetv1beta1.ModInterface {
	return ac.context.PodDisruptionBudgetsModInterface()
}

func (ac *actionContext) ServiceMonitorsModInterface() servicemonitorv1.ModInterface {
	return ac.context.ServiceMonitorsModInterface()
}

func (ac *actionContext) UpdateClusterCondition(ctx context.Context, conditionType api.ConditionType, status bool, reason, message string) error {
	return ac.context.WithStatusUpdate(ctx, func(s *api.DeploymentStatus) bool {
		return s.Conditions.Update(conditionType, status, reason, message)
	})
}

func (ac *actionContext) GetAPIObject() k8sutil.APIObject {
	return ac.context.GetAPIObject()
}

func (ac *actionContext) CreateEvent(evt *k8sutil.Event) {
	ac.context.CreateEvent(evt)
}

// Gets the specified mode of deployment
func (ac *actionContext) GetMode() api.DeploymentMode {
	return ac.context.GetSpec().GetMode()
}

func (ac *actionContext) GetSpec() api.DeploymentSpec {
	return ac.context.GetSpec()
}

// GetDatabaseClient returns a cached client for the entire database (cluster coordinators or single server),
// creating one if needed.
func (ac *actionContext) GetDatabaseClient(ctx context.Context) (driver.Client, error) {
	c, err := ac.context.GetDatabaseClient(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return c, nil
}

// GetServerClient returns a cached client for a specific server.
func (ac *actionContext) GetServerClient(ctx context.Context, group api.ServerGroup, id string) (driver.Client, error) {
	c, err := ac.context.GetServerClient(ctx, group, id)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return c, nil
}

// GetAgencyClients returns a client connection for every agency member.
func (ac *actionContext) GetAgencyClients(ctx context.Context) ([]driver.Connection, error) {
	c, err := ac.context.GetAgencyClients(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return c, nil
}

// GetAgency returns a connection to the agency.
func (ac *actionContext) GetAgency(ctx context.Context, agencyIDs ...string) (agency.Agency, error) {
	a, err := ac.context.GetAgency(ctx, agencyIDs...)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return a, nil
}

// GetSyncServerClient returns a cached client for a specific arangosync server.
func (ac *actionContext) GetSyncServerClient(ctx context.Context, group api.ServerGroup, id string) (client.API, error) {
	c, err := ac.context.GetSyncServerClient(ctx, group, id)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return c, nil
}

// GetMemberStatusByID returns the current member status
// for the member with given id.
// Returns member status, true when found, or false
// when no such member is found.
func (ac *actionContext) GetMemberStatusByID(id string) (api.MemberStatus, bool) {
	status, _ := ac.context.GetStatus()
	m, _, ok := status.Members.ElementByID(id)
	return m, ok
}

// GetMemberStatusAndGroupByID returns the current member status and group
// for the member with given id.
// Returns member status, true when found, or false
// when no such member is found.
func (ac *actionContext) GetMemberStatusAndGroupByID(id string) (api.MemberStatus, api.ServerGroup, bool) {
	status, _ := ac.context.GetStatus()
	return status.Members.ElementByID(id)
}

// CreateMember adds a new member to the given group.
// If ID is non-empty, it will be used, otherwise a new ID is created.
func (ac *actionContext) CreateMember(ctx context.Context, group api.ServerGroup, id string, mods ...CreateMemberMod) (string, error) {
	result, err := ac.context.CreateMember(ctx, group, id, mods...)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return result, nil
}

// UpdateMember updates the deployment status wrt the given member.
func (ac *actionContext) UpdateMember(ctx context.Context, member api.MemberStatus) error {
	status, lastVersion := ac.context.GetStatus()
	_, group, found := status.Members.ElementByID(member.ID)
	if !found {
		return errors.WithStack(errors.Newf("Member %s not found", member.ID))
	}
	if err := status.Members.Update(member, group); err != nil {
		return errors.WithStack(err)
	}
	if err := ac.context.UpdateStatus(ctx, status, lastVersion); err != nil {
		log.Debug().Err(err).Msg("Updating CR status failed")
		return errors.WithStack(err)
	}
	return nil
}

// RemoveMemberByID removes a member with given id.
func (ac *actionContext) RemoveMemberByID(ctx context.Context, id string) error {
	status, lastVersion := ac.context.GetStatus()
	_, group, found := status.Members.ElementByID(id)
	if !found {
		return nil
	}
	if err := status.Members.RemoveByID(id, group); err != nil {
		log.Debug().Err(err).Str("group", group.AsRole()).Msg("Failed to remove member")
		return errors.WithStack(err)
	}
	// Save removed member
	if err := ac.context.UpdateStatus(ctx, status, lastVersion); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// GetImageInfo returns the image info for an image with given name.
// Returns: (info, infoFound)
func (ac *actionContext) GetImageInfo(imageName string) (api.ImageInfo, bool) {
	status, _ := ac.context.GetStatus()
	return status.Images.GetByImage(imageName)
}

// GetImageInfo returns the image info for an current image.
// Returns: (info, infoFound)
func (ac *actionContext) GetCurrentImageInfo() (api.ImageInfo, bool) {
	status, _ := ac.context.GetStatus()

	if status.CurrentImage == nil {
		return api.ImageInfo{}, false
	}

	return *status.CurrentImage, true
}

// SetCurrentImage changes the CurrentImage field in the deployment
// status to the given image.
func (ac *actionContext) SetCurrentImage(ctx context.Context, imageInfo api.ImageInfo) error {
	return ac.context.WithStatusUpdate(ctx, func(s *api.DeploymentStatus) bool {
		if s.CurrentImage == nil || s.CurrentImage.Image != imageInfo.Image {
			s.CurrentImage = &imageInfo
			return true
		}
		return false
	}, true)
}

// DisableScalingCluster disables scaling DBservers and coordinators
func (ac *actionContext) DisableScalingCluster(ctx context.Context) error {
	return ac.context.DisableScalingCluster(ctx)
}

// EnableScalingCluster enables scaling DBservers and coordinators
func (ac *actionContext) EnableScalingCluster(ctx context.Context) error {
	return ac.context.EnableScalingCluster(ctx)
}
