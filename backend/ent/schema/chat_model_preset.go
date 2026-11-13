package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	"github.com/google/uuid"
)

type ChatModelPreset struct{ ent.Schema }

func (ChatModelPreset) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "chat_model_presets"}}
}

func (ChatModelPreset) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (ChatModelPreset) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Immutable(),
		field.Int64("user_id").Immutable(),
		field.Int64("group_id"),
		field.String("model").NotEmpty().MaxLen(256),
		field.String("display_name").MaxLen(200).Default(""),
		field.Int("sort_order").Default(0),
		field.Bool("is_default").Default(false),
	}
}

func (ChatModelPreset) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("chat_model_presets").Field("user_id").Unique().Required().Immutable(),
		edge.From("group", Group.Type).Ref("chat_model_presets").Field("group_id").Unique().Required(),
	}
}

func (ChatModelPreset) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "group_id", "model").Unique(),
		index.Fields("user_id").Unique().StorageKey("chat_presets_one_default").Annotations(entsql.IndexWhere("is_default")),
		index.Fields("user_id", "sort_order"),
	}
}
