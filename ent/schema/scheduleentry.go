package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ScheduleEntry struct {
	ent.Schema
}

func (ScheduleEntry) Fields() []ent.Field {
	return []ent.Field{
		field.Time("shift_date"),
		field.String("shift_name").Optional().Nillable(),
		field.Time("start_time").Optional().Nillable(),
		field.Time("end_time").Optional().Nillable(),
		field.Float("hours").Optional().Nillable(),
		field.String("class_code").Optional().Nillable(),
		field.String("raw_json").Optional().Sensitive(),
		field.Time("synced_at").Default(time.Now),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (ScheduleEntry) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("schedule_entries").Unique().Required(),
	}
}

func (ScheduleEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("shift_date").Edges("user").Unique(),
	}
}
