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

type ChatAttachment struct{ ent.Schema }

func (ChatAttachment) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "chat_attachments"}}
}

func (ChatAttachment) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (ChatAttachment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Immutable(),
		field.Int64("user_id").Immutable(),
		field.String("storage_key").NotEmpty().MaxLen(200).Unique().Immutable(),
		field.String("original_name").NotEmpty().MaxLen(255),
		field.String("mime_type").MaxLen(128).Default(""),
		field.Int64("byte_size").NonNegative(),
		field.String("sha256").MaxLen(64).Default(""),
		field.Enum("purpose").Values("image", "rich_layout_recognition", "user_data"),
		field.Enum("status").Values("uploading", "ready", "staged", "orphan", "deleting").Default("uploading"),
		field.UUID("import_id", uuid.UUID{}).Optional().Nillable(),
	}
}

func (ChatAttachment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("chat_attachments").Field("user_id").Unique().Required().Immutable(),
		edge.To("session_refs", ChatSessionAttachment.Type),
		edge.To("uploads", ChatAttachmentUpload.Type),
	}
}

func (ChatAttachment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "id").Unique(),
		index.Fields("user_id", "status"),
		index.Fields("status", "updated_at"),
	}
}
