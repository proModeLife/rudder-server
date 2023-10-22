package eventfilter

import (
	"github.com/samber/lo"
	"strings"

	"golang.org/x/exp/slices"

	"github.com/rudderlabs/rudder-go-kit/logger"
	backendconfig "github.com/rudderlabs/rudder-server/backend-config"
	"github.com/rudderlabs/rudder-server/processor/transformer"
	"github.com/rudderlabs/rudder-server/utils/misc"
	"github.com/rudderlabs/rudder-server/utils/types"
)

const (
	hybridModeEventsFilterKey = "hybridModeCloudEventsFilter"
	hybridMode                = "hybrid"
)

var pkgLogger = logger.NewLogger().Child("eventfilter")

// GetSupportedMessageTypes returns the supported message types for the given event, based on configuration.
// If no relevant configuration is found, returns false
func GetSupportedMessageTypes(destination *backendconfig.DestinationT) ([]string, bool) {
	var supportedMessageTypes []string
	if supportedTypes, ok := destination.DestinationDefinition.Config["supportedMessageTypes"]; ok {
		if supportedTypeInterface, ok := supportedTypes.([]interface{}); ok {
			supportedTypesArr := misc.ConvertInterfaceToStringArray(supportedTypeInterface)
			for _, supportedType := range supportedTypesArr {
				var skip bool
				switch supportedType {
				case "identify":
					skip = identifyDisabled(destination)
				default:
				}
				if !skip {
					supportedMessageTypes = append(supportedMessageTypes, supportedType)
				}
			}
			return supportedMessageTypes, true
		}
	}
	return nil, false
}

func identifyDisabled(destination *backendconfig.DestinationT) bool {
	if serverSideIdentify, flag := destination.Config["enableServerSideIdentify"]; flag {
		if v, ok := serverSideIdentify.(bool); ok {
			return !v
		}
	}
	return false
}

// GetSupportedEvents returns the supported message events for the given destination, based on configuration.
// If no relevant configuration is found, returns false
func GetSupportedMessageEvents(destination *backendconfig.DestinationT) ([]string, bool) {
	if supportedEventsI, ok := destination.Config["listOfConversions"]; ok {
		if supportedEvents, ok := supportedEventsI.([]interface{}); ok {
			var supportedMessageEvents []string
			for _, supportedEvent := range supportedEvents {
				if supportedEventMap, ok := supportedEvent.(map[string]interface{}); ok {
					if conversions, ok := supportedEventMap["conversions"]; ok {
						if supportedMessageEvent, ok := conversions.(string); ok {
							supportedMessageEvents = append(supportedMessageEvents, supportedMessageEvent)
						}
					}
				}
			}
			if len(supportedMessageEvents) != len(supportedEvents) {
				return nil, false
			}
			return supportedMessageEvents, true
		}
	}
	return nil, false
}

type eventParams struct {
	MessageType string
}

type connectionModeFilterParams struct {
	Destination      *backendconfig.DestinationT
	SrcType          string
	Event            *eventParams
	DefaultBehaviour bool
}

func messageType(event *types.SingularEventT) string {
	eventMessageTypeI := misc.MapLookup(*event, "type")
	if eventMessageTypeI == nil {
		pkgLogger.Error("Message type is not being sent for the event")
		return ""
	}
	eventMessageType, ok := eventMessageTypeI.(string)
	if !ok {
		pkgLogger.Errorf("Invalid message type: type assertion failed: %v", eventMessageTypeI)
		return ""
	}
	return eventMessageType
}

/*
AllowEventToDestTransformation lets the caller know if we need to allow the event to proceed to destination transformation.

Currently this method supports below validations(executed in the same order):

1. Validate if messageType sent in event is included in SupportedMessageTypes

2. Validate if the event is sendable to destination based on connectionMode, sourceType & messageType
*/
func AllowEventToDestTransformation(
	transformerEvent *transformer.TransformerEvent,
	supportedMsgTypes []string,
) (
	bool,
	*transformer.TransformerResponse,
) {
	messageType := strings.TrimSpace(strings.ToLower(messageType(&transformerEvent.Message)))
	if messageType == "" {
		return false, &transformer.TransformerResponse{
			Output: transformerEvent.Message, StatusCode: 400,
			Metadata: transformerEvent.Metadata,
			Error:    "Invalid message type. Type assertion failed",
		}
	}

	isSupportedMsgType := slices.Contains(supportedMsgTypes, messageType)
	if !isSupportedMsgType {
		pkgLogger.Debugw("event filtered out due to unsupported msg types",
			"supportedMsgTypes", supportedMsgTypes, "messageType", messageType,
		)
		// We will not allow the event
		return false, &transformer.TransformerResponse{
			Output: transformerEvent.Message, StatusCode: types.FilterEventCode,
			Metadata: transformerEvent.Metadata,
			Error:    "Message type not supported",
		}
	}
	// MessageType filtering -- ENDS

	// hybridModeCloudEventsFilter.srcType.[eventProperty] filtering -- STARTS
	allow := filterEventsForHybridMode(connectionModeFilterParams{
		Destination: &transformerEvent.Destination,
		SrcType:     transformerEvent.Metadata.SourceDefinitionType,
		Event:       &eventParams{MessageType: messageType},
		// Default behavior
		// When something is missing in "supportedConnectionModes" or if "supportedConnectionModes" is not defined
		// We would be checking for below things
		// 1. Check if the event.type value is present in destination.DestinationDefinition.Config["supportedMessageTypes"]
		// 2. Check if the connectionMode of destination is cloud or hybrid(evaluated through `IsProcessorEnabled`)
		// Only when 1 & 2 are true, we would allow the event to flow through to server
		// As when this will be called, we would have already checked if event.type in supportedMessageTypes
		DefaultBehaviour: transformerEvent.Destination.IsProcessorEnabled && isSupportedMsgType,
	})

	if !allow {
		return allow, &transformer.TransformerResponse{
			Output: transformerEvent.Message, StatusCode: types.FilterEventCode,
			Metadata: transformerEvent.Metadata,
			Error:    "Filtering event based on hybridModeFilter",
		}
	}
	// hybridModeCloudEventsFilter.srcType.[eventProperty] filtering -- ENDS

	return true, nil
}

/*
FilterEventsForHybridMode lets the caller know if the event is allowed to flow through server for a `specific destination`
Introduced to support hybrid-mode event filtering on cloud-side

The template inside `destinationDefinition.Config.hybridModeCloudEventsFilter` would look like this
```

	[sourceType]: {
		[eventProperty]: [...supportedEventPropertyValues]
	}

```

Example:

		{
			...
			"hybridModeCloudEventsFilter": {
	      "web": {
	        "messageType": ["track", "page"]
	      }
	    },
			...
		}
*/
func filterEventsForHybridMode(connectionModeFilterParams connectionModeFilterParams) (allow bool) {
	destination := connectionModeFilterParams.Destination
	srcType := strings.TrimSpace(connectionModeFilterParams.SrcType)
	messageType := connectionModeFilterParams.Event.MessageType

	allow = connectionModeFilterParams.DefaultBehaviour

	if srcType == "" {
		return
	}

	destConnModeI := misc.MapLookup(destination.Config, "connectionMode")
	if destConnModeI == nil {
		return
	}

	destConnectionMode, ok := destConnModeI.(string)
	if !ok || destConnectionMode != hybridMode {
		return
	}

	sourceEventPropertiesI := misc.MapLookup(destination.DestinationDefinition.Config, hybridModeEventsFilterKey, srcType)
	if sourceEventPropertiesI == nil {
		return
	}

	eventProperties, ok := sourceEventPropertiesI.(map[string]interface{})
	if !ok || len(eventProperties) == 0 {
		return
	}

	for eventProperty, supportedEventVals := range eventProperties {
		if !allow {
			return connectionModeFilterParams.DefaultBehaviour
		}

		if eventProperty == "messageType" {
			messageTypes := convertToArrayOfType[string](supportedEventVals)
			if len(messageTypes) == 0 {
				return connectionModeFilterParams.DefaultBehaviour
			}

			allow = slices.Contains(messageTypes, messageType) && connectionModeFilterParams.DefaultBehaviour
		}
	}
	return
}

type EventPropsTypes interface {
	~string
}

/*
* Converts interface{} to []T if the go type-assertion allows it
 */
func convertToArrayOfType[T EventPropsTypes](data interface{}) []T {
	switch value := data.(type) {
	case []T:
		return value
	case []interface{}:
		return lo.FilterMap(value, func(item interface{}, index int) (T, bool) {
			val, ok := item.(T)
			return val, ok
		})
	}
	return []T{}
}
