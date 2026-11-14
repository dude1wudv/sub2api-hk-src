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

type ChatAttachmentParse struct{ ent.Schema }

func (ChatAttachmentParse) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "chat_attachment_parses"}}
}
func (ChatAttachmentParse) Mixin() []ent.Mixin { return []ent.Mixin{mixins.TimeMixin{}} }
func (ChatAttachmentParse) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Immutable(),
		field.Int64("user_id").Immutable(),
		field.UUID("attachment_id", uuid.UUID{}).Immutable(),
		field.Int64("account_id").Immutable(),
		field.String("purpose").MaxLen(32).Default("text_recognition").Immutable(),
		field.Enum("status").Values("processing", "ready", "failed", "unknown").Default("processing"),
		field.JSON("result", json.RawMessage{}).Optional().SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Int("page_count").NonNegative().Max(1000).Optional().Nillable(),
		field.UUID("operation_id", uuid.UUID{}),
		field.String("request_id").MaxLen(128).Default(""),
		field.JSON("billing_receipt", json.RawMessage{}).Optional().SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Bool("billing_applied").Default(false),
		field.Int64("reserved_bytes").NonNegative().Default(0),
		field.Int64("result_bytes").NonNegative().Default(0),
	}
}
func (ChatAttachmentParse) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("chat_attachment_parses").Field("user_id").Unique().Required().Immutable(),
		edge.From("attachment", ChatAttachment.Type).Ref("parses").Field("attachment_id").Unique().Required().Immutable(),
	}
}
func (ChatAttachmentParse) Indexes() []ent.Index {
	return []ent.Index{index.Fields("attachment_id", "account_id", "purpose").Unique(), index.Fields("user_id", "status")}
}
