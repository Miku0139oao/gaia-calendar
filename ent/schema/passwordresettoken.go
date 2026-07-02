package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type PasswordResetToken struct {
	ent.Schema
}

func (PasswordResetToken) Fields() []ent.Field {
	return []ent.Field{
		field.String("email").NotEmpty(),
		field.String("token_hash").Unique().Sensitive(),
		field.Time("expires_at"),
		field.Time("consumed_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
	}
}

func (PasswordResetToken) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("password_reset_tokens").Unique().Required(),
	}
}
