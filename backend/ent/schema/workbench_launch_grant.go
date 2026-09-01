package schema

import (
	"encoding/hex"
	"fmt"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WorkbenchLaunchGrant is a short-lived, one-time authorization grant.
// Only the SHA-256 digest of the opaque code is persisted.
type WorkbenchLaunchGrant struct {
	ent.Schema
}

func (WorkbenchLaunchGrant) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "workbench_launch_grants"},
	}
}

func (WorkbenchLaunchGrant) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (WorkbenchLaunchGrant) Fields() []ent.Field {
	return []ent.Field{
		field.String("code_hash").MaxLen(64).NotEmpty().Unique().Validate(func(value string) error {
			if len(value) != 64 {
				return fmt.Errorf("code_hash must be a SHA-256 hex digest")
			}
			if _, err := hex.DecodeString(value); err != nil {
				return fmt.Errorf("code_hash must be a SHA-256 hex digest: %w", err)
			}
			return nil
		}),
		field.Int64("user_id"),
		field.Int64("api_key_id"),
		field.String("client_id").MaxLen(128).NotEmpty(),
		field.String("redirect_uri").MaxLen(2048).NotEmpty(),
		field.Time("expires_at"),
		field.Time("consumed_at").Optional().Nillable(),
	}
}

func (WorkbenchLaunchGrant) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("workbench_launch_grants").
			Field("user_id").
			Unique().
			Required(),
		edge.From("api_key", APIKey.Type).
			Ref("workbench_launch_grants").
			Field("api_key_id").
			Unique().
			Required(),
	}
}

func (WorkbenchLaunchGrant) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("api_key_id"),
		index.Fields("expires_at"),
	}
}
