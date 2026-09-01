package traininghttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"integin/internal/domain/training"
)

type mockTrainingRepo struct {
	courses     []training.Course
	enrollments []training.CourseEnrollment
	comps       []training.Competency
}

func (m *mockTrainingRepo) ListCourses(ctx context.Context, actor training.ActorContext) ([]training.Course, error) {
	return m.courses, nil
}

func (m *mockTrainingRepo) GetCourse(ctx context.Context, actor training.ActorContext, courseID string) (training.Course, error) {
	return training.Course{}, nil
}

func (m *mockTrainingRepo) CreateCourse(ctx context.Context, actor training.ActorContext, course training.Course) (training.Course, error) {
	return course, nil
}

func (m *mockTrainingRepo) EnrollTechnician(ctx context.Context, actor training.ActorContext, enrollment training.CourseEnrollment) (training.CourseEnrollment, error) {
	return enrollment, nil
}

func (m *mockTrainingRepo) ListEnrollments(ctx context.Context, actor training.ActorContext, courseID string) ([]training.CourseEnrollment, error) {
	return m.enrollments, nil
}

func (m *mockTrainingRepo) ListCompetencies(ctx context.Context, actor training.ActorContext, technicianID string) ([]training.Competency, error) {
	return m.comps, nil
}

func (m *mockTrainingRepo) GetCompetency(ctx context.Context, actor training.ActorContext, competencyID string) (training.Competency, error) {
	return training.Competency{}, nil
}

func (m *mockTrainingRepo) UpsertCompetency(ctx context.Context, actor training.ActorContext, comp training.Competency) (training.Competency, error) {
	return comp, nil
}

func TestHandleListCourses(t *testing.T) {
	repo := &mockTrainingRepo{courses: []training.Course{{ID: "c1", Title: "Test Course"}}}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/training/", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleCreateCourse(t *testing.T) {
	repo := &mockTrainingRepo{}
	h := Handler{Repository: repo}

	body := `{"title":"New Course"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/training/", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestHandleMissingHeaders(t *testing.T) {
	repo := &mockTrainingRepo{}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/training/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
