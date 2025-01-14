package command

import (
	"context"

	"github.com/zitadel/zitadel/internal/eventstore"
	"github.com/zitadel/zitadel/internal/query"
	"github.com/zitadel/zitadel/internal/repository/instance"
	"github.com/zitadel/zitadel/internal/repository/org"
	"github.com/zitadel/zitadel/internal/repository/project"
)

func (c *Commands) removeUserMemberships(ctx context.Context, memberships []*query.Membership) (_ []eventstore.Command, err error) {
	events := make([]eventstore.Command, 0)
	for _, membership := range memberships {
		if membership.IAM != nil {
			iamAgg := instance.NewAggregate(membership.IAM.IAMID)
			removeEvent := c.removeInstanceMember(ctx, &iamAgg.Aggregate, membership.UserID, true)
			events = append(events, removeEvent)
		} else if membership.Org != nil {
			orgAgg := org.NewAggregate(membership.Org.OrgID)
			removeEvent := c.removeOrgMember(ctx, &orgAgg.Aggregate, membership.UserID, true)
			events = append(events, removeEvent)
		} else if membership.Project != nil {
			projectAgg := project.NewAggregate(membership.Project.ProjectID, membership.ResourceOwner)
			removeEvent := c.removeProjectMember(ctx, &projectAgg.Aggregate, membership.UserID, true)
			events = append(events, removeEvent)
		} else if membership.ProjectGrant != nil {
			projectAgg := project.NewAggregate(membership.ProjectGrant.ProjectID, membership.ResourceOwner)
			removeEvent := c.removeProjectGrantMember(ctx, &projectAgg.Aggregate, membership.UserID, membership.ProjectGrant.GrantID, true)
			events = append(events, removeEvent)
		}
	}
	return events, nil
}
