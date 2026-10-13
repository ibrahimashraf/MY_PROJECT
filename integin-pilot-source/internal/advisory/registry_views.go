package advisory

// ApprovedModels returns a copy of every registered model whose status is
// APPROVED, ordered for stable lease. Callers receive cloned slices so the
// registry state is never mutated through the view.
func (r *Registry) ApprovedModels() []ModelRegistration {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	models := make([]ModelRegistration, 0, len(r.models))
	for _, m := range r.models {
		if m.Status != ModelApproved {
			continue
		}
		clone := m
		clone.AllowedZones = append([]Zone(nil), m.AllowedZones...)
		models = append(models, clone)
	}
	r.mu.RUnlock()
	return models
}
