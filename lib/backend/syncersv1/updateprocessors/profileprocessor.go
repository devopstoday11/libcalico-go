// Copyright (c) 2017 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package updateprocessors

import (
	"errors"
	"fmt"

	apiv2 "github.com/projectcalico/libcalico-go/lib/apis/v2"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/watchersyncer"
	log "github.com/sirupsen/logrus"
)

// Create a new SyncerUpdateProcessor to sync Profile data in v1 format for
// consumption by Felix.
func NewProfileUpdateProcessor() watchersyncer.SyncerUpdateProcessor {
	return &profileUpdateProcessor{
		v2Kind: apiv2.KindProfile,
	}
}

// Need to create custom logic for Profile since it breaks the values into 3 separate KV Pairs:
// Tags, Labels, and Rules.
type profileUpdateProcessor struct {
	v2Kind string
}

func (pup *profileUpdateProcessor) Process(kvp *model.KVPair) ([]*model.KVPair, error) {
	// Check the v2 resource is the correct type.
	rk, ok := kvp.Key.(model.ResourceKey)
	if !ok || rk.Kind != pup.v2Kind {
		return nil, fmt.Errorf("Incorrect key type - expecting resource of kind %s", pup.v2Kind)
	}

	// Convert the v2 resource to the equivalent v1 resource type.
	v2key, ok := kvp.Key.(model.ResourceKey)
	if !ok {
		return nil, errors.New("Key is not a valid V2 resource key")
	}

	if v2key.Name == "" {
		return nil, errors.New("Missing Name field to create a v1 Profile Key")
	}

	pk := model.ProfileKey{
		Name: v2key.Name,
	}

	v1labelsKey := model.ProfileLabelsKey{pk}
	v1rulesKey := model.ProfileRulesKey{pk}

	var v1profile *model.Profile
	var err error
	// Deletion events will have a value of nil. Do not convert anything for a deletion event.
	if kvp.Value != nil {
		v1profile, err = convertProfileV2ToV1Value(kvp.Value)
		if err != nil {
			// Currently treat any errors as a deletion event.
			log.WithField("Resource", kvp.Key).Warn("Unable to process resource data - treating as deleted")
		}
	}

	labelskvp := &model.KVPair{
		Key: v1labelsKey,
	}
	ruleskvp := &model.KVPair{
		Key: v1rulesKey,
	}

	if v1profile != nil {
		labelskvp.Value = v1profile.Labels
		labelskvp.Revision = kvp.Revision
		ruleskvp.Value = &v1profile.Rules
		ruleskvp.Revision = kvp.Revision
	}

	return []*model.KVPair{labelskvp, ruleskvp}, nil
}

func (pup *profileUpdateProcessor) OnSyncerStarting() {
	// Do nothing
}

func convertProfileV2ToV1Value(val interface{}) (*model.Profile, error) {
	v2res, ok := val.(*apiv2.Profile)
	if !ok {
		return nil, errors.New("Value is not a valid Profile resource value")
	}

	var irules []model.Rule
	for _, irule := range v2res.Spec.IngressRules {
		irules = append(irules, RuleAPIV2ToBackend(irule))
	}

	var erules []model.Rule
	for _, erule := range v2res.Spec.EgressRules {
		erules = append(erules, RuleAPIV2ToBackend(erule))
	}

	rules := model.ProfileRules{
		InboundRules:  irules,
		OutboundRules: erules,
	}

	v1value := &model.Profile{
		Rules:  rules,
		Labels: v2res.Spec.LabelsToApply,
	}

	return v1value, nil
}
