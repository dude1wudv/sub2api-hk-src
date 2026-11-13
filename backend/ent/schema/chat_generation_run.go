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

type ChatGenerationRun struct{ ent.Schema }

func (ChatGenerationRun) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "chat_generation_runs"}}
}

func (ChatGenerationRun) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (ChatGenerationRun) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Immutable(),
		field.Int64("user_id").Immutable(),
		field.UUID("session_id", uuid.UUID{}).Immutable(),
		field.String("assistant_message_id").NotEmpty().MaxLen(200).Immutable(),
		field.UUID("preset_id", uuid.UUID{}).Immutable(),
		field.Int64("group_id").Immutable(),
		field.String("model").NotEmpty().MaxLen(256).Immutable(),
		field.JSON("options", json.RawMessage{}).SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.UUID("request_id", uuid.UUID{}).Immutable(),
		field.Enum("operation").Values("send", "regenerate", "continue").Immutable(),
		field.Enum("status").Values("running", "done", "stopped", "error").Default("running"),
		field.UUID("owner_instance_id", uuid.UUID{}).Immutable(),
		field.Time("lease_expires_at").SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("started_at").SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("heartbeat_at").SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("ended_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.JSON("gateway_request_ids", []string{}).Default([]string{}),
	}
}

func (ChatGenerationRun) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("chat_generation_runs").Field("user_id").Unique().Required().Immutable(),
		edge.From("session", ChatSession.Type).Ref("generation_runs").Field("session_id").Unique().Required().Immutable(),
	}
}

func (ChatGenerationRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "request_id").Unique(),
		index.Fields("session_id").Unique().StorageKey("chat_runs_one_active").Annotations(entsql.IndexWhere("status = 'running'")),
		index.Fields("status", "lease_expires_at"),
	}
}
