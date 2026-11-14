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

type ChatAttachmentUpload struct{ ent.Schema }

func (ChatAttachmentUpload) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "chat_attachment_uploads"}}
}

func (ChatAttachmentUpload) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (ChatAttachmentUpload) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").Immutable(),
		field.UUID("attachment_id", uuid.UUID{}).Immutable(),
		field.Int64("account_id").Immutable(),
		field.String("remote_file_id").MaxLen(512).Default(""),
		field.Enum("status").Values("uploading", "ready", "invalid", "deleting", "error").Default("uploading"),
		field.Int("cleanup_attempts").NonNegative().Default(0),
		field.UUID("lease_token", uuid.UUID{}).Optional().Nillable(),
		field.Time("lease_expires_at").Optional().Nillable(),
	}
}

func (ChatAttachmentUpload) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("chat_attachment_uploads").Field("user_id").Unique().Required().Immutable(),
		edge.From("attachment", ChatAttachment.Type).Ref("uploads").Field("attachment_id").Unique().Required().Immutable(),
	}
}

func (ChatAttachmentUpload) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("attachment_id", "account_id").Unique(),
		index.Fields("user_id", "status"),
	}
}
