package schema

import (
	"fmt"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WorkbenchCredential binds a managed workbench to the user's dedicated API key.
type WorkbenchCredential struct {
	ent.Schema
}

func (WorkbenchCredential) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "workbench_credentials"},
	}
}

func (WorkbenchCredential) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (WorkbenchCredential) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("workbench").MaxLen(64).NotEmpty().Validate(func(value string) error {
			if value != "heliosgen" {
				return fmt.Errorf("unsupported workbench %q", value)
			}
			return nil
		}),
		field.Int64("api_key_id"),
	}
}

func (WorkbenchCredential) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("workbench_credentials").
			Field("user_id").
			Unique().
			Required(),
		edge.From("api_key", APIKey.Type).
			Ref("workbench_credentials").
			Field("api_key_id").
			Unique().
			Required(),
	}
}

func (WorkbenchCredential) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "workbench").Unique(),
		index.Fields("api_key_id").Unique(),
	}
}
