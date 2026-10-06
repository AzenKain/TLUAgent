package models

import (
	"time"

	"tluagent-web/internal/gen/sqlc"
)

// StudentProfile represents a persistent student academic standing record.
type StudentProfile struct {
	UserID           string    `json:"user_id"`
	Major            string    `json:"major"`
	Cohort           string    `json:"cohort"`
	AcademicStanding string    `json:"academic_standing"`
	CompletedCredits int       `json:"completed_credits"`
	CumulativeGPA    float64   `json:"cumulative_gpa"`
	TargetGPA        float64   `json:"target_gpa"`
	AdvisorNotes     string    `json:"advisor_notes"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// FromSqlc maps a sqlc.StudentAcademicProfile row to a domain model.
func (p *StudentProfile) FromSqlc(row sqlc.StudentAcademicProfile) *StudentProfile {
	p.UserID = row.UserID
	p.Major = row.Major
	p.Cohort = row.Cohort
	p.AcademicStanding = row.AcademicStanding
	p.CompletedCredits = int(row.CompletedCredits)
	p.CumulativeGPA = row.CumulativeGpa
	p.TargetGPA = row.TargetGpa
	p.AdvisorNotes = row.AdvisorNotes
	p.UpdatedAt = row.UpdatedAt
	return p
}

// ToSqlcUpsertParams converts the profile into sqlc upsert parameters.
func (p *StudentProfile) ToSqlcUpsertParams() sqlc.UpsertStudentProfileParams {
	now := time.Now().UTC()
	return sqlc.UpsertStudentProfileParams{
		UserID:           p.UserID,
		Major:            p.Major,
		Cohort:           p.Cohort,
		AcademicStanding: p.AcademicStanding,
		CompletedCredits: int64(p.CompletedCredits),
		CumulativeGpa:    p.CumulativeGPA,
		TargetGpa:        p.TargetGPA,
		AdvisorNotes:     p.AdvisorNotes,
		UpdatedAt:        now,
	}
}

// UserMemoryItem represents an extracted or declared long-term student memory fact.
type UserMemoryItem struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Category    string    `json:"category"`
	MemoryKey   string    `json:"memory_key"`
	MemoryValue string    `json:"memory_value"`
	Confidence  float64   `json:"confidence"`
	Source      string    `json:"source"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FromSqlc maps a sqlc.UserMemory row to a domain model.
func (m *UserMemoryItem) FromSqlc(row sqlc.UserMemory) *UserMemoryItem {
	m.ID = row.ID
	m.UserID = row.UserID
	m.Category = row.Category
	m.MemoryKey = row.MemoryKey
	m.MemoryValue = row.MemoryValue
	m.Confidence = row.Confidence
	m.Source = row.Source
	m.IsActive = row.IsActive == 1
	m.CreatedAt = row.CreatedAt
	m.UpdatedAt = row.UpdatedAt
	return m
}

// ToSqlcUpsertParams converts the memory item into sqlc upsert parameters.
func (m *UserMemoryItem) ToSqlcUpsertParams() sqlc.UpsertUserMemoryParams {
	now := time.Now().UTC()
	activeInt := int64(1)
	conf := m.Confidence
	if conf <= 0 {
		conf = 1.0
	}
	return sqlc.UpsertUserMemoryParams{
		ID:          m.ID,
		UserID:      m.UserID,
		Category:    m.Category,
		MemoryKey:   m.MemoryKey,
		MemoryValue: m.MemoryValue,
		Confidence:  conf,
		Source:      m.Source,
		IsActive:    activeInt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
