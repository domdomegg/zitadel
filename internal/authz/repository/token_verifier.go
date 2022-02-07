package repository

import (
	"context"
	"time"
)

type TokenVerifierRepository interface {
	VerifyAccessToken(ctx context.Context, tokenString, verifierClientID, projectID string) (userID string, agentID string, clientID, prefLang, resourceOwner string, creationDate time.Time, err error)
	ProjectIDAndOriginsByClientID(ctx context.Context, clientID string) (projectID string, origins []string, err error)
	CheckOrgFeatures(ctx context.Context, orgID string, requiredFeatures ...string) error
	VerifierClientID(ctx context.Context, appName string) (clientID, projectID string, err error)
}
