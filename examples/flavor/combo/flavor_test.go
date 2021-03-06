package main

import (
	"errors"
	"testing"

	mock_flavor "github.com/docker/infrakit/pkg/mock/spi/flavor"
	"github.com/docker/infrakit/pkg/plugin"
	"github.com/docker/infrakit/pkg/plugin/group"
	group_types "github.com/docker/infrakit/pkg/plugin/group/types"
	"github.com/docker/infrakit/pkg/spi/flavor"
	"github.com/docker/infrakit/pkg/spi/instance"
	"github.com/docker/infrakit/pkg/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func logicalID(v string) *instance.LogicalID {
	id := instance.LogicalID(v)
	return &id
}

var inst = instance.Spec{
	Properties:  types.AnyString("{}"),
	Tags:        map[string]string{},
	Init:        "",
	LogicalID:   logicalID("id"),
	Attachments: []instance.Attachment{{ID: "att1", Type: "nic"}},
}

func pluginLookup(plugins map[string]flavor.Plugin) group.FlavorPluginLookup {
	return func(key plugin.Name) (flavor.Plugin, error) {
		plugin, has := plugins[key.String()]
		if has {
			return plugin, nil
		}
		return nil, errors.New("Plugin doesn't exist")
	}
}

func TestMergeBehavior(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a := mock_flavor.NewMockPlugin(ctrl)
	b := mock_flavor.NewMockPlugin(ctrl)

	plugins := map[string]flavor.Plugin{"a": a, "b": b}

	combo := NewPlugin(pluginLookup(plugins))

	flavorProperties := types.AnyString(`{
	  "Flavors": [
	    {
	      "Plugin": "a",
	      "Properties": {"a": "1"}
	    },
	    {
	      "Plugin": "b",
	      "Properties": {"b": "2"}
	    }
	  ]
	}`)

	allocation := group_types.AllocationMethod{Size: 1}

	a.EXPECT().Prepare(types.AnyString(`{"a": "1"}`), inst, allocation).Return(instance.Spec{
		Properties:  inst.Properties,
		Tags:        map[string]string{"a": "1", "c": "4"},
		Init:        "init data a",
		LogicalID:   inst.LogicalID,
		Attachments: []instance.Attachment{{ID: "a", Type: "nic"}},
	}, nil)

	b.EXPECT().Prepare(types.AnyString(`{"b": "2"}`), inst, allocation).Return(instance.Spec{
		Properties:  inst.Properties,
		Tags:        map[string]string{"b": "2", "c": "5"},
		Init:        "init data b",
		LogicalID:   inst.LogicalID,
		Attachments: []instance.Attachment{{ID: "b", Type: "gpu"}},
	}, nil)

	result, err := combo.Prepare(flavorProperties, inst, group_types.AllocationMethod{Size: 1})
	require.NoError(t, err)

	expected := instance.Spec{
		Properties:  inst.Properties,
		Tags:        map[string]string{"a": "1", "b": "2", "c": "5"},
		Init:        "init data a\ninit data b",
		LogicalID:   inst.LogicalID,
		Attachments: []instance.Attachment{{ID: "att1", Type: "nic"}, {ID: "a", Type: "nic"}, {ID: "b", Type: "gpu"}},
	}
	require.Equal(t, expected, result)
}

func TestMergeNoLogicalID(t *testing.T) {
	// Tests regression of a bug where a zero value was returned for the LogicalID despite nil being passed.

	inst = instance.Spec{
		Properties:  types.AnyString("{}"),
		Tags:        map[string]string{},
		Init:        "",
		Attachments: []instance.Attachment{{ID: "att1", Type: "nic"}},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a := mock_flavor.NewMockPlugin(ctrl)
	b := mock_flavor.NewMockPlugin(ctrl)

	plugins := map[string]flavor.Plugin{"a": a, "b": b}

	combo := NewPlugin(pluginLookup(plugins))

	flavorProperties := types.AnyString(`{
	  "Flavors": [
	    {
	      "Plugin": "a",
	      "Properties": {"a": "1"}
	    },
	    {
	      "Plugin": "b",
	      "Properties": {"b": "2"}
	    }
	  ]
	}`)

	allocation := group_types.AllocationMethod{Size: 1}

	a.EXPECT().Prepare(types.AnyString(`{"a": "1"}`), inst, allocation).Return(instance.Spec{
		Properties:  inst.Properties,
		Tags:        map[string]string{"a": "1", "c": "4"},
		Init:        "init data a",
		LogicalID:   inst.LogicalID,
		Attachments: []instance.Attachment{{ID: "a", Type: "nic"}},
	}, nil)

	b.EXPECT().Prepare(types.AnyString(`{"b": "2"}`), inst, allocation).Return(instance.Spec{
		Properties:  inst.Properties,
		Tags:        map[string]string{"b": "2", "c": "5"},
		Init:        "init data b",
		LogicalID:   inst.LogicalID,
		Attachments: []instance.Attachment{{ID: "b", Type: "gpu"}},
	}, nil)

	result, err := combo.Prepare(flavorProperties, inst, group_types.AllocationMethod{Size: 1})
	require.NoError(t, err)

	expected := instance.Spec{
		Properties:  inst.Properties,
		Tags:        map[string]string{"a": "1", "b": "2", "c": "5"},
		Init:        "init data a\ninit data b",
		LogicalID:   inst.LogicalID,
		Attachments: []instance.Attachment{{ID: "att1", Type: "nic"}, {ID: "a", Type: "nic"}, {ID: "b", Type: "gpu"}},
	}
	require.Equal(t, expected, result)
}
