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

type ChatSessionAttachment struct{ ent.Schema }

func (ChatSessionAttachment) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "chat_session_attachments"}}
}

func (ChatSessionAttachment) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (ChatSessionAttachment) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").Immutable(),
		field.UUID("session_id", uuid.UUID{}).Immutable(),
		field.UUID("attachment_id", uuid.UUID{}).Immutable(),
	}
}

func (ChatSessionAttachment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("chat_session_attachments").Field("user_id").Unique().Required().Immutable(),
		edge.From("session", ChatSession.Type).Ref("attachment_refs").Field("session_id").Unique().Required().Immutable(),
		edge.From("attachment", ChatAttachment.Type).Ref("session_refs").Field("attachment_id").Unique().Required().Immutable(),
	}
}

func (ChatSessionAttachment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("session_id", "attachment_id").Unique(),
		index.Fields("attachment_id"),
	}
}
