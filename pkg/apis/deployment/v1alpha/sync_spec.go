//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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
// Author Ewout Prangsma
//

package v1alpha

import (
	"github.com/pkg/errors"

	"github.com/arangodb/kube-arangodb/pkg/util"
)

// SyncSpec holds dc2dc replication specific configuration settings
type SyncSpec struct {
	Enabled *bool `json:"enabled,omitempty"`

	ExternalAccess SyncExternalAccessSpec `json:"externalAccess"`
	Authentication SyncAuthenticationSpec `json:"auth"`
	TLS            TLSSpec                `json:"tls"`
	Monitoring     MonitoringSpec         `json:"monitoring"`
}

// IsEnabled returns the value of enabled.
func (s SyncSpec) IsEnabled() bool {
	return util.BoolOrDefault(s.Enabled)
}

// Validate the given spec
func (s SyncSpec) Validate(mode DeploymentMode) error {
	if s.IsEnabled() && !mode.SupportsSync() {
		return maskAny(errors.Wrapf(ValidationError, "Cannot enable sync with mode: '%s'", mode))
	}
	if s.IsEnabled() {
		if err := s.ExternalAccess.Validate(); err != nil {
			return maskAny(err)
		}
		if err := s.Authentication.Validate(); err != nil {
			return maskAny(err)
		}
		if err := s.TLS.Validate(); err != nil {
			return maskAny(err)
		}
	}
	if err := s.Monitoring.Validate(); err != nil {
		return maskAny(err)
	}
	return nil
}

// SetDefaults fills in missing defaults
func (s *SyncSpec) SetDefaults(defaultJWTSecretName, defaultClientAuthCASecretName, defaultTLSCASecretName, defaultMonitoringSecretName string) {
	s.ExternalAccess.SetDefaults()
	s.Authentication.SetDefaults(defaultJWTSecretName, defaultClientAuthCASecretName)
	s.TLS.SetDefaults(defaultTLSCASecretName)
	s.Monitoring.SetDefaults(defaultMonitoringSecretName)
}

// SetDefaultsFrom fills unspecified fields with a value from given source spec.
func (s *SyncSpec) SetDefaultsFrom(source SyncSpec) {
	if s.Enabled == nil {
		s.Enabled = util.NewBoolOrNil(source.Enabled)
	}
	s.ExternalAccess.SetDefaultsFrom(source.ExternalAccess)
	s.Authentication.SetDefaultsFrom(source.Authentication)
	s.TLS.SetDefaultsFrom(source.TLS)
	s.Monitoring.SetDefaultsFrom(source.Monitoring)
}

// ResetImmutableFields replaces all immutable fields in the given target with values from the source spec.
// It returns a list of fields that have been reset.
// Field names are relative to given field prefix.
func (s SyncSpec) ResetImmutableFields(fieldPrefix string, target *SyncSpec) []string {
	var resetFields []string
	if list := s.ExternalAccess.ResetImmutableFields(fieldPrefix+".externalAccess", &target.ExternalAccess); len(list) > 0 {
		resetFields = append(resetFields, list...)
	}
	if list := s.Authentication.ResetImmutableFields(fieldPrefix+".auth", &target.Authentication); len(list) > 0 {
		resetFields = append(resetFields, list...)
	}
	return resetFields
}
