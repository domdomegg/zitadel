package projection

import (
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/zitadel/zitadel/internal/domain"
	"github.com/zitadel/zitadel/internal/errors"
	"github.com/zitadel/zitadel/internal/eventstore"
	"github.com/zitadel/zitadel/internal/eventstore/handler"
	"github.com/zitadel/zitadel/internal/eventstore/repository"
	"github.com/zitadel/zitadel/internal/repository/project"
)

func TestAppProjection_reduces(t *testing.T) {
	type args struct {
		event func(t *testing.T) eventstore.Event
	}
	tests := []struct {
		name   string
		args   args
		reduce func(event eventstore.Event) (*handler.Statement, error)
		want   wantReduce
	}{
		{
			name: "project.reduceAppAdded",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(project.ApplicationAddedType),
					project.AggregateType,
					[]byte(`{
			"appId": "app-id",
			"name": "my-app"
		}`),
				), project.ApplicationAddedEventMapper),
			},
			reduce: (&AppProjection{}).reduceAppAdded,
			want: wantReduce{
				aggregateType:    eventstore.AggregateType("project"),
				sequence:         15,
				previousSequence: 10,
				projection:       AppProjectionTable,
				executer: &testExecuter{
					executions: []execution{
						{
							expectedStmt: "INSERT INTO projections.apps (id, name, project_id, creation_date, change_date, resource_owner, instance_id, state, sequence) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
							expectedArgs: []interface{}{
								"app-id",
								"my-app",
								"agg-id",
								anyArg{},
								anyArg{},
								"ro-id",
								"instance-id",
								domain.AppStateActive,
								uint64(15),
							},
						},
					},
				},
			},
		},
		{
			name: "project.reduceAppChanged",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(project.ApplicationChangedType),
					project.AggregateType,
					[]byte(`{
			"appId": "app-id",
			"name": "my-app"
		}`),
				), project.ApplicationChangedEventMapper),
			},
			reduce: (&AppProjection{}).reduceAppChanged,
			want: wantReduce{
				aggregateType:    eventstore.AggregateType("project"),
				sequence:         15,
				previousSequence: 10,
				projection:       AppProjectionTable,
				executer: &testExecuter{
					executions: []execution{
						{
							expectedStmt: "UPDATE projections.apps SET (name, change_date, sequence) = ($1, $2, $3) WHERE (id = $4) AND (instance_id = $5)",
							expectedArgs: []interface{}{
								"my-app",
								anyArg{},
								uint64(15),
								"app-id",
								"instance-id",
							},
						},
					},
				},
			},
		},
		{
			name: "project.reduceAppDeactivated",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(project.ApplicationDeactivatedType),
					project.AggregateType,
					[]byte(`{
			"appId": "app-id"
		}`),
				), project.ApplicationDeactivatedEventMapper),
			},
			reduce: (&AppProjection{}).reduceAppDeactivated,
			want: wantReduce{
				aggregateType:    eventstore.AggregateType("project"),
				sequence:         15,
				previousSequence: 10,
				projection:       AppProjectionTable,
				executer: &testExecuter{
					executions: []execution{
						{
							expectedStmt: "UPDATE projections.apps SET (state, change_date, sequence) = ($1, $2, $3) WHERE (id = $4) AND (instance_id = $5)",
							expectedArgs: []interface{}{
								domain.AppStateInactive,
								anyArg{},
								uint64(15),
								"app-id",
								"instance-id",
							},
						},
					},
				},
			},
		},
		{
			name: "project.reduceAppReactivated",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(project.ApplicationReactivatedType),
					project.AggregateType,
					[]byte(`{
			"appId": "app-id"
		}`),
				), project.ApplicationReactivatedEventMapper),
			},
			reduce: (&AppProjection{}).reduceAppReactivated,
			want: wantReduce{
				aggregateType:    eventstore.AggregateType("project"),
				sequence:         15,
				previousSequence: 10,
				projection:       AppProjectionTable,
				executer: &testExecuter{
					executions: []execution{
						{
							expectedStmt: "UPDATE projections.apps SET (state, change_date, sequence) = ($1, $2, $3) WHERE (id = $4) AND (instance_id = $5)",
							expectedArgs: []interface{}{
								domain.AppStateActive,
								anyArg{},
								uint64(15),
								"app-id",
								"instance-id",
							},
						},
					},
				},
			},
		},
		{
			name: "project.reduceAppRemoved",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(project.ApplicationRemovedType),
					project.AggregateType,
					[]byte(`{
			"appId": "app-id"
		}`),
				), project.ApplicationRemovedEventMapper),
			},
			reduce: (&AppProjection{}).reduceAppRemoved,
			want: wantReduce{
				aggregateType:    eventstore.AggregateType("project"),
				sequence:         15,
				previousSequence: 10,
				projection:       AppProjectionTable,
				executer: &testExecuter{
					executions: []execution{
						{
							expectedStmt: "DELETE FROM projections.apps WHERE (id = $1) AND (instance_id = $2)",
							expectedArgs: []interface{}{
								"app-id",
								"instance-id",
							},
						},
					},
				},
			},
		},
		{
			name: "project.reduceProjectRemoved",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(project.ProjectRemovedType),
					project.AggregateType,
					[]byte(`{}`),
				), project.ProjectRemovedEventMapper),
			},
			reduce: (&AppProjection{}).reduceProjectRemoved,
			want: wantReduce{
				aggregateType:    eventstore.AggregateType("project"),
				sequence:         15,
				previousSequence: 10,
				projection:       AppProjectionTable,
				executer: &testExecuter{
					executions: []execution{
						{
							expectedStmt: "DELETE FROM projections.apps WHERE (project_id = $1) AND (instance_id = $2)",
							expectedArgs: []interface{}{
								"agg-id",
								"instance-id",
							},
						},
					},
				},
			},
		},
		{
			name: "project.reduceAPIConfigAdded",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(project.APIConfigAddedType),
					project.AggregateType,
					[]byte(`{
		            "appId": "app-id",
					"clientId": "client-id",
					"clientSecret": {},
				    "authMethodType": 1
				}`),
				), project.APIConfigAddedEventMapper),
			},
			reduce: (&AppProjection{}).reduceAPIConfigAdded,
			want: wantReduce{
				aggregateType:    eventstore.AggregateType("project"),
				sequence:         15,
				previousSequence: 10,
				projection:       AppProjectionTable,
				executer: &testExecuter{
					executions: []execution{
						{
							expectedStmt: "INSERT INTO projections.apps_api_configs (app_id, instance_id, client_id, client_secret, auth_method) VALUES ($1, $2, $3, $4, $5)",
							expectedArgs: []interface{}{
								"app-id",
								"instance-id",
								"client-id",
								anyArg{},
								domain.APIAuthMethodTypePrivateKeyJWT,
							},
						},
						{
							expectedStmt: "UPDATE projections.apps SET (change_date, sequence) = ($1, $2) WHERE (id = $3) AND (instance_id = $4)",
							expectedArgs: []interface{}{
								anyArg{},
								uint64(15),
								"app-id",
								"instance-id",
							},
						},
					},
				},
			},
		},
		{
			name: "project.reduceAPIConfigChanged",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(project.APIConfigChangedType),
					project.AggregateType,
					[]byte(`{
		            "appId": "app-id",
					"clientId": "client-id",
					"clientSecret": {},
				    "authMethodType": 1
				}`),
				), project.APIConfigChangedEventMapper),
			},
			reduce: (&AppProjection{}).reduceAPIConfigChanged,
			want: wantReduce{
				aggregateType:    eventstore.AggregateType("project"),
				sequence:         15,
				previousSequence: 10,
				projection:       AppProjectionTable,
				executer: &testExecuter{
					executions: []execution{
						{
							expectedStmt: "UPDATE projections.apps_api_configs SET (client_secret, auth_method) = ($1, $2) WHERE (app_id = $3) AND (instance_id = $4)",
							expectedArgs: []interface{}{
								anyArg{},
								domain.APIAuthMethodTypePrivateKeyJWT,
								"app-id",
								"instance-id",
							},
						},
						{
							expectedStmt: "UPDATE projections.apps SET (change_date, sequence) = ($1, $2) WHERE (id = $3) AND (instance_id = $4)",
							expectedArgs: []interface{}{
								anyArg{},
								uint64(15),
								"app-id",
								"instance-id",
							},
						},
					},
				},
			},
		},
		{
			name: "project.reduceAPIConfigChanged noop",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(project.APIConfigChangedType),
					project.AggregateType,
					[]byte(`{
		            "appId": "app-id"
				}`),
				), project.APIConfigChangedEventMapper),
			},
			reduce: (&AppProjection{}).reduceAPIConfigChanged,
			want: wantReduce{
				aggregateType:    eventstore.AggregateType("project"),
				sequence:         15,
				previousSequence: 10,
				projection:       AppProjectionTable,
				executer: &testExecuter{
					executions: []execution{},
				},
			},
		},
		{
			name: "project.reduceAPIConfigSecretChanged",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(project.APIConfigSecretChangedType),
					project.AggregateType,
					[]byte(`{
                        "appId": "app-id",
                        "client_secret": {}
		}`),
				), project.APIConfigSecretChangedEventMapper),
			},
			reduce: (&AppProjection{}).reduceAPIConfigSecretChanged,
			want: wantReduce{
				aggregateType:    eventstore.AggregateType("project"),
				sequence:         15,
				previousSequence: 10,
				projection:       AppProjectionTable,
				executer: &testExecuter{
					executions: []execution{
						{
							expectedStmt: "UPDATE projections.apps_api_configs SET (client_secret) = ($1) WHERE (app_id = $2) AND (instance_id = $3)",
							expectedArgs: []interface{}{
								anyArg{},
								"app-id",
								"instance-id",
							},
						},
						{
							expectedStmt: "UPDATE projections.apps SET (change_date, sequence) = ($1, $2) WHERE (id = $3) AND (instance_id = $4)",
							expectedArgs: []interface{}{
								anyArg{},
								uint64(15),
								"app-id",
								"instance-id",
							},
						},
					},
				},
			},
		},
		{
			name: "project.reduceOIDCConfigAdded",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(project.OIDCConfigAddedType),
					project.AggregateType,
					[]byte(`{
                        "oidcVersion": 0,
                        "appId": "app-id",
                        "clientId": "client-id",
                        "clientSecret": {},
                        "redirectUris": ["redirect.one.ch", "redirect.two.ch"],
                        "responseTypes": [1,2],
                        "grantTypes": [1,2],
                        "applicationType": 2,
                        "authMethodType": 2,
                        "postLogoutRedirectUris": ["logout.one.ch", "logout.two.ch"],
                        "devMode": true,
                        "accessTokenType": 1,
                        "accessTokenRoleAssertion": true,
                        "idTokenRoleAssertion": true,
                        "idTokenUserinfoAssertion": true,
                        "clockSkew": 1000,
                        "additionalOrigins": ["origin.one.ch", "origin.two.ch"]
		}`),
				), project.OIDCConfigAddedEventMapper),
			},
			reduce: (&AppProjection{}).reduceOIDCConfigAdded,
			want: wantReduce{
				aggregateType:    eventstore.AggregateType("project"),
				sequence:         15,
				previousSequence: 10,
				projection:       AppProjectionTable,
				executer: &testExecuter{
					executions: []execution{
						{
							expectedStmt: "INSERT INTO projections.apps_oidc_configs (app_id, instance_id, version, client_id, client_secret, redirect_uris, response_types, grant_types, application_type, auth_method_type, post_logout_redirect_uris, is_dev_mode, access_token_type, access_token_role_assertion, id_token_role_assertion, id_token_userinfo_assertion, clock_skew, additional_origins) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)",
							expectedArgs: []interface{}{
								"app-id",
								"instance-id",
								domain.OIDCVersionV1,
								"client-id",
								anyArg{},
								pq.StringArray{"redirect.one.ch", "redirect.two.ch"},
								pq.Array([]domain.OIDCResponseType{1, 2}),
								pq.Array([]domain.OIDCGrantType{1, 2}),
								domain.OIDCApplicationTypeNative,
								domain.OIDCAuthMethodTypeNone,
								pq.StringArray{"logout.one.ch", "logout.two.ch"},
								true,
								domain.OIDCTokenTypeJWT,
								true,
								true,
								true,
								1 * time.Microsecond,
								pq.StringArray{"origin.one.ch", "origin.two.ch"},
							},
						},
						{
							expectedStmt: "UPDATE projections.apps SET (change_date, sequence) = ($1, $2) WHERE (id = $3) AND (instance_id = $4)",
							expectedArgs: []interface{}{
								anyArg{},
								uint64(15),
								"app-id",
								"instance-id",
							},
						},
					},
				},
			},
		},
		{
			name: "project.reduceOIDCConfigChanged",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(project.OIDCConfigChangedType),
					project.AggregateType,
					[]byte(`{
                        "oidcVersion": 0,
                        "appId": "app-id",
                        "redirectUris": ["redirect.one.ch", "redirect.two.ch"],
                        "responseTypes": [1,2],
                        "grantTypes": [1,2],
                        "applicationType": 2,
                        "authMethodType": 2,
                        "postLogoutRedirectUris": ["logout.one.ch", "logout.two.ch"],
                        "devMode": true,
                        "accessTokenType": 1,
                        "accessTokenRoleAssertion": true,
                        "idTokenRoleAssertion": true,
                        "idTokenUserinfoAssertion": true,
                        "clockSkew": 1000,
                        "additionalOrigins": ["origin.one.ch", "origin.two.ch"]
		}`),
				), project.OIDCConfigChangedEventMapper),
			},
			reduce: (&AppProjection{}).reduceOIDCConfigChanged,
			want: wantReduce{
				aggregateType:    eventstore.AggregateType("project"),
				sequence:         15,
				previousSequence: 10,
				projection:       AppProjectionTable,
				executer: &testExecuter{
					executions: []execution{
						{
							expectedStmt: "UPDATE projections.apps_oidc_configs SET (version, redirect_uris, response_types, grant_types, application_type, auth_method_type, post_logout_redirect_uris, is_dev_mode, access_token_type, access_token_role_assertion, id_token_role_assertion, id_token_userinfo_assertion, clock_skew, additional_origins) = ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) WHERE (app_id = $15) AND (instance_id = $16)",
							expectedArgs: []interface{}{
								domain.OIDCVersionV1,
								pq.StringArray{"redirect.one.ch", "redirect.two.ch"},
								pq.Array([]domain.OIDCResponseType{1, 2}),
								pq.Array([]domain.OIDCGrantType{1, 2}),
								domain.OIDCApplicationTypeNative,
								domain.OIDCAuthMethodTypeNone,
								pq.StringArray{"logout.one.ch", "logout.two.ch"},
								true,
								domain.OIDCTokenTypeJWT,
								true,
								true,
								true,
								1 * time.Microsecond,
								pq.StringArray{"origin.one.ch", "origin.two.ch"},
								"app-id",
								"instance-id",
							},
						},
						{
							expectedStmt: "UPDATE projections.apps SET (change_date, sequence) = ($1, $2) WHERE (id = $3) AND (instance_id = $4)",
							expectedArgs: []interface{}{
								anyArg{},
								uint64(15),
								"app-id",
								"instance-id",
							},
						},
					},
				},
			},
		},
		{
			name: "project.reduceOIDCConfigChanged noop",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(project.OIDCConfigChangedType),
					project.AggregateType,
					[]byte(`{
                        "appId": "app-id"
		}`),
				), project.OIDCConfigChangedEventMapper),
			},
			reduce: (&AppProjection{}).reduceOIDCConfigChanged,
			want: wantReduce{
				aggregateType:    eventstore.AggregateType("project"),
				sequence:         15,
				previousSequence: 10,
				projection:       AppProjectionTable,
				executer: &testExecuter{
					executions: []execution{},
				},
			},
		},
		{
			name: "project.reduceOIDCConfigSecretChanged",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(project.OIDCConfigSecretChangedType),
					project.AggregateType,
					[]byte(`{
                        "appId": "app-id",
                        "client_secret": {}
		}`),
				), project.OIDCConfigSecretChangedEventMapper),
			},
			reduce: (&AppProjection{}).reduceOIDCConfigSecretChanged,
			want: wantReduce{
				aggregateType:    eventstore.AggregateType("project"),
				sequence:         15,
				previousSequence: 10,
				projection:       AppProjectionTable,
				executer: &testExecuter{
					executions: []execution{
						{
							expectedStmt: "UPDATE projections.apps_oidc_configs SET (client_secret) = ($1) WHERE (app_id = $2) AND (instance_id = $3)",
							expectedArgs: []interface{}{
								anyArg{},
								"app-id",
								"instance-id",
							},
						},
						{
							expectedStmt: "UPDATE projections.apps SET (change_date, sequence) = ($1, $2) WHERE (id = $3) AND (instance_id = $4)",
							expectedArgs: []interface{}{
								anyArg{},
								uint64(15),
								"app-id",
								"instance-id",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := baseEvent(t)
			got, err := tt.reduce(event)
			if _, ok := err.(errors.InvalidArgument); !ok {
				t.Errorf("no wrong event mapping: %v, got: %v", err, got)
			}

			event = tt.args.event(t)
			got, err = tt.reduce(event)
			assertReduce(t, got, err, tt.want)
		})
	}
}
