package instance

import (
	"context"
	"encoding/json"

	"github.com/zitadel/zitadel/internal/eventstore"

	"github.com/zitadel/zitadel/internal/errors"
	"github.com/zitadel/zitadel/internal/eventstore/repository"
)

const (
	GlobalOrgSetEventType eventstore.EventType = "iam.global.org.set"
)

type GlobalOrgSetEvent struct {
	eventstore.BaseEvent `json:"-"`

	OrgID string `json:"globalOrgId"`
}

func (e *GlobalOrgSetEvent) Data() interface{} {
	return e
}

func (e *GlobalOrgSetEvent) UniqueConstraints() []*eventstore.EventUniqueConstraint {
	return nil
}

func NewGlobalOrgSetEventEvent(
	ctx context.Context,
	aggregate *eventstore.Aggregate,
	orgID string,
) *GlobalOrgSetEvent {
	return &GlobalOrgSetEvent{
		BaseEvent: *eventstore.NewBaseEventForPush(
			ctx,
			aggregate,
			GlobalOrgSetEventType,
		),
		OrgID: orgID,
	}
}

func GlobalOrgSetMapper(event *repository.Event) (eventstore.Event, error) {
	e := &GlobalOrgSetEvent{
		BaseEvent: *eventstore.BaseEventFromRepo(event),
	}
	err := json.Unmarshal(event.Data, e)
	if err != nil {
		return nil, errors.ThrowInternal(err, "IAM-cdFZH", "unable to unmarshal global org set")
	}

	return e, nil
}
