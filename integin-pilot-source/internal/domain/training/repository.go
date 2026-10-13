package training

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidIdentity = errors.New("training identity is incomplete")
)

type ActorContext struct {
	TenantID       string
	OrganizationID string
	ActorID        string
}

func (a ActorContext) Validate() error {
	if strings.TrimSpace(a.TenantID) == "" || strings.TrimSpace(a.OrganizationID) == "" || strings.TrimSpace(a.ActorID) == "" {
		return ErrInvalidIdentity
	}
	return nil
}

type Repository interface {
	ListCourses(ctx context.Context, actor ActorContext) ([]Course, error)
	GetCourse(ctx context.Context, actor ActorContext, courseID string) (Course, error)
	CreateCourse(ctx context.Context, actor ActorContext, course Course) (Course, error)
	EnrollTechnician(ctx context.Context, actor ActorContext, enrollment CourseEnrollment) (CourseEnrollment, error)
	ListEnrollments(ctx context.Context, actor ActorContext, courseID string) ([]CourseEnrollment, error)
	ListCompetencies(ctx context.Context, actor ActorContext, technicianID string) ([]Competency, error)
	GetCompetency(ctx context.Context, actor ActorContext, competencyID string) (Competency, error)
	UpsertCompetency(ctx context.Context, actor ActorContext, comp Competency) (Competency, error)
}

type TransactionRunner interface {
	WithinTransaction(ctx context.Context, actor ActorContext, fn func(context.Context, Repository) error) error
}

type ServiceDependencies struct {
	Repository   Repository
	Transactions TransactionRunner
}

func (d ServiceDependencies) Validate() error {
	if d.Repository == nil || d.Transactions == nil {
		return ErrInvalidIdentity
	}
	return nil
}
