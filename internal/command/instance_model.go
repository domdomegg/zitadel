package command

import (
	"golang.org/x/text/language"

	"github.com/zitadel/zitadel/internal/domain"
	"github.com/zitadel/zitadel/internal/eventstore"
	"github.com/zitadel/zitadel/internal/repository/instance"
)

type InstanceWriteModel struct {
	eventstore.WriteModel

	Name            string
	State           domain.InstanceState
	GeneratedDomain string

	GlobalOrgID     string
	ProjectID       string
	DefaultLanguage language.Tag
}

func NewInstanceWriteModel(instanceID string) *InstanceWriteModel {
	return &InstanceWriteModel{
		WriteModel: eventstore.WriteModel{
			AggregateID:   instanceID,
			ResourceOwner: instanceID,
		},
	}
}

func (wm *InstanceWriteModel) Reduce() error {
	for _, event := range wm.Events {
		switch e := event.(type) {
		case *instance.InstanceAddedEvent:
			wm.Name = e.Name
			wm.State = domain.InstanceStateActive
		case *instance.InstanceChangedEvent:
			wm.Name = e.Name
		case *instance.InstanceRemovedEvent:
			wm.State = domain.InstanceStateRemoved
		case *instance.DomainAddedEvent:
			if !e.Generated {
				continue
			}
			wm.GeneratedDomain = e.Domain
		case *instance.ProjectSetEvent:
			wm.ProjectID = e.ProjectID
		case *instance.GlobalOrgSetEvent:
			wm.GlobalOrgID = e.OrgID
		case *instance.DefaultLanguageSetEvent:
			wm.DefaultLanguage = e.Language
		}
	}
	return nil
}

func (wm *InstanceWriteModel) Query() *eventstore.SearchQueryBuilder {
	return eventstore.NewSearchQueryBuilder(eventstore.ColumnsEvent).
		ResourceOwner(wm.ResourceOwner).
		AddQuery().
		AggregateTypes(instance.AggregateType).
		AggregateIDs(wm.AggregateID).
		EventTypes(
			instance.InstanceAddedEventType,
			instance.InstanceChangedEventType,
			instance.InstanceRemovedEventType,
			instance.InstanceDomainAddedEventType,
			instance.InstanceDomainRemovedEventType,
			instance.ProjectSetEventType,
			instance.GlobalOrgSetEventType,
			instance.DefaultLanguageSetEventType).
		Builder()
}

func InstanceAggregateFromWriteModel(wm *eventstore.WriteModel) *eventstore.Aggregate {
	return eventstore.AggregateFromWriteModel(wm, instance.AggregateType, instance.AggregateVersion)
}
