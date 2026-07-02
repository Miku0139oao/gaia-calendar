package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type GaiaCredential struct {
	ent.Schema
}

func (GaiaCredential) Fields() []ent.Field {
	return []ent.Field{
		field.String("company_code").Default(""),
		field.String("employee_account").NotEmpty(),
		field.String("encrypted_password").Sensitive(),
		field.String("credential_status").Default("unchecked"),
		field.Time("last_login_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (GaiaCredential) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("gaia_credentials").Unique().Required(),
	}
}

func (GaiaCredential) Indexes() []ent.Index {
	return []ent.Index{
		index.Edges("user").Unique(),
	}
}
