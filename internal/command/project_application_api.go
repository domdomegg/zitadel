package command

import (
	"context"
	"strings"

	"github.com/zitadel/logging"

	"github.com/zitadel/zitadel/internal/command/preparation"
	"github.com/zitadel/zitadel/internal/crypto"
	"github.com/zitadel/zitadel/internal/domain"
	"github.com/zitadel/zitadel/internal/errors"
	"github.com/zitadel/zitadel/internal/eventstore"
	project_repo "github.com/zitadel/zitadel/internal/repository/project"
	"github.com/zitadel/zitadel/internal/telemetry/tracing"
)

type addAPIApp struct {
	AddApp
	AuthMethodType domain.APIAuthMethodType

	ClientID          string
	ClientSecret      *crypto.CryptoValue
	ClientSecretPlain string
}

func (c *Commands) AddAPIAppCommand(app *addAPIApp, clientSecretAlg crypto.HashAlgorithm) preparation.Validation {
	return func() (preparation.CreateCommands, error) {
		if app.ID == "" {
			return nil, errors.ThrowInvalidArgument(nil, "PROJE-XHsKt", "Errors.Invalid.Argument")
		}
		if app.Name = strings.TrimSpace(app.Name); app.Name == "" {
			return nil, errors.ThrowInvalidArgument(nil, "PROJE-F7g21", "Errors.Invalid.Argument")
		}
		return func(ctx context.Context, filter preparation.FilterToQueryReducer) ([]eventstore.Command, error) {
			project, err := projectWriteModel(ctx, filter, app.Aggregate.ID, app.Aggregate.ResourceOwner)
			if err != nil || !project.State.Valid() {
				return nil, errors.ThrowNotFound(err, "PROJE-Sf2gb", "Errors.Project.NotFound")
			}

			app.ClientID, err = domain.NewClientID(c.idGenerator, project.Name)
			if err != nil {
				return nil, errors.ThrowInternal(err, "V2-f0pgP", "Errors.Internal")
			}

			if app.AuthMethodType == domain.APIAuthMethodTypeBasic {
				app.ClientSecret, app.ClientSecretPlain, err = newAppClientSecret(ctx, filter, clientSecretAlg)
				if err != nil {
					return nil, err
				}
			}

			return []eventstore.Command{
				project_repo.NewApplicationAddedEvent(
					ctx,
					&app.Aggregate.Aggregate,
					app.ID,
					app.Name,
				),
				project_repo.NewAPIConfigAddedEvent(
					ctx,
					&app.Aggregate.Aggregate,
					app.ID,
					app.ClientID,
					app.ClientSecret,
					app.AuthMethodType,
				),
			}, nil
		}, nil
	}
}

func (c *Commands) AddAPIApplication(ctx context.Context, application *domain.APIApp, resourceOwner string, appSecretGenerator crypto.Generator) (_ *domain.APIApp, err error) {
	if application == nil || application.AggregateID == "" {
		return nil, errors.ThrowInvalidArgument(nil, "PROJECT-5m9E", "Errors.Application.Invalid")
	}
	project, err := c.getProjectByID(ctx, application.AggregateID, resourceOwner)
	if err != nil {
		return nil, errors.ThrowPreconditionFailed(err, "PROJECT-9fnsf", "Errors.Project.NotFound")
	}
	addedApplication := NewAPIApplicationWriteModel(application.AggregateID, resourceOwner)
	projectAgg := ProjectAggregateFromWriteModel(&addedApplication.WriteModel)
	events, stringPw, err := c.addAPIApplication(ctx, projectAgg, project, application, resourceOwner, appSecretGenerator)
	if err != nil {
		return nil, err
	}
	addedApplication.AppID = application.AppID
	pushedEvents, err := c.eventstore.Push(ctx, events...)
	if err != nil {
		return nil, err
	}
	err = AppendAndReduce(addedApplication, pushedEvents...)
	if err != nil {
		return nil, err
	}
	result := apiWriteModelToAPIConfig(addedApplication)
	result.ClientSecretString = stringPw
	return result, nil
}

func (c *Commands) addAPIApplication(ctx context.Context, projectAgg *eventstore.Aggregate, proj *domain.Project, apiAppApp *domain.APIApp, resourceOwner string, appSecretGenerator crypto.Generator) (events []eventstore.Command, stringPW string, err error) {
	if !apiAppApp.IsValid() {
		return nil, "", errors.ThrowInvalidArgument(nil, "PROJECT-Bff2g", "Errors.Application.Invalid")
	}
	apiAppApp.AppID, err = c.idGenerator.Next()
	if err != nil {
		return nil, "", err
	}

	events = []eventstore.Command{
		project_repo.NewApplicationAddedEvent(ctx, projectAgg, apiAppApp.AppID, apiAppApp.AppName),
	}

	var stringPw string
	err = domain.SetNewClientID(apiAppApp, c.idGenerator, proj)
	if err != nil {
		return nil, "", err
	}
	stringPw, err = domain.SetNewClientSecretIfNeeded(apiAppApp, appSecretGenerator)
	if err != nil {
		return nil, "", err
	}
	events = append(events, project_repo.NewAPIConfigAddedEvent(ctx,
		projectAgg,
		apiAppApp.AppID,
		apiAppApp.ClientID,
		apiAppApp.ClientSecret,
		apiAppApp.AuthMethodType))

	return events, stringPw, nil
}

func (c *Commands) ChangeAPIApplication(ctx context.Context, apiApp *domain.APIApp, resourceOwner string) (*domain.APIApp, error) {
	if apiApp.AppID == "" || apiApp.AggregateID == "" {
		return nil, errors.ThrowInvalidArgument(nil, "COMMAND-1m900", "Errors.Project.App.APIConfigInvalid")
	}

	existingAPI, err := c.getAPIAppWriteModel(ctx, apiApp.AggregateID, apiApp.AppID, resourceOwner)
	if err != nil {
		return nil, err
	}
	if existingAPI.State == domain.AppStateUnspecified || existingAPI.State == domain.AppStateRemoved {
		return nil, errors.ThrowNotFound(nil, "COMMAND-2n8uU", "Errors.Project.App.NotExisting")
	}
	if !existingAPI.IsAPI() {
		return nil, errors.ThrowInvalidArgument(nil, "COMMAND-Gnwt3", "Errors.Project.App.IsNotAPI")
	}
	projectAgg := ProjectAggregateFromWriteModel(&existingAPI.WriteModel)
	changedEvent, hasChanged, err := existingAPI.NewChangedEvent(
		ctx,
		projectAgg,
		apiApp.AppID,
		apiApp.AuthMethodType)
	if err != nil {
		return nil, err
	}
	if !hasChanged {
		return nil, errors.ThrowPreconditionFailed(nil, "COMMAND-1m88i", "Errors.NoChangesFound")
	}

	pushedEvents, err := c.eventstore.Push(ctx, changedEvent)
	if err != nil {
		return nil, err
	}
	err = AppendAndReduce(existingAPI, pushedEvents...)
	if err != nil {
		return nil, err
	}

	return apiWriteModelToAPIConfig(existingAPI), nil
}

func (c *Commands) ChangeAPIApplicationSecret(ctx context.Context, projectID, appID, resourceOwner string, appSecretGenerator crypto.Generator) (*domain.APIApp, error) {
	if projectID == "" || appID == "" {
		return nil, errors.ThrowInvalidArgument(nil, "COMMAND-99i83", "Errors.IDMissing")
	}

	existingAPI, err := c.getAPIAppWriteModel(ctx, projectID, appID, resourceOwner)
	if err != nil {
		return nil, err
	}
	if existingAPI.State == domain.AppStateUnspecified || existingAPI.State == domain.AppStateRemoved {
		return nil, errors.ThrowNotFound(nil, "COMMAND-2g66f", "Errors.Project.App.NotExisting")
	}
	if !existingAPI.IsAPI() {
		return nil, errors.ThrowInvalidArgument(nil, "COMMAND-aeH4", "Errors.Project.App.IsNotAPI")
	}
	cryptoSecret, stringPW, err := domain.NewClientSecret(appSecretGenerator)
	if err != nil {
		return nil, err
	}

	projectAgg := ProjectAggregateFromWriteModel(&existingAPI.WriteModel)

	pushedEvents, err := c.eventstore.Push(ctx, project_repo.NewAPIConfigSecretChangedEvent(ctx, projectAgg, appID, cryptoSecret))
	if err != nil {
		return nil, err
	}
	err = AppendAndReduce(existingAPI, pushedEvents...)
	if err != nil {
		return nil, err
	}

	result := apiWriteModelToAPIConfig(existingAPI)
	result.ClientSecretString = stringPW
	return result, err
}

func (c *Commands) VerifyAPIClientSecret(ctx context.Context, projectID, appID, secret string) (err error) {
	ctx, span := tracing.NewSpan(ctx)
	defer func() { span.EndWithError(err) }()

	app, err := c.getAPIAppWriteModel(ctx, projectID, appID, "")
	if err != nil {
		return err
	}
	if !app.State.Exists() {
		return errors.ThrowPreconditionFailed(nil, "COMMAND-DFnbf", "Errors.Project.App.NoExisting")
	}
	if !app.IsAPI() {
		return errors.ThrowInvalidArgument(nil, "COMMAND-Bf3fw", "Errors.Project.App.IsNotAPI")
	}
	if app.ClientSecret == nil {
		return errors.ThrowPreconditionFailed(nil, "COMMAND-D3t5g", "Errors.Project.App.APIConfigInvalid")
	}

	projectAgg := ProjectAggregateFromWriteModel(&app.WriteModel)
	ctx, spanPasswordComparison := tracing.NewNamedSpan(ctx, "crypto.CompareHash")
	err = crypto.CompareHash(app.ClientSecret, []byte(secret), c.userPasswordAlg)
	spanPasswordComparison.EndWithError(err)
	if err == nil {
		_, err = c.eventstore.Push(ctx, project_repo.NewAPIConfigSecretCheckSucceededEvent(ctx, projectAgg, app.AppID))
		return err
	}
	_, err = c.eventstore.Push(ctx, project_repo.NewAPIConfigSecretCheckFailedEvent(ctx, projectAgg, app.AppID))
	logging.Log("COMMAND-g3f12").OnError(err).Error("could not push event APIClientSecretCheckFailed")
	return errors.ThrowInvalidArgument(nil, "COMMAND-SADfg", "Errors.Project.App.ClientSecretInvalid")
}

func (c *Commands) getAPIAppWriteModel(ctx context.Context, projectID, appID, resourceOwner string) (*APIApplicationWriteModel, error) {
	appWriteModel := NewAPIApplicationWriteModelWithAppID(projectID, appID, resourceOwner)
	err := c.eventstore.FilterToQueryReducer(ctx, appWriteModel)
	if err != nil {
		return nil, err
	}
	return appWriteModel, nil
}
