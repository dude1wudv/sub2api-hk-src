package repository

import (
	"context"
	"errors"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/ent/workbenchcredential"
	"github.com/Wei-Shaw/sub2api/ent/workbenchlaunchgrant"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type workbenchIntegrationRepository struct {
	client *dbent.Client
}

func NewWorkbenchIntegrationRepository(client *dbent.Client) service.WorkbenchIntegrationRepository {
	return &workbenchIntegrationRepository{client: client}
}

func (r *workbenchIntegrationRepository) withTx(ctx context.Context, fn func(context.Context, *dbent.Client) error) error {
	if r == nil || r.client == nil {
		return errors.New("nil ent client")
	}
	if existing := dbent.TxFromContext(ctx); existing != nil {
		return fn(ctx, existing.Client())
	}
	tx, err := r.client.Tx(ctx)
	if errors.Is(err, dbent.ErrTxStarted) {
		return fn(ctx, r.client)
	}
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	if err := fn(txCtx, tx.Client()); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *workbenchIntegrationRepository) EnsureCredential(ctx context.Context, userID int64, workbench, apiKey string) (*service.WorkbenchCredential, error) {
	if userID <= 0 || workbench != service.WorkbenchHeliosGen || apiKey == "" {
		return nil, service.ErrWorkbenchUnavailable
	}
	var result *service.WorkbenchCredential
	err := r.withTx(ctx, func(txCtx context.Context, client *dbent.Client) error {
		// Locking the owner row serializes first-use requests without relying on
		// API-key names. The unique constraint remains the final integrity guard.
		owner, err := client.User.Query().
			Where(user.IDEQ(userID), user.DeletedAtIsNil()).
			ForUpdate().
			Only(txCtx)
		if err != nil {
			if dbent.IsNotFound(err) {
				return service.ErrWorkbenchUnavailable
			}
			return err
		}
		if owner.Status != service.StatusActive {
			return service.ErrWorkbenchUnavailable
		}

		binding, err := client.WorkbenchCredential.Query().
			Where(workbenchcredential.UserIDEQ(userID), workbenchcredential.WorkbenchEQ(workbench)).
			Only(txCtx)
		if err != nil && !dbent.IsNotFound(err) {
			return err
		}
		if err == nil {
			boundKey, keyErr := client.APIKey.Query().
				Where(apikey.IDEQ(binding.APIKeyID), apikey.DeletedAtIsNil()).
				Only(txCtx)
			if keyErr == nil {
				if boundKey.UserID != userID {
					return errors.New("workbench credential owner mismatch")
				}
				if boundKey.Status != service.StatusAPIKeyActive {
					return service.ErrWorkbenchKeyDisabled
				}
				result = workbenchCredentialEntityToService(binding)
				return nil
			}
			if !dbent.IsNotFound(keyErr) {
				return keyErr
			}
		}

		created, err := client.APIKey.Create().
			SetUserID(userID).
			SetKey(apiKey).
			SetName(service.WorkbenchKeyName).
			SetStatus(service.StatusAPIKeyActive).
			SetQuota(0).
			SetQuotaUsed(0).
			SetRateLimit5h(0).
			SetRateLimit1d(0).
			SetRateLimit7d(0).
			Save(txCtx)
		if err != nil {
			return translatePersistenceError(err, nil, service.ErrAPIKeyExists)
		}
		if binding == nil {
			binding, err = client.WorkbenchCredential.Create().
				SetUserID(userID).
				SetWorkbench(workbench).
				SetAPIKeyID(created.ID).
				Save(txCtx)
		} else {
			binding, err = client.WorkbenchCredential.UpdateOneID(binding.ID).
				SetAPIKeyID(created.ID).
				Save(txCtx)
		}
		if err != nil {
			return translatePersistenceError(err, nil, service.ErrAPIKeyExists)
		}
		result = workbenchCredentialEntityToService(binding)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *workbenchIntegrationRepository) CreateGrant(ctx context.Context, grant *service.WorkbenchLaunchGrant) error {
	if r == nil || r.client == nil {
		return service.ErrWorkbenchUnavailable
	}
	if grant == nil || grant.CodeHash == "" || grant.UserID <= 0 || grant.APIKeyID <= 0 {
		return service.ErrWorkbenchInvalidGrant
	}
	created, err := r.client.WorkbenchLaunchGrant.Create().
		SetCodeHash(grant.CodeHash).
		SetUserID(grant.UserID).
		SetAPIKeyID(grant.APIKeyID).
		SetClientID(grant.ClientID).
		SetRedirectURI(grant.RedirectURI).
		SetExpiresAt(grant.ExpiresAt).
		Save(ctx)
	if err != nil {
		// Do not expose a database constraint (or the hash) to callers/logging.
		if isUniqueConstraintViolation(err) {
			return service.ErrWorkbenchUnavailable
		}
		return err
	}
	grant.ID = created.ID
	grant.CreatedAt = created.CreatedAt
	grant.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *workbenchIntegrationRepository) ConsumeGrant(ctx context.Context, codeHash string, now time.Time) (*service.WorkbenchGrantResolution, error) {
	if codeHash == "" {
		return nil, service.ErrWorkbenchInvalidGrant
	}
	var result *service.WorkbenchGrantResolution
	err := r.withTx(ctx, func(txCtx context.Context, client *dbent.Client) error {
		affected, err := client.WorkbenchLaunchGrant.Update().
			Where(
				workbenchlaunchgrant.CodeHashEQ(codeHash),
				workbenchlaunchgrant.ConsumedAtIsNil(),
				workbenchlaunchgrant.ExpiresAtGT(now),
			).
			SetConsumedAt(now).
			Save(txCtx)
		if err != nil {
			return err
		}
		if affected != 1 {
			return service.ErrWorkbenchInvalidGrant
		}
		grant, err := client.WorkbenchLaunchGrant.Query().
			Where(workbenchlaunchgrant.CodeHashEQ(codeHash)).
			Only(txCtx)
		if err != nil {
			return err
		}

		resolution := &service.WorkbenchGrantResolution{Grant: workbenchLaunchGrantEntityToService(grant)}
		key, keyErr := client.APIKey.Query().
			Where(apikey.IDEQ(grant.APIKeyID), apikey.DeletedAtIsNil()).
			ForUpdate().
			Only(txCtx)
		if keyErr == nil {
			resolution.APIKey = apiKeyEntityToService(key)
		} else if !dbent.IsNotFound(keyErr) {
			return keyErr
		}
		owner, ownerErr := client.User.Query().Where(user.IDEQ(grant.UserID)).ForUpdate().Only(txCtx)
		if ownerErr == nil {
			resolution.User = userEntityToService(owner)
		} else if !dbent.IsNotFound(ownerErr) {
			return ownerErr
		}
		binding, bindingErr := client.WorkbenchCredential.Query().
			Where(
				workbenchcredential.UserIDEQ(grant.UserID),
				workbenchcredential.WorkbenchEQ(service.WorkbenchHeliosGen),
				workbenchcredential.APIKeyIDEQ(grant.APIKeyID),
			).
			ForUpdate().
			Only(txCtx)
		if bindingErr == nil {
			resolution.Credential = workbenchCredentialEntityToService(binding)
		} else if !dbent.IsNotFound(bindingErr) {
			return bindingErr
		}
		result = resolution
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func workbenchCredentialEntityToService(entity *dbent.WorkbenchCredential) *service.WorkbenchCredential {
	if entity == nil {
		return nil
	}
	return &service.WorkbenchCredential{
		ID: entity.ID, UserID: entity.UserID, Workbench: entity.Workbench,
		APIKeyID: entity.APIKeyID, CreatedAt: entity.CreatedAt, UpdatedAt: entity.UpdatedAt,
	}
}

func workbenchLaunchGrantEntityToService(entity *dbent.WorkbenchLaunchGrant) *service.WorkbenchLaunchGrant {
	if entity == nil {
		return nil
	}
	return &service.WorkbenchLaunchGrant{
		ID: entity.ID, CodeHash: entity.CodeHash, UserID: entity.UserID,
		APIKeyID: entity.APIKeyID, ClientID: entity.ClientID, RedirectURI: entity.RedirectURI,
		ExpiresAt: entity.ExpiresAt, ConsumedAt: entity.ConsumedAt,
		CreatedAt: entity.CreatedAt, UpdatedAt: entity.UpdatedAt,
	}
}
