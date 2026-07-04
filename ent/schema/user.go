package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("email").Unique().NotEmpty(),
		field.String("nickname").Optional().Nillable().Unique(),
		field.String("password_hash").Sensitive(),
		field.Bool("email_verified").Default(false),
		field.String("role").Default("user"),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("last_login_at").Optional().Nillable(),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("verification_codes", EmailVerificationCode.Type),
		edge.To("password_reset_tokens", PasswordResetToken.Type),
		edge.To("sessions", AppSession.Type),
		edge.To("gaia_credentials", GaiaCredential.Type),
		edge.To("gaia_sessions", GaiaSession.Type),
		edge.To("schedule_entries", ScheduleEntry.Type),
		edge.To("sync_runs", ScheduleSyncRun.Type),
		edge.To("calendar_subscriptions", CalendarSubscription.Type),
	}
}
