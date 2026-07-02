package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type AppSession struct {
	ent.Schema
}

func (AppSession) Fields() []ent.Field {
	return []ent.Field{
		field.String("token_hash").Unique().Sensitive(),
		field.Time("expires_at"),
		field.Time("created_at").Default(time.Now),
		field.Time("revoked_at").Optional().Nillable(),
	}
}

func (AppSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("sessions").Unique().Required(),
	}
}
