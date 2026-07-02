package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type EmailVerificationCode struct {
	ent.Schema
}

func (EmailVerificationCode) Fields() []ent.Field {
	return []ent.Field{
		field.String("email").NotEmpty(),
		field.String("code_hash").Sensitive(),
		field.Time("expires_at"),
		field.Time("consumed_at").Optional().Nillable(),
		field.Int("attempt_count").Default(0),
		field.Time("created_at").Default(time.Now),
	}
}

func (EmailVerificationCode) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("verification_codes").Unique(),
	}
}
