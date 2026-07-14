package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SubscriptionPurchaseClaim serializes one-time subscription purchases per user and group.
type SubscriptionPurchaseClaim struct {
	ent.Schema
}

func (SubscriptionPurchaseClaim) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "subscription_purchase_claims"},
	}
}

func (SubscriptionPurchaseClaim) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int64("subscription_group_id"),
		field.Int64("payment_order_id").
			Optional().
			Nillable(),
		field.String("status").
			MaxLen(20).
			Default("PENDING"),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (SubscriptionPurchaseClaim) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "subscription_group_id").Unique(),
		index.Fields("payment_order_id").
			Unique().
			Annotations(entsql.IndexWhere("payment_order_id IS NOT NULL")),
		index.Fields("status"),
	}
}
