package schema

import (
	"encoding/json"
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

type ChatPreference struct{ ent.Schema }

func (ChatPreference) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "chat_preferences"}}
}

func (ChatPreference) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (ChatPreference) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").Unique().Immutable(),
		field.JSON("preferences", json.RawMessage{}).SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Int64("history_revision").NonNegative().Default(0),
	}
}

func (ChatPreference) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("chat_preferences").Field("user_id").Unique().Required().Immutable(),
	}
}
