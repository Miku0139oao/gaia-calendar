package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type GaiaSession struct {
	ent.Schema
}

func (GaiaSession) Fields() []ent.Field {
	return []ent.Field{
		field.String("company_code").Default(""),
		field.String("encrypted_payload").Sensitive(),
		field.Time("expires_at").Optional().Nillable(),
		field.String("last_error").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (GaiaSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("gaia_sessions").Unique().Required(),
	}
}
