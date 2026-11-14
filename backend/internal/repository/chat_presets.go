package repository

import (
	"context"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/chatmodelpreset"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

func NewChatPresetRepository(client *dbent.Client) service.ChatPresetRepository {
	return &chatRepository{client: client}
}

func chatPresetRecord(row *dbent.ChatModelPreset) *service.ChatModelPreset {
	return &service.ChatModelPreset{ID: row.ID, UserID: row.UserID, GroupID: row.GroupID, Model: row.Model, DisplayName: row.DisplayName, SortOrder: row.SortOrder, IsDefault: row.IsDefault, NormalReasoningEffort: row.NormalReasoningEffort, ThinkingReasoningEffort: row.ThinkingReasoningEffort}
}

func (r *chatRepository) ListPresets(ctx context.Context, userID int64) ([]service.ChatModelPreset, error) {
	rows, err := clientFromContext(ctx, r.client).ChatModelPreset.Query().Where(chatmodelpreset.UserIDEQ(userID)).Order(dbent.Asc(chatmodelpreset.FieldSortOrder), dbent.Asc(chatmodelpreset.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]service.ChatModelPreset, 0, len(rows))
	for _, row := range rows {
		result = append(result, *chatPresetRecord(row))
	}
	return result, nil
}

func (r *chatRepository) GetPreset(ctx context.Context, userID int64, id uuid.UUID) (*service.ChatModelPreset, error) {
	row, err := clientFromContext(ctx, r.client).ChatModelPreset.Query().Where(chatmodelpreset.UserIDEQ(userID), chatmodelpreset.IDEQ(id)).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrChatNotFound, nil)
	}
	return chatPresetRecord(row), nil
}

func (r *chatRepository) CreatePreset(ctx context.Context, userID, groupID int64, model, name string) (*service.ChatModelPreset, error) {
	var result *service.ChatModelPreset
	err := r.withUser(ctx, userID, func(ctx context.Context, client *dbent.Client) error {
		row, err := client.ChatModelPreset.Query().Where(chatmodelpreset.UserIDEQ(userID), chatmodelpreset.GroupIDEQ(groupID), chatmodelpreset.ModelEQ(model)).Only(ctx)
		if err == nil {
			result = chatPresetRecord(row)
			return nil
		}
		if !dbent.IsNotFound(err) {
			return err
		}
		last, err := client.ChatModelPreset.Query().Where(chatmodelpreset.UserIDEQ(userID)).Order(dbent.Desc(chatmodelpreset.FieldSortOrder)).First(ctx)
		if err != nil && !dbent.IsNotFound(err) {
			return err
		}
		order := 0
		if last != nil {
			order = last.SortOrder + 1
		}
		row, err = client.ChatModelPreset.Create().SetUserID(userID).SetGroupID(groupID).SetModel(model).SetDisplayName(name).SetSortOrder(order).SetIsDefault(last == nil).Save(ctx)
		if err != nil {
			return err
		}
		result = chatPresetRecord(row)
		return nil
	})
	return result, err
}

func (r *chatRepository) UpdatePreset(ctx context.Context, userID int64, id uuid.UUID, patch service.ChatPresetPatch) (*service.ChatModelPreset, error) {
	var result *service.ChatModelPreset
	err := r.withUser(ctx, userID, func(ctx context.Context, client *dbent.Client) error {
		row, err := client.ChatModelPreset.Query().Where(chatmodelpreset.UserIDEQ(userID), chatmodelpreset.IDEQ(id)).Only(ctx)
		if err != nil {
			return translatePersistenceError(err, service.ErrChatNotFound, nil)
		}
		groupID, model := row.GroupID, row.Model
		if patch.GroupID != nil {
			groupID = *patch.GroupID
		}
		if patch.Model != nil {
			model = *patch.Model
		}
		duplicate, err := client.ChatModelPreset.Query().Where(chatmodelpreset.UserIDEQ(userID), chatmodelpreset.GroupIDEQ(groupID), chatmodelpreset.ModelEQ(model), chatmodelpreset.IDNEQ(id)).Exist(ctx)
		if err != nil {
			return err
		}
		if duplicate {
			return service.ErrChatPresetDuplicate
		}
		if patch.IsDefault != nil && *patch.IsDefault {
			if _, err = client.ChatModelPreset.Update().Where(chatmodelpreset.UserIDEQ(userID), chatmodelpreset.IsDefaultEQ(true)).SetIsDefault(false).Save(ctx); err != nil {
				return err
			}
		}
		update := client.ChatModelPreset.Update().Where(chatmodelpreset.UserIDEQ(userID), chatmodelpreset.IDEQ(id))
		if patch.GroupID != nil {
			update.SetGroupID(groupID)
		}
		if patch.Model != nil {
			update.SetModel(model)
		}
		if patch.DisplayName != nil {
			update.SetDisplayName(*patch.DisplayName)
		}
		if patch.IsDefault != nil {
			update.SetIsDefault(*patch.IsDefault)
		}
		if patch.NormalReasoningEffort != nil {
			update.SetNormalReasoningEffort(*patch.NormalReasoningEffort)
		}
		if patch.ThinkingReasoningEffort != nil {
			update.SetThinkingReasoningEffort(*patch.ThinkingReasoningEffort)
		}
		if _, err = update.Save(ctx); err != nil {
			return err
		}
		row, err = client.ChatModelPreset.Query().Where(chatmodelpreset.UserIDEQ(userID), chatmodelpreset.IDEQ(id)).Only(ctx)
		if err != nil {
			return err
		}
		result = chatPresetRecord(row)
		return nil
	})
	return result, err
}

func (r *chatRepository) OrderPresets(ctx context.Context, userID int64, ids []uuid.UUID) error {
	return r.withUser(ctx, userID, func(ctx context.Context, client *dbent.Client) error {
		owned, err := client.ChatModelPreset.Query().Where(chatmodelpreset.UserIDEQ(userID)).IDs(ctx)
		if err != nil {
			return err
		}
		if len(owned) != len(ids) {
			return service.ErrChatInvalid
		}
		remaining := make(map[uuid.UUID]bool, len(owned))
		for _, id := range owned {
			remaining[id] = true
		}
		for _, id := range ids {
			if !remaining[id] {
				return service.ErrChatInvalid
			}
			delete(remaining, id)
		}
		for order, id := range ids {
			if _, err := client.ChatModelPreset.Update().Where(chatmodelpreset.UserIDEQ(userID), chatmodelpreset.IDEQ(id)).SetSortOrder(order).Save(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *chatRepository) DeletePreset(ctx context.Context, userID int64, id uuid.UUID) error {
	return r.withUser(ctx, userID, func(ctx context.Context, client *dbent.Client) error {
		_, err := client.ChatModelPreset.Delete().Where(chatmodelpreset.UserIDEQ(userID), chatmodelpreset.IDEQ(id)).Exec(ctx)
		return err
	})
}
