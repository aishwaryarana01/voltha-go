/*
 * Copyright 2019-present Open Networking Foundation

 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at

 * http://www.apache.org/licenses/LICENSE-2.0

 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package core

import (
	"context"
	"encoding/json"
	"github.com/opencord/voltha-go/common/log"
	"github.com/opencord/voltha-go/common/version"
	"github.com/opencord/voltha-go/db/model"
	"github.com/opencord/voltha-protos/go/voltha"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Enumerated type to keep track of miscellaneous data path agents
type DataModelType int

// Enumerated list of data path agents
const (
	Adapters DataModelType = 1 + iota
	AlarmFilters
	CoreInstances
	DeviceTypes
	DeviceGroups
	Voltha
)

const SENTINEL_ADAPTER_ID = "adapter_sentinel"

// String equivalent for data path agents
var commonTypes = []string{
	"Adapters",
	"AlarmFilters",
	"CoreInstances",
	"DeviceTypes",
	"DeviceGroups",
	"Voltha",
}

// String converts the enumerated data path agent value to its string equivalent
func (t DataModelType) String() string {
	return commonTypes[t-1]
}

const MultipleValuesMsg = "Expected a single value for KV query for an instance (%s) of type '%s', but received multiple values"

// ModelProxyManager controls requests made to the miscellaneous data path agents
type ModelProxyManager struct {
	modelProxy       map[string]*ModelProxy
	clusterDataProxy *model.Proxy
}

func newModelProxyManager(cdProxy *model.Proxy) *ModelProxyManager {
	var mgr ModelProxyManager
	mgr.modelProxy = make(map[string]*ModelProxy)
	mgr.clusterDataProxy = cdProxy
	return &mgr
}

// GetDeviceType returns the device type associated to the provided id
func (mpMgr *ModelProxyManager) GetVoltha(ctx context.Context) (*voltha.Voltha, error) {
	log.Debug("GetVoltha")

	/*
	 * For now, encode all the version information into a JSON object and
	 * pass that back as "version" so the client can get all the
	 * information associated with the version. Long term the API should
	 * better accomidate this, but for now this will work.
	 */
	data, err := json.Marshal(&version.VersionInfo)
	info := version.VersionInfo.Version
	if err != nil {
		log.Warnf("Unable to encode version information as JSON: %s", err.Error())
	} else {
		info = string(data)
	}

	return &voltha.Voltha{
		Version: info,
	}, nil
}

// ListCoreInstances returns all the core instances known to the system
func (mpMgr *ModelProxyManager) ListCoreInstances(ctx context.Context) (*voltha.CoreInstances, error) {
	log.Debug("ListCoreInstances")

	// TODO: Need to retrieve the list of registered cores

	return &voltha.CoreInstances{}, status.Errorf(codes.NotFound, "no-core-instances")
}

// GetCoreInstance returns the core instance associated to the provided id
func (mpMgr *ModelProxyManager) GetCoreInstance(ctx context.Context, id string) (*voltha.CoreInstance, error) {
	log.Debugw("GetCoreInstance", log.Fields{"id": id})

	// TODO: Need to retrieve the list of registered cores

	return &voltha.CoreInstance{}, status.Errorf(codes.NotFound, "core-instance-%s", id)
}

// ListAdapters returns all the device types known to the system
func (mpMgr *ModelProxyManager) ListAdapters(ctx context.Context) (*voltha.Adapters, error) {
	log.Debug("ListAdapters")

	var agent *ModelProxy
	var exists bool
	var adapter *voltha.Adapter

	if agent, exists = mpMgr.modelProxy[Adapters.String()]; !exists {
		agent = newModelProxy("adapters", mpMgr.clusterDataProxy)
		mpMgr.modelProxy[Adapters.String()] = agent
	}

	adapters := &voltha.Adapters{}
	if items, err := agent.Get(); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	} else if items != nil {
		list, ok := items.([]interface{})
		if !ok {
			list = []interface{}{items}
		}
		for _, item := range list {
			adapter = item.(*voltha.Adapter)
			if adapter.Id != SENTINEL_ADAPTER_ID { // don't report the sentinel
				adapters.Items = append(adapters.Items, adapter)
			}
		}
		log.Debugw("retrieved-adapters", log.Fields{"adapters": adapters})
		return adapters, nil
	}

	return adapters, status.Errorf(codes.NotFound, "no-adapters")
}

// ListDeviceTypes returns all the device types known to the system
func (mpMgr *ModelProxyManager) ListDeviceTypes(ctx context.Context) (*voltha.DeviceTypes, error) {
	log.Debug("ListDeviceTypes")

	var agent *ModelProxy
	var exists bool

	if agent, exists = mpMgr.modelProxy[DeviceTypes.String()]; !exists {
		agent = newModelProxy("device_types", mpMgr.clusterDataProxy)
		mpMgr.modelProxy[DeviceTypes.String()] = agent
	}

	deviceTypes := &voltha.DeviceTypes{}
	if items, err := agent.Get(); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	} else if items != nil {
		list, ok := items.([]interface{})
		if !ok {
			list = []interface{}{items}
		}
		for _, item := range list {
			deviceTypes.Items = append(deviceTypes.Items, item.(*voltha.DeviceType))
		}
		return deviceTypes, nil
	}

	return deviceTypes, status.Errorf(codes.NotFound, "no-device-types")
}

// GetDeviceType returns the device type associated to the provided id
func (mpMgr *ModelProxyManager) GetDeviceType(ctx context.Context, id string) (*voltha.DeviceType, error) {
	log.Debugw("GetDeviceType", log.Fields{"id": id})

	var agent *ModelProxy
	var exists bool

	if agent, exists = mpMgr.modelProxy[DeviceTypes.String()]; !exists {
		agent = newModelProxy("device_types", mpMgr.clusterDataProxy)
		mpMgr.modelProxy[DeviceTypes.String()] = agent
	}

	if deviceType, err := agent.Get(id); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	} else if deviceType != nil {
		_, ok := deviceType.(*voltha.DeviceType)
		if !ok {
			return nil, status.Errorf(codes.Internal, MultipleValuesMsg,
				id, DeviceTypes.String())
		}
		return deviceType.(*voltha.DeviceType), nil
	}

	return &voltha.DeviceType{}, status.Errorf(codes.NotFound, "device-type-%s", id)
}

// ListDeviceGroups returns all the device groups known to the system
func (mpMgr *ModelProxyManager) ListDeviceGroups(ctx context.Context) (*voltha.DeviceGroups, error) {
	log.Debug("ListDeviceGroups")

	var agent *ModelProxy
	var exists bool

	if agent, exists = mpMgr.modelProxy[DeviceGroups.String()]; !exists {
		agent = newModelProxy("device_groups", mpMgr.clusterDataProxy)
		mpMgr.modelProxy[DeviceGroups.String()] = agent
	}

	deviceGroups := &voltha.DeviceGroups{}
	if items, err := agent.Get(); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	} else if items != nil {
		list, ok := items.([]interface{})
		if !ok {
			list = []interface{}{items}
		}
		for _, item := range list {
			deviceGroups.Items = append(deviceGroups.Items, item.(*voltha.DeviceGroup))
		}
		return deviceGroups, nil
	}

	return deviceGroups, status.Errorf(codes.NotFound, "no-device-groups")
}

// GetDeviceGroup returns the device group associated to the provided id
func (mpMgr *ModelProxyManager) GetDeviceGroup(ctx context.Context, id string) (*voltha.DeviceGroup, error) {
	log.Debugw("GetDeviceGroup", log.Fields{"id": id})

	var agent *ModelProxy
	var exists bool

	if agent, exists = mpMgr.modelProxy[DeviceGroups.String()]; !exists {
		agent = newModelProxy("device_groups", mpMgr.clusterDataProxy)
		mpMgr.modelProxy[DeviceGroups.String()] = agent
	}

	if deviceGroup, err := agent.Get(id); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	} else if deviceGroup != nil {
		_, ok := deviceGroup.(*voltha.DeviceGroup)
		if !ok {
			return nil, status.Errorf(codes.Internal, MultipleValuesMsg,
				id, DeviceGroups.String())
		}
		return deviceGroup.(*voltha.DeviceGroup), nil
	}

	return &voltha.DeviceGroup{}, status.Errorf(codes.NotFound, "device-group-%s", id)
}

// ListAlarmFilters returns all the alarm filters known to the system
func (mpMgr *ModelProxyManager) ListAlarmFilters(ctx context.Context) (*voltha.AlarmFilters, error) {
	log.Debug("ListAlarmFilters")

	var agent *ModelProxy
	var exists bool

	if agent, exists = mpMgr.modelProxy[AlarmFilters.String()]; !exists {
		agent = newModelProxy("alarm_filters", mpMgr.clusterDataProxy)
		mpMgr.modelProxy[AlarmFilters.String()] = agent
	}

	alarmFilters := &voltha.AlarmFilters{}
	if items, err := agent.Get(); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	} else if items != nil {
		list, ok := items.([]interface{})
		if !ok {
			list = []interface{}{items}
		}
		for _, item := range list {
			alarmFilters.Filters = append(alarmFilters.Filters, item.(*voltha.AlarmFilter))
		}
		return alarmFilters, nil
	}

	return alarmFilters, status.Errorf(codes.NotFound, "no-alarm-filters")
}

// GetAlarmFilter returns the alarm filter associated to the provided id
func (mpMgr *ModelProxyManager) GetAlarmFilter(ctx context.Context, id string) (*voltha.AlarmFilter, error) {
	log.Debugw("GetAlarmFilter", log.Fields{"id": id})

	var agent *ModelProxy
	var exists bool

	if agent, exists = mpMgr.modelProxy[AlarmFilters.String()]; !exists {
		agent = newModelProxy("alarm_filters", mpMgr.clusterDataProxy)
		mpMgr.modelProxy[AlarmFilters.String()] = agent
	}

	if alarmFilter, err := agent.Get(id); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	} else if alarmFilter != nil {
		_, ok := alarmFilter.(*voltha.AlarmFilter)
		if !ok {
			return nil, status.Errorf(codes.Internal, MultipleValuesMsg,
				id, AlarmFilters.String())
		}
		return alarmFilter.(*voltha.AlarmFilter), nil
	}
	return &voltha.AlarmFilter{}, status.Errorf(codes.NotFound, "alarm-filter-%s", id)
}
