package schema

import (
	"encoding/json"
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	"github.com/google/uuid"
)

type ChatSession struct{ ent.Schema }

func (ChatSession) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "chat_sessions"}}
}

func (ChatSession) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (ChatSession) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Immutable(),
		field.Int64("user_id").Immutable(),
		field.String("title").MaxRuneLen(500).Default(""),
		field.Bool("title_manual").Default(false),
		field.JSON("snapshot", json.RawMessage{}).SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Int64("revision").NonNegative().Default(0),
	}
}

func (ChatSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("chat_sessions").Field("user_id").Unique().Required().Immutable(),
		edge.To("generation_runs", ChatGenerationRun.Type),
		edge.To("attachment_refs", ChatSessionAttachment.Type),
	}
}

func (ChatSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "id").Unique(),
		index.Fields("user_id", "updated_at", "id"),
	}
}
