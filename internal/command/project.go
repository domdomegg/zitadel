package command

import (
	"context"
	"strings"

	"github.com/zitadel/logging"
	"github.com/zitadel/zitadel/internal/command/preparation"
	"github.com/zitadel/zitadel/internal/domain"
	"github.com/zitadel/zitadel/internal/errors"
	caos_errs "github.com/zitadel/zitadel/internal/errors"
	"github.com/zitadel/zitadel/internal/eventstore"
	"github.com/zitadel/zitadel/internal/repository/project"
)

func (c *Commands) AddProject(ctx context.Context, project *domain.Project, resourceOwner, ownerUserID string) (_ *domain.Project, err error) {
	events, addedProject, err := c.addProject(ctx, project, resourceOwner, ownerUserID)
	if err != nil {
		return nil, err
	}
	pushedEvents, err := c.eventstore.Push(ctx, events...)
	if err != nil {
		return nil, err
	}
	err = AppendAndReduce(addedProject, pushedEvents...)
	if err != nil {
		return nil, err
	}
	return projectWriteModelToProject(addedProject), nil
}

func (c *Commands) addProject(ctx context.Context, projectAdd *domain.Project, resourceOwner, ownerUserID string) (_ []eventstore.Command, _ *ProjectWriteModel, err error) {
	if !projectAdd.IsValid() {
		return nil, nil, caos_errs.ThrowInvalidArgument(nil, "PROJECT-IOVCC", "Errors.Project.Invalid")
	}
	projectAdd.AggregateID, err = c.idGenerator.Next()
	if err != nil {
		return nil, nil, err
	}
	addedProject := NewProjectWriteModel(projectAdd.AggregateID, resourceOwner)
	projectAgg := ProjectAggregateFromWriteModel(&addedProject.WriteModel)

	projectRole := domain.RoleProjectOwner
	events := []eventstore.Command{
		project.NewProjectAddedEvent(
			ctx,
			projectAgg,
			projectAdd.Name,
			projectAdd.ProjectRoleAssertion,
			projectAdd.ProjectRoleCheck,
			projectAdd.HasProjectCheck,
			projectAdd.PrivateLabelingSetting),
		project.NewProjectMemberAddedEvent(ctx, projectAgg, ownerUserID, projectRole),
	}
	return events, addedProject, nil
}

func AddProjectCommand(
	a *project.Aggregate,
	name string,
	owner string,
	projectRoleAssertion bool,
	projectRoleCheck bool,
	hasProjectCheck bool,
	privateLabelingSetting domain.PrivateLabelingSetting,
) preparation.Validation {
	return func() (preparation.CreateCommands, error) {
		if name = strings.TrimSpace(name); name == "" {
			return nil, errors.ThrowInvalidArgument(nil, "PROJE-C01yo", "Errors.Invalid.Argument")
		}
		if !privateLabelingSetting.Valid() {
			return nil, errors.ThrowInvalidArgument(nil, "PROJE-AO52V", "Errors.Invalid.Argument")
		}
		if owner == "" {
			return nil, errors.ThrowPreconditionFailed(nil, "PROJE-hzxwo", "Errors.Invalid.Argument")
		}
		return func(ctx context.Context, filter preparation.FilterToQueryReducer) ([]eventstore.Command, error) {
			return []eventstore.Command{
				project.NewProjectAddedEvent(ctx, &a.Aggregate,
					name,
					projectRoleAssertion,
					projectRoleCheck,
					hasProjectCheck,
					privateLabelingSetting,
				),
				project.NewProjectMemberAddedEvent(ctx, &a.Aggregate,
					owner,
					domain.RoleProjectOwner),
			}, nil
		}, nil
	}
}

func projectWriteModel(ctx context.Context, filter preparation.FilterToQueryReducer, projectID, resourceOwner string) (project *ProjectWriteModel, err error) {
	project = NewProjectWriteModel(projectID, resourceOwner)
	events, err := filter(ctx, project.Query())
	if err != nil {
		return nil, err
	}

	project.AppendEvents(events...)
	if err := project.Reduce(); err != nil {
		return nil, err
	}

	return project, nil
}

func (c *Commands) getProjectByID(ctx context.Context, projectID, resourceOwner string) (*domain.Project, error) {
	projectWriteModel, err := c.getProjectWriteModelByID(ctx, projectID, resourceOwner)
	if err != nil {
		return nil, err
	}
	if projectWriteModel.State == domain.ProjectStateUnspecified || projectWriteModel.State == domain.ProjectStateRemoved {
		return nil, caos_errs.ThrowNotFound(nil, "PROJECT-Gd2hh", "Errors.Project.NotFound")
	}
	return projectWriteModelToProject(projectWriteModel), nil
}

func (c *Commands) checkProjectExists(ctx context.Context, projectID, resourceOwner string) error {
	projectWriteModel, err := c.getProjectWriteModelByID(ctx, projectID, resourceOwner)
	if err != nil {
		return err
	}
	if projectWriteModel.State == domain.ProjectStateUnspecified || projectWriteModel.State == domain.ProjectStateRemoved {
		return caos_errs.ThrowPreconditionFailed(nil, "COMMAND-4M0fs", "Errors.Project.NotFound")
	}
	return nil
}

func (c *Commands) ChangeProject(ctx context.Context, projectChange *domain.Project, resourceOwner string) (*domain.Project, error) {
	if !projectChange.IsValid() || projectChange.AggregateID == "" {
		return nil, caos_errs.ThrowInvalidArgument(nil, "COMMAND-4m9vS", "Errors.Project.Invalid")
	}

	existingProject, err := c.getProjectWriteModelByID(ctx, projectChange.AggregateID, resourceOwner)
	if err != nil {
		return nil, err
	}
	if existingProject.State == domain.ProjectStateUnspecified || existingProject.State == domain.ProjectStateRemoved {
		return nil, caos_errs.ThrowNotFound(nil, "COMMAND-3M9sd", "Errors.Project.NotFound")
	}

	projectAgg := ProjectAggregateFromWriteModel(&existingProject.WriteModel)
	changedEvent, hasChanged, err := existingProject.NewChangedEvent(
		ctx,
		projectAgg,
		projectChange.Name,
		projectChange.ProjectRoleAssertion,
		projectChange.ProjectRoleCheck,
		projectChange.HasProjectCheck,
		projectChange.PrivateLabelingSetting)
	if err != nil {
		return nil, err
	}
	if !hasChanged {
		return nil, caos_errs.ThrowPreconditionFailed(nil, "COMMAND-2M0fs", "Errors.NoChangesFound")
	}
	pushedEvents, err := c.eventstore.Push(ctx, changedEvent)
	if err != nil {
		return nil, err
	}
	err = AppendAndReduce(existingProject, pushedEvents...)
	if err != nil {
		return nil, err
	}
	return projectWriteModelToProject(existingProject), nil
}

func (c *Commands) DeactivateProject(ctx context.Context, projectID string, resourceOwner string) (*domain.ObjectDetails, error) {
	if projectID == "" || resourceOwner == "" {
		return nil, caos_errs.ThrowInvalidArgument(nil, "COMMAND-88iF0", "Errors.Project.ProjectIDMissing")
	}

	existingProject, err := c.getProjectWriteModelByID(ctx, projectID, resourceOwner)
	if err != nil {
		return nil, err
	}
	if existingProject.State == domain.ProjectStateUnspecified || existingProject.State == domain.ProjectStateRemoved {
		return nil, caos_errs.ThrowNotFound(nil, "COMMAND-112M9", "Errors.Project.NotFound")
	}
	if existingProject.State != domain.ProjectStateActive {
		return nil, caos_errs.ThrowPreconditionFailed(nil, "COMMAND-mki55", "Errors.Project.NotActive")
	}

	projectAgg := ProjectAggregateFromWriteModel(&existingProject.WriteModel)
	pushedEvents, err := c.eventstore.Push(ctx, project.NewProjectDeactivatedEvent(ctx, projectAgg))
	if err != nil {
		return nil, err
	}
	err = AppendAndReduce(existingProject, pushedEvents...)
	if err != nil {
		return nil, err
	}
	return writeModelToObjectDetails(&existingProject.WriteModel), nil
}

func (c *Commands) ReactivateProject(ctx context.Context, projectID string, resourceOwner string) (*domain.ObjectDetails, error) {
	if projectID == "" || resourceOwner == "" {
		return nil, caos_errs.ThrowInvalidArgument(nil, "COMMAND-3ihsF", "Errors.Project.ProjectIDMissing")
	}

	existingProject, err := c.getProjectWriteModelByID(ctx, projectID, resourceOwner)
	if err != nil {
		return nil, err
	}
	if existingProject.State == domain.ProjectStateUnspecified || existingProject.State == domain.ProjectStateRemoved {
		return nil, caos_errs.ThrowNotFound(nil, "COMMAND-3M9sd", "Errors.Project.NotFound")
	}
	if existingProject.State != domain.ProjectStateInactive {
		return nil, caos_errs.ThrowPreconditionFailed(nil, "COMMAND-5M9bs", "Errors.Project.NotInactive")
	}

	projectAgg := ProjectAggregateFromWriteModel(&existingProject.WriteModel)
	pushedEvents, err := c.eventstore.Push(ctx, project.NewProjectReactivatedEvent(ctx, projectAgg))
	if err != nil {
		return nil, err
	}
	err = AppendAndReduce(existingProject, pushedEvents...)
	if err != nil {
		return nil, err
	}
	return writeModelToObjectDetails(&existingProject.WriteModel), nil
}

func (c *Commands) RemoveProject(ctx context.Context, projectID, resourceOwner string, cascadingUserGrantIDs ...string) (*domain.ObjectDetails, error) {
	if projectID == "" || resourceOwner == "" {
		return nil, caos_errs.ThrowInvalidArgument(nil, "COMMAND-66hM9", "Errors.Project.ProjectIDMissing")
	}

	existingProject, err := c.getProjectWriteModelByID(ctx, projectID, resourceOwner)
	if err != nil {
		return nil, err
	}
	if existingProject.State == domain.ProjectStateUnspecified || existingProject.State == domain.ProjectStateRemoved {
		return nil, caos_errs.ThrowNotFound(nil, "COMMAND-3M9sd", "Errors.Project.NotFound")
	}
	projectAgg := ProjectAggregateFromWriteModel(&existingProject.WriteModel)
	events := []eventstore.Command{
		project.NewProjectRemovedEvent(ctx, projectAgg, existingProject.Name),
	}

	for _, grantID := range cascadingUserGrantIDs {
		event, _, err := c.removeUserGrant(ctx, grantID, "", true)
		if err != nil {
			logging.LogWithFields("COMMAND-b8Djf", "usergrantid", grantID).WithError(err).Warn("could not cascade remove user grant")
			continue
		}
		events = append(events, event)
	}

	pushedEvents, err := c.eventstore.Push(ctx, events...)
	if err != nil {
		return nil, err
	}
	err = AppendAndReduce(existingProject, pushedEvents...)
	if err != nil {
		return nil, err
	}
	return writeModelToObjectDetails(&existingProject.WriteModel), nil
}

func (c *Commands) getProjectWriteModelByID(ctx context.Context, projectID, resourceOwner string) (*ProjectWriteModel, error) {
	projectWriteModel := NewProjectWriteModel(projectID, resourceOwner)
	err := c.eventstore.FilterToQueryReducer(ctx, projectWriteModel)
	if err != nil {
		return nil, err
	}
	return projectWriteModel, nil
}
