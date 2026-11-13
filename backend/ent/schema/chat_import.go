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

type ChatImport struct{ ent.Schema }

func (ChatImport) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "chat_imports"}}
}

func (ChatImport) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (ChatImport) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Immutable(),
		field.Int64("user_id").Immutable(),
		field.Enum("status").Values("pending", "committed", "failed").Default("pending"),
		field.Time("expires_at").SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Int64("history_revision").NonNegative(),
		field.JSON("manifest", json.RawMessage{}).SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
	}
}

func (ChatImport) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("chat_imports").Field("user_id").Unique().Required().Immutable(),
	}
}

func (ChatImport) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "id").Unique(),
		index.Fields("status", "expires_at"),
	}
}
