package command

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/text/language"

	"github.com/zitadel/zitadel/internal/api/authz"
	"github.com/zitadel/zitadel/internal/api/http"
	"github.com/zitadel/zitadel/internal/command/preparation"
	"github.com/zitadel/zitadel/internal/crypto"
	"github.com/zitadel/zitadel/internal/domain"
	"github.com/zitadel/zitadel/internal/errors"
	"github.com/zitadel/zitadel/internal/eventstore"
	"github.com/zitadel/zitadel/internal/eventstore/repository"
	"github.com/zitadel/zitadel/internal/eventstore/v1/models"
	"github.com/zitadel/zitadel/internal/id"
	id_mock "github.com/zitadel/zitadel/internal/id/mock"
	"github.com/zitadel/zitadel/internal/repository/org"
	"github.com/zitadel/zitadel/internal/repository/user"
)

func TestAddDomain(t *testing.T) {
	type args struct {
		a      *org.Aggregate
		domain string
		filter preparation.FilterToQueryReducer
	}

	agg := org.NewAggregate("test")

	tests := []struct {
		name string
		args args
		want Want
	}{
		{
			name: "invalid domain",
			args: args{
				a:      agg,
				domain: "",
			},
			want: Want{
				ValidationErr: errors.ThrowInvalidArgument(nil, "ORG-r3h4J", "Errors.Invalid.Argument"),
			},
		},
		{
			name: "correct (should verify domain)",
			args: args{
				a:      agg,
				domain: "domain",
				filter: func(ctx context.Context, queryFactory *eventstore.SearchQueryBuilder) ([]eventstore.Event, error) {
					return []eventstore.Event{
						org.NewDomainPolicyAddedEvent(ctx, &agg.Aggregate, true, true, true),
					}, nil
				},
			},
			want: Want{
				Commands: []eventstore.Command{
					org.NewDomainAddedEvent(context.Background(), &agg.Aggregate, "domain"),
				},
			},
		},
		{
			name: "correct (should not verify domain)",
			args: args{
				a:      agg,
				domain: "domain",
				filter: func(ctx context.Context, queryFactory *eventstore.SearchQueryBuilder) ([]eventstore.Event, error) {
					return []eventstore.Event{
						org.NewDomainPolicyAddedEvent(ctx, &agg.Aggregate, true, false, false),
					}, nil
				},
			},
			want: Want{
				Commands: []eventstore.Command{
					org.NewDomainAddedEvent(context.Background(), &agg.Aggregate, "domain"),
					org.NewDomainVerifiedEvent(context.Background(), &agg.Aggregate, "domain"),
				},
			},
		},
		{
			name: "already verified",
			args: args{
				a:      agg,
				domain: "domain",
				filter: func(ctx context.Context, queryFactory *eventstore.SearchQueryBuilder) ([]eventstore.Event, error) {
					return []eventstore.Event{
						org.NewDomainAddedEvent(ctx, &agg.Aggregate, "domain"),
						org.NewDomainVerificationAddedEvent(ctx, &agg.Aggregate, "domain", domain.OrgDomainValidationTypeHTTP, nil),
						org.NewDomainVerifiedEvent(ctx, &agg.Aggregate, "domain"),
					}, nil
				},
			},
			want: Want{
				CreateErr: errors.ThrowAlreadyExists(nil, "", ""),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AssertValidation(t, AddOrgDomain(tt.args.a, tt.args.domain), tt.args.filter, tt.want)
		})
	}
}

func TestVerifyDomain(t *testing.T) {
	type args struct {
		a      *org.Aggregate
		domain string
	}

	tests := []struct {
		name string
		args args
		want Want
	}{
		{
			name: "invalid domain",
			args: args{
				a:      org.NewAggregate("test"),
				domain: "",
			},
			want: Want{
				ValidationErr: errors.ThrowInvalidArgument(nil, "ORG-yqlVQ", "Errors.Invalid.Argument"),
			},
		},
		{
			name: "correct",
			args: args{
				a:      org.NewAggregate("test"),
				domain: "domain",
			},
			want: Want{
				Commands: []eventstore.Command{
					org.NewDomainVerifiedEvent(context.Background(), &org.NewAggregate("test").Aggregate, "domain"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AssertValidation(t, VerifyOrgDomain(tt.args.a, tt.args.domain), nil, tt.want)
		})
	}
}

func TestSetDomainPrimary(t *testing.T) {
	type args struct {
		a      *org.Aggregate
		domain string
		filter preparation.FilterToQueryReducer
	}

	agg := org.NewAggregate("test")

	tests := []struct {
		name string
		args args
		want Want
	}{
		{
			name: "invalid domain",
			args: args{
				a:      agg,
				domain: "",
			},
			want: Want{
				ValidationErr: errors.ThrowInvalidArgument(nil, "ORG-gmNqY", "Errors.Invalid.Argument"),
			},
		},
		{
			name: "not exists",
			args: args{
				a:      agg,
				domain: "domain",
				filter: func(ctx context.Context, queryFactory *eventstore.SearchQueryBuilder) ([]eventstore.Event, error) {
					return nil, nil
				},
			},
			want: Want{
				CreateErr: errors.ThrowNotFound(nil, "", ""),
			},
		},
		{
			name: "not verified",
			args: args{
				a:      agg,
				domain: "domain",
				filter: func(ctx context.Context, queryFactory *eventstore.SearchQueryBuilder) ([]eventstore.Event, error) {
					return []eventstore.Event{org.NewDomainAddedEvent(ctx, &agg.Aggregate, "domain")}, nil
				},
			},
			want: Want{
				CreateErr: errors.ThrowPreconditionFailed(nil, "", ""),
			},
		},
		{
			name: "already primary",
			args: args{
				a:      agg,
				domain: "domain",
				filter: func(ctx context.Context, queryFactory *eventstore.SearchQueryBuilder) ([]eventstore.Event, error) {
					return []eventstore.Event{
						org.NewDomainAddedEvent(ctx, &agg.Aggregate, "domain"),
						org.NewDomainVerificationAddedEvent(ctx, &agg.Aggregate, "domain", domain.OrgDomainValidationTypeHTTP, nil),
						org.NewDomainVerifiedEvent(ctx, &agg.Aggregate, "domain"),
						org.NewDomainPrimarySetEvent(ctx, &agg.Aggregate, "domain"),
					}, nil
				},
			},
			want: Want{
				CreateErr: errors.ThrowPreconditionFailed(nil, "", ""),
			},
		},
		{
			name: "correct",
			args: args{
				a:      agg,
				domain: "domain",
				filter: func(ctx context.Context, queryFactory *eventstore.SearchQueryBuilder) ([]eventstore.Event, error) {
					return []eventstore.Event{
						org.NewDomainAddedEvent(ctx, &agg.Aggregate, "domain"),
						org.NewDomainVerificationAddedEvent(ctx, &agg.Aggregate, "domain", domain.OrgDomainValidationTypeHTTP, nil),
						org.NewDomainVerifiedEvent(ctx, &agg.Aggregate, "domain"),
					}, nil
				},
			},
			want: Want{
				Commands: []eventstore.Command{
					org.NewDomainPrimarySetEvent(context.Background(), &agg.Aggregate, "domain"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AssertValidation(t, SetPrimaryOrgDomain(tt.args.a, tt.args.domain), tt.args.filter, tt.want)
		})
	}
}

func TestCommandSide_AddOrgDomain(t *testing.T) {
	type fields struct {
		eventstore *eventstore.Eventstore
	}
	type args struct {
		ctx            context.Context
		domain         *domain.OrgDomain
		claimedUserIDs []string
	}
	type res struct {
		want *domain.OrgDomain
		err  func(error) bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		res    res
	}{
		{
			name: "invalid domain, error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
				),
			},
			args: args{
				ctx:    context.Background(),
				domain: &domain.OrgDomain{},
			},
			res: res{
				err: errors.IsErrorInvalidArgument,
			},
		},
		{
			name: "domain already exists, precondition error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
						eventFromEventPusher(
							org.NewDomainAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
					),
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain: "domain.ch",
				},
			},
			res: res{
				err: errors.IsErrorAlreadyExists,
			},
		},
		{
			name: "domain add, ok",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
					),
					expectPush(
						[]*repository.Event{
							eventFromEventPusher(org.NewDomainAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							)),
						},
					),
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain: "domain.ch",
				},
			},
			res: res{
				want: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID:   "org1",
						ResourceOwner: "org1",
					},
					Domain: "domain.ch",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Commands{
				eventstore: tt.fields.eventstore,
			}
			got, err := r.AddOrgDomain(tt.args.ctx, tt.args.domain, tt.args.claimedUserIDs)
			if tt.res.err == nil {
				assert.NoError(t, err)
			}
			if tt.res.err != nil && !tt.res.err(err) {
				t.Errorf("got wrong err: %v ", err)
			}
			if tt.res.err == nil {
				assert.Equal(t, tt.res.want, got)
			}
		})
	}
}

func TestCommandSide_GenerateOrgDomainValidation(t *testing.T) {
	type fields struct {
		eventstore      *eventstore.Eventstore
		secretGenerator crypto.Generator
	}
	type args struct {
		ctx    context.Context
		domain *domain.OrgDomain
	}
	type res struct {
		wantToken string
		wantURL   string
		err       func(error) bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		res    res
	}{
		{
			name: "invalid domain, error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
				},
			},
			res: res{
				err: errors.IsErrorInvalidArgument,
			},
		},
		{
			name: "missing aggregateid, error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					Domain: "domain.ch",
				},
			},
			res: res{
				err: errors.IsErrorInvalidArgument,
			},
		},
		{
			name: "invalid validation type, error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain: "domain.ch",
				},
			},
			res: res{
				err: errors.IsErrorInvalidArgument,
			},
		},
		{
			name: "domain not exists, precondition error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
					),
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain:         "domain.ch",
					ValidationType: domain.OrgDomainValidationTypeDNS,
				},
			},
			res: res{
				err: errors.IsNotFound,
			},
		},
		{
			name: "domain already verified, precondition error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
						eventFromEventPusher(
							org.NewDomainAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
						eventFromEventPusher(
							org.NewDomainVerifiedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
					),
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain:         "domain.ch",
					ValidationType: domain.OrgDomainValidationTypeDNS,
				},
			},
			res: res{
				err: errors.IsPreconditionFailed,
			},
		},
		{
			name: "add dns validation, ok",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
						eventFromEventPusher(
							org.NewDomainAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
					),
					expectPush(
						[]*repository.Event{
							eventFromEventPusher(org.NewDomainVerificationAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
								domain.OrgDomainValidationTypeDNS,
								&crypto.CryptoValue{
									CryptoType: crypto.TypeEncryption,
									Algorithm:  "enc",
									KeyID:      "id",
									Crypted:    []byte("a"),
								},
							)),
						},
					),
				),
				secretGenerator: GetMockSecretGenerator(t),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain:         "domain.ch",
					ValidationType: domain.OrgDomainValidationTypeDNS,
				},
			},
			res: res{
				wantToken: "a",
				wantURL:   "_zitadel-challenge.domain.ch",
			},
		},
		{
			name: "add http validation, ok",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
						eventFromEventPusher(
							org.NewDomainAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
					),
					expectPush(
						[]*repository.Event{
							eventFromEventPusher(org.NewDomainVerificationAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
								domain.OrgDomainValidationTypeHTTP,
								&crypto.CryptoValue{
									CryptoType: crypto.TypeEncryption,
									Algorithm:  "enc",
									KeyID:      "id",
									Crypted:    []byte("a"),
								},
							)),
						},
					),
				),
				secretGenerator: GetMockSecretGenerator(t),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain:         "domain.ch",
					ValidationType: domain.OrgDomainValidationTypeHTTP,
				},
			},
			res: res{
				wantToken: "a",
				wantURL:   "https://domain.ch/.well-known/zitadel-challenge/a",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Commands{
				eventstore:                  tt.fields.eventstore,
				domainVerificationGenerator: tt.fields.secretGenerator,
			}
			token, url, err := r.GenerateOrgDomainValidation(tt.args.ctx, tt.args.domain)
			if tt.res.err == nil {
				assert.NoError(t, err)
			}
			if tt.res.err != nil && !tt.res.err(err) {
				t.Errorf("got wrong err: %v ", err)
			}
			if tt.res.err == nil {
				assert.Equal(t, tt.res.wantToken, token)
				assert.Equal(t, tt.res.wantURL, url)
			}
		})
	}
}

func TestCommandSide_ValidateOrgDomain(t *testing.T) {
	type fields struct {
		eventstore           *eventstore.Eventstore
		idGenerator          id.Generator
		secretGenerator      crypto.Generator
		alg                  crypto.EncryptionAlgorithm
		domainValidationFunc func(domain, token, verifier string, checkType http.CheckType) error
	}
	type args struct {
		ctx            context.Context
		domain         *domain.OrgDomain
		claimedUserIDs []string
	}
	type res struct {
		want *domain.ObjectDetails
		err  func(error) bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		res    res
	}{
		{
			name: "invalid domain, error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
				},
			},
			res: res{
				err: errors.IsErrorInvalidArgument,
			},
		},
		{
			name: "missing aggregateid, error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					Domain: "domain.ch",
				},
			},
			res: res{
				err: errors.IsErrorInvalidArgument,
			},
		},
		{
			name: "domain not exists, precondition error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
					),
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain:         "domain.ch",
					ValidationType: domain.OrgDomainValidationTypeDNS,
				},
			},
			res: res{
				err: errors.IsNotFound,
			},
		},
		{
			name: "domain already verified, precondition error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
						eventFromEventPusher(
							org.NewDomainAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
						eventFromEventPusher(
							org.NewDomainVerifiedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
					),
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain:         "domain.ch",
					ValidationType: domain.OrgDomainValidationTypeDNS,
				},
			},
			res: res{
				err: errors.IsPreconditionFailed,
			},
		},
		{
			name: "no code existing, precondition error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
						eventFromEventPusher(
							org.NewDomainAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
					),
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain:         "domain.ch",
					ValidationType: domain.OrgDomainValidationTypeDNS,
				},
			},
			res: res{
				err: errors.IsPreconditionFailed,
			},
		},
		{
			name: "invalid domain verification, precondition error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
						eventFromEventPusher(
							org.NewDomainAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
						eventFromEventPusher(
							org.NewDomainVerificationAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
								domain.OrgDomainValidationTypeDNS,
								&crypto.CryptoValue{
									CryptoType: crypto.TypeEncryption,
									Algorithm:  "enc",
									KeyID:      "id",
									Crypted:    []byte("a"),
								},
							),
						),
					),
					expectPush(
						[]*repository.Event{
							eventFromEventPusher(org.NewDomainVerificationFailedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							)),
						},
					),
				),
				alg:                  crypto.CreateMockEncryptionAlg(gomock.NewController(t)),
				domainValidationFunc: invalidDomainVerification,
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain:         "domain.ch",
					ValidationType: domain.OrgDomainValidationTypeDNS,
				},
			},
			res: res{
				err: errors.IsErrorInvalidArgument,
			},
		},
		{
			name: "domain verification, ok",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
						eventFromEventPusher(
							org.NewDomainAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
						eventFromEventPusher(
							org.NewDomainVerificationAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
								domain.OrgDomainValidationTypeDNS,
								&crypto.CryptoValue{
									CryptoType: crypto.TypeEncryption,
									Algorithm:  "enc",
									KeyID:      "id",
									Crypted:    []byte("a"),
								},
							),
						),
					),
					expectPush(
						[]*repository.Event{
							eventFromEventPusher(org.NewDomainVerifiedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							)),
						},
						uniqueConstraintsFromEventConstraint(org.NewAddOrgDomainUniqueConstraint("domain.ch")),
					),
				),
				alg:                  crypto.CreateMockEncryptionAlg(gomock.NewController(t)),
				domainValidationFunc: validDomainVerification,
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain:         "domain.ch",
					ValidationType: domain.OrgDomainValidationTypeDNS,
				},
			},
			res: res{
				want: &domain.ObjectDetails{
					ResourceOwner: "org1",
				},
			},
		},
		{
			name: "domain verification, claimed users not found, ok",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
						eventFromEventPusher(
							org.NewDomainAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
						eventFromEventPusher(
							org.NewDomainVerificationAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
								domain.OrgDomainValidationTypeDNS,
								&crypto.CryptoValue{
									CryptoType: crypto.TypeEncryption,
									Algorithm:  "enc",
									KeyID:      "id",
									Crypted:    []byte("a"),
								},
							),
						),
					),
					expectFilter(),
					expectPush(
						[]*repository.Event{
							eventFromEventPusher(org.NewDomainVerifiedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							)),
						},
						uniqueConstraintsFromEventConstraint(org.NewAddOrgDomainUniqueConstraint("domain.ch")),
					),
				),
				alg:                  crypto.CreateMockEncryptionAlg(gomock.NewController(t)),
				domainValidationFunc: validDomainVerification,
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain:         "domain.ch",
					ValidationType: domain.OrgDomainValidationTypeDNS,
				},
				claimedUserIDs: []string{"user1"},
			},
			res: res{
				want: &domain.ObjectDetails{
					ResourceOwner: "org1",
				},
			},
		},
		{
			name: "domain verification, claimed users, ok",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
						eventFromEventPusher(
							org.NewDomainAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
						eventFromEventPusher(
							org.NewDomainVerificationAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
								domain.OrgDomainValidationTypeDNS,
								&crypto.CryptoValue{
									CryptoType: crypto.TypeEncryption,
									Algorithm:  "enc",
									KeyID:      "id",
									Crypted:    []byte("a"),
								},
							),
						),
					),
					expectFilter(
						eventFromEventPusher(
							user.NewHumanAddedEvent(context.Background(),
								&user.NewAggregate("user1", "org2").Aggregate,
								"username@domain.ch",
								"firstname",
								"lastname",
								"nickname",
								"displayname",
								language.German,
								domain.GenderUnspecified,
								"email",
								true,
							),
						),
					),
					expectFilter(
						eventFromEventPusher(
							org.NewDomainPolicyAddedEvent(context.Background(),
								&org.NewAggregate("org2").Aggregate,
								false, false, false))),
					expectPush(
						[]*repository.Event{
							eventFromEventPusher(org.NewDomainVerifiedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							)),
							eventFromEventPusher(user.NewDomainClaimedEvent(context.Background(),
								&user.NewAggregate("user1", "org2").Aggregate,
								"tempid@temporary.zitadel.ch",
								"username@domain.ch",
								false,
							)),
						},
						uniqueConstraintsFromEventConstraint(org.NewAddOrgDomainUniqueConstraint("domain.ch")),
						uniqueConstraintsFromEventConstraint(user.NewRemoveUsernameUniqueConstraint("username@domain.ch", "org2", false)),
						uniqueConstraintsFromEventConstraint(user.NewAddUsernameUniqueConstraint("tempid@temporary.zitadel.ch", "org2", false)),
					),
				),
				alg:                  crypto.CreateMockEncryptionAlg(gomock.NewController(t)),
				domainValidationFunc: validDomainVerification,
				idGenerator:          id_mock.NewIDGeneratorExpectIDs(t, "tempid"),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain:         "domain.ch",
					ValidationType: domain.OrgDomainValidationTypeDNS,
				},
				claimedUserIDs: []string{"user1"},
			},
			res: res{
				want: &domain.ObjectDetails{
					ResourceOwner: "org1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Commands{
				eventstore:                  tt.fields.eventstore,
				domainVerificationGenerator: tt.fields.secretGenerator,
				domainVerificationAlg:       tt.fields.alg,
				domainVerificationValidator: tt.fields.domainValidationFunc,
				idGenerator:                 tt.fields.idGenerator,
			}
			got, err := r.ValidateOrgDomain(authz.WithRequestedDomain(tt.args.ctx, "zitadel.ch"), tt.args.domain, tt.args.claimedUserIDs)
			if tt.res.err == nil {
				assert.NoError(t, err)
			}
			if tt.res.err != nil && !tt.res.err(err) {
				t.Errorf("got wrong err: %v ", err)
			}
			if tt.res.err == nil {
				assert.Equal(t, tt.res.want, got)
			}
		})
	}
}

func TestCommandSide_SetPrimaryDomain(t *testing.T) {
	type fields struct {
		eventstore *eventstore.Eventstore
	}
	type args struct {
		ctx    context.Context
		domain *domain.OrgDomain
	}
	type res struct {
		want *domain.ObjectDetails
		err  func(error) bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		res    res
	}{
		{
			name: "invalid domain, error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
				},
			},
			res: res{
				err: errors.IsErrorInvalidArgument,
			},
		},
		{
			name: "missing aggregateid, error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					Domain: "domain.ch",
				},
			},
			res: res{
				err: errors.IsErrorInvalidArgument,
			},
		},
		{
			name: "domain not exists, precondition error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
					),
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain:         "domain.ch",
					ValidationType: domain.OrgDomainValidationTypeDNS,
				},
			},
			res: res{
				err: errors.IsNotFound,
			},
		},
		{
			name: "domain not verified, precondition error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
						eventFromEventPusher(
							org.NewDomainAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
					),
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain: "domain.ch",
				},
			},
			res: res{
				err: errors.IsPreconditionFailed,
			},
		},
		{
			name: "set primary, ok",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
						eventFromEventPusher(
							org.NewDomainAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
						eventFromEventPusher(
							org.NewDomainVerifiedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
					),
					expectPush(
						[]*repository.Event{
							eventFromEventPusher(org.NewDomainPrimarySetEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							)),
						},
					),
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain: "domain.ch",
				},
			},
			res: res{
				want: &domain.ObjectDetails{
					ResourceOwner: "org1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Commands{
				eventstore: tt.fields.eventstore,
			}
			got, err := r.SetPrimaryOrgDomain(tt.args.ctx, tt.args.domain)
			if tt.res.err == nil {
				assert.NoError(t, err)
			}
			if tt.res.err != nil && !tt.res.err(err) {
				t.Errorf("got wrong err: %v ", err)
			}
			if tt.res.err == nil {
				assert.Equal(t, tt.res.want, got)
			}
		})
	}
}

func TestCommandSide_RemoveOrgDomain(t *testing.T) {
	type fields struct {
		eventstore *eventstore.Eventstore
	}
	type args struct {
		ctx    context.Context
		domain *domain.OrgDomain
	}
	type res struct {
		want *domain.ObjectDetails
		err  func(error) bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		res    res
	}{
		{
			name: "invalid domain, error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
				},
			},
			res: res{
				err: errors.IsErrorInvalidArgument,
			},
		},
		{
			name: "missing aggregateid, error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					Domain: "domain.ch",
				},
			},
			res: res{
				err: errors.IsErrorInvalidArgument,
			},
		},
		{
			name: "domain not exists, precondition error",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
					),
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain:         "domain.ch",
					ValidationType: domain.OrgDomainValidationTypeDNS,
				},
			},
			res: res{
				err: errors.IsNotFound,
			},
		},
		{
			name: "remove verified domain, ok",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
						eventFromEventPusher(
							org.NewDomainAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
						eventFromEventPusher(
							org.NewDomainVerifiedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
						eventFromEventPusher(
							org.NewDomainPrimarySetEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
					),
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain: "domain.ch",
				},
			},
			res: res{
				err: errors.IsPreconditionFailed,
			},
		},
		{
			name: "remove domain, ok",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
						eventFromEventPusher(
							org.NewDomainAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
					),
					expectPush(
						[]*repository.Event{
							eventFromEventPusher(org.NewDomainRemovedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch", false,
							)),
						},
					),
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain: "domain.ch",
				},
			},
			res: res{
				want: &domain.ObjectDetails{
					ResourceOwner: "org1",
				},
			},
		},
		{
			name: "remove verified domain, ok",
			fields: fields{
				eventstore: eventstoreExpect(
					t,
					expectFilter(
						eventFromEventPusher(
							org.NewOrgAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"name",
							),
						),
						eventFromEventPusher(
							org.NewDomainAddedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
						eventFromEventPusher(
							org.NewDomainVerifiedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch",
							),
						),
					),
					expectPush(
						[]*repository.Event{
							eventFromEventPusher(org.NewDomainRemovedEvent(context.Background(),
								&org.NewAggregate("org1").Aggregate,
								"domain.ch", true,
							)),
						},
						uniqueConstraintsFromEventConstraint(org.NewRemoveOrgDomainUniqueConstraint("domain.ch")),
					),
				),
			},
			args: args{
				ctx: context.Background(),
				domain: &domain.OrgDomain{
					ObjectRoot: models.ObjectRoot{
						AggregateID: "org1",
					},
					Domain: "domain.ch",
				},
			},
			res: res{
				want: &domain.ObjectDetails{
					ResourceOwner: "org1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Commands{
				eventstore: tt.fields.eventstore,
			}
			got, err := r.RemoveOrgDomain(tt.args.ctx, tt.args.domain)
			if tt.res.err == nil {
				assert.NoError(t, err)
			}
			if tt.res.err != nil && !tt.res.err(err) {
				t.Errorf("got wrong err: %v ", err)
			}
			if tt.res.err == nil {
				assert.Equal(t, tt.res.want, got)
			}
		})
	}
}

func invalidDomainVerification(domain, token, verifier string, checkType http.CheckType) error {
	return errors.ThrowInvalidArgument(nil, "HTTP-GH422", "Errors.Internal")
}

func validDomainVerification(domain, token, verifier string, checkType http.CheckType) error {
	return nil
}
