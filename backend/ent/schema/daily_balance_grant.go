package schema

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DailyBalanceGrant holds the schema definition for the DailyBalanceGrant entity.
//
// 每日余额赠额：一次充值生成一笔带 24h 有效期的独立额度桶，仅能在其绑定的
// 专属分组（groups.daily_balance_enabled = true）内消费。区别于 users.balance
// 单一长期余额池——每日额度是多笔、各自带过期时间、需按到期时间 FIFO 消耗的对象。
//
// 删除策略：硬删除（过期/耗尽后通过 status 标记，无需软删除）。
type DailyBalanceGrant struct {
	ent.Schema
}

func (DailyBalanceGrant) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "daily_balance_grants"},
	}
}

func (DailyBalanceGrant) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int64("group_id"),
		field.Float("amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0).
			Comment("充值原始金额"),
		field.Float("remaining").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0).
			Comment("剩余可用，原子递减至 0"),
		field.String("status").
			MaxLen(20).
			Default(domain.DailyGrantStatusActive).
			Comment("active / exhausted / expired"),
		field.String("source").
			MaxLen(20).
			Default(domain.DailyGrantSourceAdmin).
			Comment("admin / redeem"),
		field.String("source_ref").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Comment("兑换码 code 或操作者标识，便于审计"),
		field.Time("granted_at").
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("expires_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("granted_at + 24h"),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (DailyBalanceGrant) Indexes() []ent.Index {
	return []ent.Index{
		// 热查询：取某用户某专属分组的有效 Grant，按 expires_at 升序消耗
		index.Fields("user_id", "group_id", "status", "expires_at"),
		index.Fields("status", "expires_at"),
		index.Fields("expires_at"),
	}
}
