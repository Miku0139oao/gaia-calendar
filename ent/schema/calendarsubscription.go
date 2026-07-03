package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type CalendarSubscription struct {
	ent.Schema
}

func (CalendarSubscription) Fields() []ent.Field {
	return []ent.Field{
		field.String("token_hash").Unique().Sensitive(),
		field.String("encrypted_token").Sensitive(),
		field.Bool("enabled").Default(true),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (CalendarSubscription) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("calendar_subscriptions").Unique().Required(),
		edge.To("request_logs", CalendarRequestLog.Type),
	}
}

func (CalendarSubscription) Indexes() []ent.Index {
	return []ent.Index{
		index.Edges("user").Unique(),
	}
}
