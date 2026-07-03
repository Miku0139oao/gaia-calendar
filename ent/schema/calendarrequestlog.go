package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type CalendarRequestLog struct {
	ent.Schema
}

func (CalendarRequestLog) Fields() []ent.Field {
	return []ent.Field{
		field.Time("requested_at").Default(time.Now),
		field.String("user_agent").Default(""),
		field.String("remote_addr").Default(""),
		field.String("path").Default(""),
	}
}

func (CalendarRequestLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("subscription", CalendarSubscription.Type).Ref("request_logs").Unique().Required(),
	}
}

func (CalendarRequestLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("requested_at"),
		index.Edges("subscription"),
	}
}
