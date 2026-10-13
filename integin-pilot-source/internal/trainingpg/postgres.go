package trainingpg

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"integin/internal/domain/training"
	"integin/internal/shared/pgtx"
)

var ErrNilDB = errors.New("training postgres repository requires a database")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

func (r *Repository) begin(ctx context.Context, actor training.ActorContext) (*sql.Tx, error) {
	return pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
}

func (r *Repository) ListCourses(ctx context.Context, actor training.ActorContext) ([]training.Course, error) {
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx,
		`SELECT id, tenant_id, organization_id, title, instructor_id, status, created_at
		 FROM course
		 WHERE tenant_id = current_setting('integin.tenant_id', true)
		   AND organization_id = current_setting('integin.organization_id', true)
		 ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []training.Course
	for rows.Next() {
		var c training.Course
		var instructorID sql.NullString
		if err := rows.Scan(&c.ID, &c.TenantID, &c.OrganizationID, &c.Title, &instructorID, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		if instructorID.Valid {
			c.InstructorID = instructorID.String
		}
		courses = append(courses, c)
	}
	return courses, tx.Commit()
}

func (r *Repository) GetCourse(ctx context.Context, actor training.ActorContext, courseID string) (training.Course, error) {
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return training.Course{}, err
	}
	defer tx.Rollback()

	var c training.Course
	var instructorID sql.NullString
	err = tx.QueryRowContext(ctx,
		`SELECT id, tenant_id, organization_id, title, instructor_id, status, created_at
		 FROM course
		 WHERE id = $1 AND tenant_id = current_setting('integin.tenant_id', true)
		   AND organization_id = current_setting('integin.organization_id', true)`,
		courseID,
	).Scan(&c.ID, &c.TenantID, &c.OrganizationID, &c.Title, &instructorID, &c.Status, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return training.Course{}, training.ErrCourseNotFound
	}
	if err != nil {
		return training.Course{}, err
	}
	if instructorID.Valid {
		c.InstructorID = instructorID.String
	}
	return c, tx.Commit()
}

func (r *Repository) CreateCourse(ctx context.Context, actor training.ActorContext, course training.Course) (training.Course, error) {
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return training.Course{}, err
	}
	defer tx.Rollback()

	now := time.Now()
	if course.ID == "" {
		course.ID = course.TenantID + ":course:" + now.Format("20060102150405")
	}
	course.CreatedAt = now
	if course.Status == "" {
		course.Status = training.CourseDraft
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO course (id, tenant_id, organization_id, title, instructor_id, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		course.ID, course.TenantID, course.OrganizationID, course.Title,
		course.InstructorID, string(course.Status), course.CreatedAt,
	)
	if err != nil {
		return training.Course{}, err
	}
	return course, tx.Commit()
}

func (r *Repository) EnrollTechnician(ctx context.Context, actor training.ActorContext, enrollment training.CourseEnrollment) (training.CourseEnrollment, error) {
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return training.CourseEnrollment{}, err
	}
	defer tx.Rollback()

	now := time.Now()
	if enrollment.ID == "" {
		enrollment.ID = enrollment.TenantID + ":enroll:" + now.Format("20060102150405")
	}
	enrollment.EnrolledAt = now

	_, err = tx.ExecContext(ctx,
		`INSERT INTO course_enrollment (id, tenant_id, organization_id, course_id, technician_id, enrolled_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		enrollment.ID, enrollment.TenantID, enrollment.OrganizationID,
		enrollment.CourseID, enrollment.TechnicianID, enrollment.EnrolledAt,
	)
	if err != nil {
		return training.CourseEnrollment{}, err
	}
	return enrollment, tx.Commit()
}

func (r *Repository) ListEnrollments(ctx context.Context, actor training.ActorContext, courseID string) ([]training.CourseEnrollment, error) {
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx,
		`SELECT id, tenant_id, organization_id, course_id, technician_id, enrolled_at, completed_at
		 FROM course_enrollment
		 WHERE course_id = $1 AND tenant_id = current_setting('integin.tenant_id', true)
		   AND organization_id = current_setting('integin.organization_id', true)`,
		courseID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var enrollments []training.CourseEnrollment
	for rows.Next() {
		var e training.CourseEnrollment
		var completedAt sql.NullTime
		if err := rows.Scan(&e.ID, &e.TenantID, &e.OrganizationID, &e.CourseID, &e.TechnicianID, &e.EnrolledAt, &completedAt); err != nil {
			return nil, err
		}
		if completedAt.Valid {
			e.CompletedAt = &completedAt.Time
		}
		enrollments = append(enrollments, e)
	}
	return enrollments, tx.Commit()
}

func (r *Repository) ListCompetencies(ctx context.Context, actor training.ActorContext, technicianID string) ([]training.Competency, error) {
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx,
		`SELECT id, tenant_id, organization_id, technician_id, equipment_type_id,
		        certification_ref, expires_at, status, verified_by, created_at
		 FROM technician_competency
		 WHERE technician_id = $1 AND tenant_id = current_setting('integin.tenant_id', true)
		   AND organization_id = current_setting('integin.organization_id', true)`,
		technicianID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comps []training.Competency
	for rows.Next() {
		var c training.Competency
		var certRef, verifiedBy sql.NullString
		var expiresAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.TenantID, &c.OrganizationID, &c.TechnicianID,
			&c.EquipmentTypeID, &certRef, &expiresAt, &c.Status, &verifiedBy, &c.CreatedAt); err != nil {
			return nil, err
		}
		if certRef.Valid {
			c.CertificationRef = certRef.String
		}
		if expiresAt.Valid {
			c.ExpiresAt = &expiresAt.Time
		}
		if verifiedBy.Valid {
			c.VerifiedBy = verifiedBy.String
		}
		comps = append(comps, c)
	}
	return comps, tx.Commit()
}

func (r *Repository) GetCompetency(ctx context.Context, actor training.ActorContext, competencyID string) (training.Competency, error) {
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return training.Competency{}, err
	}
	defer tx.Rollback()

	var c training.Competency
	var certRef, verifiedBy sql.NullString
	var expiresAt sql.NullTime
	err = tx.QueryRowContext(ctx,
		`SELECT id, tenant_id, organization_id, technician_id, equipment_type_id,
		        certification_ref, expires_at, status, verified_by, created_at
		 FROM technician_competency
		 WHERE id = $1 AND tenant_id = current_setting('integin.tenant_id', true)
		   AND organization_id = current_setting('integin.organization_id', true)`,
		competencyID,
	).Scan(&c.ID, &c.TenantID, &c.OrganizationID, &c.TechnicianID,
		&c.EquipmentTypeID, &certRef, &expiresAt, &c.Status, &verifiedBy, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return training.Competency{}, training.ErrCompetencyNotFound
	}
	if err != nil {
		return training.Competency{}, err
	}
	if certRef.Valid {
		c.CertificationRef = certRef.String
	}
	if expiresAt.Valid {
		c.ExpiresAt = &expiresAt.Time
	}
	if verifiedBy.Valid {
		c.VerifiedBy = verifiedBy.String
	}
	return c, tx.Commit()
}

func (r *Repository) UpsertCompetency(ctx context.Context, actor training.ActorContext, comp training.Competency) (training.Competency, error) {
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return training.Competency{}, err
	}
	defer tx.Rollback()

	now := time.Now()
	if comp.ID == "" {
		comp.ID = comp.TenantID + ":comp:" + now.Format("20060102150405")
	}
	comp.CreatedAt = now
	if comp.Status == "" {
		comp.Status = training.CompetencyCurrent
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO technician_competency (id, tenant_id, organization_id, technician_id, equipment_type_id,
		        certification_ref, expires_at, status, verified_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (tenant_id, organization_id, technician_id, equipment_type_id)
		 DO UPDATE SET certification_ref = EXCLUDED.certification_ref, expires_at = EXCLUDED.expires_at,
		               status = EXCLUDED.status, verified_by = EXCLUDED.verified_by`,
		comp.ID, comp.TenantID, comp.OrganizationID, comp.TechnicianID, comp.EquipmentTypeID,
		comp.CertificationRef, comp.ExpiresAt, string(comp.Status), comp.VerifiedBy, comp.CreatedAt,
	)
	if err != nil {
		return training.Competency{}, err
	}
	return comp, tx.Commit()
}

func (r *Repository) WithinTransaction(ctx context.Context, actor training.ActorContext, fn func(context.Context, training.Repository) error) error {
	if r == nil {
		return ErrNilDB
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(ctx, r); err != nil {
		return err
	}
	return tx.Commit()
}
