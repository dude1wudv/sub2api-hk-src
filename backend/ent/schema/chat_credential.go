package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

type ChatCredential struct{ ent.Schema }

func (ChatCredential) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "chat_credentials"}}
}

func (ChatCredential) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (ChatCredential) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").Immutable(),
		field.Int64("group_id").Immutable(),
		field.Int64("api_key_id"),
	}
}

func (ChatCredential) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("chat_credentials").Field("user_id").Unique().Required().Immutable(),
		edge.From("group", Group.Type).Ref("chat_credentials").Field("group_id").Unique().Required().Immutable(),
		edge.From("api_key", APIKey.Type).Ref("chat_credentials").Field("api_key_id").Unique().Required(),
	}
}

func (ChatCredential) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "group_id").Unique(),
		index.Fields("api_key_id").Unique(),
	}
}
