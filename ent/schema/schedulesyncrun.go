package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type ScheduleSyncRun struct {
	ent.Schema
}

func (ScheduleSyncRun) Fields() []ent.Field {
	return []ent.Field{
		field.Time("started_at").Default(time.Now),
		field.Time("finished_at").Optional().Nillable(),
		field.String("status").Default("running"),
		field.String("error_message").Optional().Nillable(),
		field.Time("range_start"),
		field.Time("range_end"),
		field.Int("entry_count").Default(0),
		field.Time("marked_at").Optional().Nillable(),
	}
}

func (ScheduleSyncRun) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("sync_runs").Unique().Required(),
	}
}
