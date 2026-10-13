package main

import (
	"encoding/json"
	"log"
	"os"
)

func main() {
	doc := loadSpec("openapi/integin-v1.json")
	paths := getPaths(doc)
	removeOldStubs(paths)
	addLicensePaths(paths)
	addFeatureFlagPaths(paths)
	addTrainingPaths(paths)
	addSettingsPaths(paths)
	addInspectionPaths(paths)
	addSearchPaths(paths)
	addAuditLogPaths(paths)
	addAnalyticsPaths(paths)
	addReportsPaths(paths)
	addComponentSchemas(doc)
	addParameters(doc)
	updateTags(doc)
	saveSpec("openapi/integin-v1.json", doc)
	log.Println("OpenAPI spec updated with proper schemas")
}

func loadSpec(path string) map[string]interface{} {
	//nolint:gosec // path is CLI arg for OpenAPI fix tool
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read spec: %v", err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		log.Fatalf("unmarshal spec: %v", err)
	}
	return doc
}

func saveSpec(path string, doc map[string]interface{}) {
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		log.Fatalf("marshal spec: %v", err)
	}
	if err := os.WriteFile(path, out, 0600); err != nil {
		log.Fatalf("write spec: %v", err)
	}
}

func getPaths(doc map[string]interface{}) map[string]interface{} {
	return doc["paths"].(map[string]interface{})
}

func removeOldStubs(paths map[string]interface{}) {
	for _, p := range []string{
		"/api/v1/licenses/", "/api/v1/admin/feature-flags", "/api/v1/admin/feature-flags/",
		"/api/v1/training/", "/api/v1/admin/settings", "/api/v1/admin/settings/",
		"/api/v1/inspections", "/api/v1/search/", "/api/v1/audit-log", "/api/v1/audit-log/",
		"/api/v1/analytics/", "/api/v1/reports/",
	} {
		delete(paths, p)
	}
}

func tenantParams() []interface{} {
	return []interface{}{
		map[string]interface{}{"$ref": "#/components/parameters/TenantID"},
		map[string]interface{}{"$ref": "#/components/parameters/OrganizationID"},
	}
}

func addLicensePaths(paths map[string]interface{}) {
	paths["/api/v1/licenses/"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags": []interface{}{"License"}, "operationId": "listLicenses",
			"summary": "List license entitlements for the tenant", "parameters": tenantParams(),
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "License list",
					"content":     map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"type": "array", "items": map[string]interface{}{"$ref": "#/components/schemas/License"}}}},
				},
				"400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"},
				"405": map[string]interface{}{"$ref": "#/components/responses/MethodNotAllowed"},
			},
		},
		"post": map[string]interface{}{
			"tags": []interface{}{"License"}, "operationId": "issueLicense",
			"summary": "Issue a new license entitlement", "parameters": tenantParams(),
			"requestBody": map[string]interface{}{"required": true, "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/LicenseIssueRequest"}}}},
			"responses": map[string]interface{}{
				"201": map[string]interface{}{"description": "License issued", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/License"}}}},
				"400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"},
				"405": map[string]interface{}{"$ref": "#/components/responses/MethodNotAllowed"},
			},
		},
	}
}

func addFeatureFlagPaths(paths map[string]interface{}) {
	paths["/api/v1/admin/feature-flags"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags": []interface{}{"Feature flags"}, "operationId": "listFeatureFlags",
			"summary": "List feature flag overrides for the tenant", "parameters": tenantParams(),
			"responses": map[string]interface{}{
				"200": map[string]interface{}{"description": "Feature flag list", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"type": "array", "items": map[string]interface{}{"$ref": "#/components/schemas/FeatureFlag"}}}}},
				"400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"},
				"405": map[string]interface{}{"$ref": "#/components/responses/MethodNotAllowed"},
			},
		},
		"put": map[string]interface{}{
			"tags": []interface{}{"Feature flags"}, "operationId": "upsertFeatureFlag",
			"summary": "Create or update a feature flag override", "parameters": tenantParams(),
			"requestBody": map[string]interface{}{"required": true, "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/FeatureFlagUpsert"}}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{"description": "Feature flag upserted", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/FeatureFlag"}}}},
				"400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"},
				"405": map[string]interface{}{"$ref": "#/components/responses/MethodNotAllowed"},
			},
		},
	}
	paths["/api/v1/admin/feature-flags/"] = map[string]interface{}{
		"delete": map[string]interface{}{
			"tags": []interface{}{"Feature flags"}, "operationId": "deleteFeatureFlag",
			"summary": "Delete a feature flag override",
			"parameters": append(tenantParams(),
				map[string]interface{}{"name": "flag_key", "in": "path", "required": true, "schema": map[string]interface{}{"type": "string"}},
				map[string]interface{}{"name": "scope", "in": "path", "required": true, "schema": map[string]interface{}{"type": "string", "enum": []interface{}{"TENANT", "ORGANIZATION"}}},
				map[string]interface{}{"name": "scope_id", "in": "path", "required": true, "schema": map[string]interface{}{"type": "string"}},
			),
			"responses": map[string]interface{}{"200": map[string]interface{}{"description": "Feature flag deleted"}, "404": map[string]interface{}{"description": "Not found"}},
		},
	}
}

func addTrainingPaths(paths map[string]interface{}) {
	paths["/api/v1/training/"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags": []interface{}{"Training"}, "operationId": "listTrainingCourses",
			"summary": "List training courses for the tenant", "parameters": tenantParams(),
			"responses": map[string]interface{}{"200": map[string]interface{}{"description": "Course list", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"type": "array", "items": map[string]interface{}{"$ref": "#/components/schemas/TrainingCourse"}}}}}, "400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"}, "405": map[string]interface{}{"$ref": "#/components/responses/MethodNotAllowed"}},
		},
		"post": map[string]interface{}{
			"tags": []interface{}{"Training"}, "operationId": "createTrainingCourse",
			"summary": "Create a training course", "parameters": tenantParams(),
			"requestBody": map[string]interface{}{"required": true, "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/TrainingCourseCreate"}}}},
			"responses":   map[string]interface{}{"201": map[string]interface{}{"description": "Course created", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/TrainingCourse"}}}}, "400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"}, "405": map[string]interface{}{"$ref": "#/components/responses/MethodNotAllowed"}},
		},
	}
}

func addSettingsPaths(paths map[string]interface{}) {
	paths["/api/v1/admin/settings"] = map[string]interface{}{
		"get": map[string]interface{}{"tags": []interface{}{"Settings"}, "operationId": "listSettings", "summary": "List tenant settings", "parameters": tenantParams(), "responses": map[string]interface{}{"200": map[string]interface{}{"description": "Setting list", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"type": "array", "items": map[string]interface{}{"$ref": "#/components/schemas/Setting"}}}}}, "400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"}, "405": map[string]interface{}{"$ref": "#/components/responses/MethodNotAllowed"}}},
		"put": map[string]interface{}{"tags": []interface{}{"Settings"}, "operationId": "upsertSetting", "summary": "Create or update a setting", "parameters": tenantParams(), "requestBody": map[string]interface{}{"required": true, "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/SettingUpsert"}}}}, "responses": map[string]interface{}{"200": map[string]interface{}{"description": "Setting upserted", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/Setting"}}}}, "400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"}, "405": map[string]interface{}{"$ref": "#/components/responses/MethodNotAllowed"}}},
	}
	paths["/api/v1/admin/settings/"] = map[string]interface{}{
		"delete": map[string]interface{}{"tags": []interface{}{"Settings"}, "operationId": "deleteSetting", "summary": "Delete a setting", "parameters": append(tenantParams(), map[string]interface{}{"name": "key", "in": "path", "required": true, "schema": map[string]interface{}{"type": "string"}}), "responses": map[string]interface{}{"200": map[string]interface{}{"description": "Setting deleted"}, "404": map[string]interface{}{"description": "Not found"}}},
	}
}

func addInspectionPaths(paths map[string]interface{}) {
	paths["/api/v1/inspections"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags": []interface{}{"Inspections"}, "operationId": "createInspection", "summary": "Create an inspection record", "parameters": tenantParams(),
			"requestBody": map[string]interface{}{"required": true, "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/InspectionCreate"}}}},
			"responses":   map[string]interface{}{"201": map[string]interface{}{"description": "Inspection created", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/Inspection"}}}}, "400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"}, "405": map[string]interface{}{"$ref": "#/components/responses/MethodNotAllowed"}},
		},
	}
}

func addSearchPaths(paths map[string]interface{}) {
	paths["/api/v1/search/"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags": []interface{}{"Search"}, "operationId": "searchEntities", "summary": "Full-text search across assets, work orders, and inspections",
			"parameters": append(tenantParams(),
				map[string]interface{}{"name": "q", "in": "query", "required": true, "schema": map[string]interface{}{"type": "string"}, "description": "Search query"},
				map[string]interface{}{"name": "types", "in": "query", "required": false, "schema": map[string]interface{}{"type": "string"}, "description": "Comma-separated entity types to search (ASSET, WORK_ORDER, INSPECTION)"},
			),
			"responses": map[string]interface{}{"200": map[string]interface{}{"description": "Search results", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/SearchResult"}}}}, "400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"}, "405": map[string]interface{}{"$ref": "#/components/responses/MethodNotAllowed"}},
		},
	}
}

func addAuditLogPaths(paths map[string]interface{}) {
	commonParams := []interface{}{
		map[string]interface{}{"name": "entity_type", "in": "query", "required": false, "schema": map[string]interface{}{"type": "string"}},
		map[string]interface{}{"name": "entity_id", "in": "query", "required": false, "schema": map[string]interface{}{"type": "string"}},
		map[string]interface{}{"name": "actor_id", "in": "query", "required": false, "schema": map[string]interface{}{"type": "string"}},
		map[string]interface{}{"name": "event_type", "in": "query", "required": false, "schema": map[string]interface{}{"type": "string"}},
		map[string]interface{}{"name": "from", "in": "query", "required": false, "schema": map[string]interface{}{"type": "string", "format": "date-time"}},
		map[string]interface{}{"name": "to", "in": "query", "required": false, "schema": map[string]interface{}{"type": "string", "format": "date-time"}},
		map[string]interface{}{"name": "limit", "in": "query", "required": false, "schema": map[string]interface{}{"type": "integer", "default": 50, "maximum": 200}},
		map[string]interface{}{"name": "offset", "in": "query", "required": false, "schema": map[string]interface{}{"type": "integer", "default": 0}},
	}
	paths["/api/v1/audit-log"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags": []interface{}{"Audit log"}, "operationId": "queryAuditLog", "summary": "Query immutable audit log entries",
			"parameters": append(append([]interface{}{}, tenantParams()...), commonParams...),
			"responses":  map[string]interface{}{"200": map[string]interface{}{"description": "Audit log entries", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/AuditLogQueryResult"}}}}, "400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"}, "405": map[string]interface{}{"$ref": "#/components/responses/MethodNotAllowed"}},
		},
		"post": map[string]interface{}{
			"tags": []interface{}{"Audit log"}, "operationId": "appendAuditLog", "summary": "Append an immutable audit log entry", "parameters": tenantParams(),
			"requestBody": map[string]interface{}{"required": true, "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/AuditLogAppendRequest"}}}},
			"responses":   map[string]interface{}{"201": map[string]interface{}{"description": "Entry appended", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/AuditLogEntry"}}}}, "400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"}, "405": map[string]interface{}{"$ref": "#/components/responses/MethodNotAllowed"}},
		},
	}
	paths["/api/v1/audit-log/"] = map[string]interface{}{
		"get": map[string]interface{}{"tags": []interface{}{"Audit log"}, "operationId": "queryAuditLogTrailingSlash", "summary": "Query immutable audit log entries", "parameters": tenantParams(), "responses": map[string]interface{}{"200": map[string]interface{}{"description": "Audit log entries", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/AuditLogQueryResult"}}}}}},
	}
	paths["/api/v1/audit-log/verify"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags": []interface{}{"Audit log"}, "operationId": "verifyAuditChain", "summary": "Verify audit log hash chain integrity",
			"parameters": append(tenantParams(),
				map[string]interface{}{"name": "from", "in": "query", "required": false, "schema": map[string]interface{}{"type": "string", "format": "date-time"}},
				map[string]interface{}{"name": "to", "in": "query", "required": false, "schema": map[string]interface{}{"type": "string", "format": "date-time"}},
			),
			"responses": map[string]interface{}{"200": map[string]interface{}{"description": "Chain verification result", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/AuditLogVerifyResult"}}}}, "400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"}, "405": map[string]interface{}{"$ref": "#/components/responses/MethodNotAllowed"}},
		},
	}
}

func addAnalyticsPaths(paths map[string]interface{}) {
	paths["/api/v1/analytics/"] = map[string]interface{}{"get": map[string]interface{}{"tags": []interface{}{"Analytics"}, "operationId": "analyticsRoot", "summary": "Analytics root (dashboard subroutes accessed via /dashboard)", "parameters": tenantParams(), "responses": map[string]interface{}{"405": map[string]interface{}{"$ref": "#/components/responses/MethodNotAllowed"}}}}
	paths["/api/v1/analytics/dashboard"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags": []interface{}{"Analytics"}, "operationId": "getDashboard", "summary": "Get analytics dashboard with KPIs, trends, and breakdowns",
			"parameters": append(tenantParams(),
				map[string]interface{}{"name": "from", "in": "query", "required": false, "schema": map[string]interface{}{"type": "string", "format": "date-time"}, "description": "Start date (default: 1 month ago)"},
				map[string]interface{}{"name": "to", "in": "query", "required": false, "schema": map[string]interface{}{"type": "string", "format": "date-time"}, "description": "End date (default: now)"},
				map[string]interface{}{"name": "inspector_id", "in": "query", "required": false, "schema": map[string]interface{}{"type": "string"}},
				map[string]interface{}{"name": "asset_type", "in": "query", "required": false, "schema": map[string]interface{}{"type": "string"}},
			),
			"responses": map[string]interface{}{"200": map[string]interface{}{"description": "Dashboard data", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/DashboardResponse"}}}}, "400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"}, "405": map[string]interface{}{"$ref": "#/components/responses/MethodNotAllowed"}},
		},
	}
}

func addReportsPaths(paths map[string]interface{}) {
	paths["/api/v1/reports/"] = map[string]interface{}{"get": map[string]interface{}{"tags": []interface{}{"Reports"}, "operationId": "reportsRoot", "summary": "Reports root (subroutes accessed via /configs, /generate)", "parameters": tenantParams(), "responses": map[string]interface{}{"405": map[string]interface{}{"$ref": "#/components/responses/MethodNotAllowed"}}}}
	paths["/api/v1/reports/configs"] = map[string]interface{}{
		"get":  map[string]interface{}{"tags": []interface{}{"Reports"}, "operationId": "listReportConfigs", "summary": "List report configurations", "parameters": tenantParams(), "responses": map[string]interface{}{"200": map[string]interface{}{"description": "Report config list", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"type": "array", "items": map[string]interface{}{"$ref": "#/components/schemas/ReportConfig"}}}}}, "400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"}}},
		"post": map[string]interface{}{"tags": []interface{}{"Reports"}, "operationId": "createReportConfig", "summary": "Create a report configuration", "parameters": tenantParams(), "requestBody": map[string]interface{}{"required": true, "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/ReportConfigCreate"}}}}, "responses": map[string]interface{}{"201": map[string]interface{}{"description": "Report config created", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/ReportConfig"}}}}, "400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"}}},
	}
	paths["/api/v1/reports/generate"] = map[string]interface{}{"post": map[string]interface{}{"tags": []interface{}{"Reports"}, "operationId": "generateReport", "summary": "Generate a report from a config", "parameters": tenantParams(), "requestBody": map[string]interface{}{"required": true, "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/ReportGenerateRequest"}}}}, "responses": map[string]interface{}{"201": map[string]interface{}{"description": "Report generated", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"$ref": "#/components/schemas/GeneratedReport"}}}}, "400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"}}}}
	paths["/api/v1/reports"] = map[string]interface{}{"get": map[string]interface{}{"tags": []interface{}{"Reports"}, "operationId": "listGeneratedReports", "summary": "List generated reports", "parameters": tenantParams(), "responses": map[string]interface{}{"200": map[string]interface{}{"description": "Generated report list", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": map[string]interface{}{"type": "array", "items": map[string]interface{}{"$ref": "#/components/schemas/GeneratedReport"}}}}}, "400": map[string]interface{}{"$ref": "#/components/responses/BadRequest"}}}}
	paths["/api/v1/reports/:id/download"] = map[string]interface{}{"get": map[string]interface{}{"tags": []interface{}{"Reports"}, "operationId": "downloadReport", "summary": "Download a generated report as CSV", "parameters": append(tenantParams(), map[string]interface{}{"name": "id", "in": "path", "required": true, "schema": map[string]interface{}{"type": "string"}}), "responses": map[string]interface{}{"200": map[string]interface{}{"description": "CSV file", "content": map[string]interface{}{"text/csv": map[string]interface{}{"schema": map[string]interface{}{"type": "string", "format": "binary"}}}}, "404": map[string]interface{}{"description": "Report not found"}, "409": map[string]interface{}{"description": "Report not ready"}}}}
}

func addComponentSchemas(doc map[string]interface{}) {
	schemas := doc["components"].(map[string]interface{})["schemas"].(map[string]interface{})
	schemas["License"] = map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "tenant_id": map[string]interface{}{"type": "string"}, "organization_id": map[string]interface{}{"type": "string"}, "tier": map[string]interface{}{"type": "string", "enum": []interface{}{"starter", "professional", "enterprise"}}, "status": map[string]interface{}{"type": "string", "enum": []interface{}{"active", "expired", "revoked"}}, "issued_at": map[string]interface{}{"type": "string", "format": "date-time"}, "expires_at": map[string]interface{}{"type": "string", "format": "date-time"}}}
	schemas["LicenseIssueRequest"] = map[string]interface{}{"type": "object", "required": []interface{}{"tier"}, "properties": map[string]interface{}{"tier": map[string]interface{}{"type": "string", "enum": []interface{}{"starter", "professional", "enterprise"}}, "expires_at": map[string]interface{}{"type": "string", "format": "date-time"}}}
	schemas["FeatureFlag"] = map[string]interface{}{"type": "object", "properties": map[string]interface{}{"flag_key": map[string]interface{}{"type": "string"}, "scope": map[string]interface{}{"type": "string", "enum": []interface{}{"TENANT", "ORGANIZATION"}}, "scope_id": map[string]interface{}{"type": "string"}, "state": map[string]interface{}{"type": "string", "enum": []interface{}{"ENABLED", "DISABLED", "CONDITIONAL"}}, "actor_id": map[string]interface{}{"type": "string"}, "created_at": map[string]interface{}{"type": "string", "format": "date-time"}}}
	schemas["FeatureFlagUpsert"] = map[string]interface{}{"type": "object", "required": []interface{}{"flag_key", "scope", "scope_id", "state", "actor_id"}, "properties": map[string]interface{}{"flag_key": map[string]interface{}{"type": "string"}, "scope": map[string]interface{}{"type": "string", "enum": []interface{}{"TENANT", "ORGANIZATION"}}, "scope_id": map[string]interface{}{"type": "string"}, "state": map[string]interface{}{"type": "string", "enum": []interface{}{"ENABLED", "DISABLED", "CONDITIONAL"}}, "actor_id": map[string]interface{}{"type": "string"}}}
	schemas["TrainingCourse"] = map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "title": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"}, "created_at": map[string]interface{}{"type": "string", "format": "date-time"}}}
	schemas["TrainingCourseCreate"] = map[string]interface{}{"type": "object", "required": []interface{}{"title"}, "properties": map[string]interface{}{"title": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"}}}
	schemas["Setting"] = map[string]interface{}{"type": "object", "properties": map[string]interface{}{"key": map[string]interface{}{"type": "string"}, "value": map[string]interface{}{"type": "string"}, "updated_at": map[string]interface{}{"type": "string", "format": "date-time"}}}
	schemas["SettingUpsert"] = map[string]interface{}{"type": "object", "required": []interface{}{"key", "value"}, "properties": map[string]interface{}{"key": map[string]interface{}{"type": "string"}, "value": map[string]interface{}{"type": "string"}}}
	schemas["InspectionCreate"] = map[string]interface{}{"type": "object", "required": []interface{}{"work_order_id", "scope_item_id", "assignment_id", "asset_id", "inspector_id"}, "properties": map[string]interface{}{"work_order_id": map[string]interface{}{"type": "string"}, "scope_item_id": map[string]interface{}{"type": "string"}, "assignment_id": map[string]interface{}{"type": "string"}, "asset_id": map[string]interface{}{"type": "string"}, "inspector_id": map[string]interface{}{"type": "string"}}}
	schemas["Inspection"] = map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "work_order_id": map[string]interface{}{"type": "string"}, "asset_id": map[string]interface{}{"type": "string"}, "inspector_id": map[string]interface{}{"type": "string"}, "lifecycle_state": map[string]interface{}{"type": "string"}, "finalization_state": map[string]interface{}{"type": "string"}, "created_at": map[string]interface{}{"type": "string", "format": "date-time"}}}
	schemas["SearchResult"] = map[string]interface{}{"type": "object", "properties": map[string]interface{}{"results": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "entity_type": map[string]interface{}{"type": "string"}, "title": map[string]interface{}{"type": "string"}, "snippet": map[string]interface{}{"type": "string"}, "rank": map[string]interface{}{"type": "number"}, "created_at": map[string]interface{}{"type": "string", "format": "date-time"}}}}, "total": map[string]interface{}{"type": "integer"}, "query": map[string]interface{}{"type": "string"}}}
	schemas["AuditLogEntry"] = map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "tenant_id": map[string]interface{}{"type": "string"}, "organization_id": map[string]interface{}{"type": "string"}, "event_type": map[string]interface{}{"type": "string"}, "entity_type": map[string]interface{}{"type": "string"}, "entity_id": map[string]interface{}{"type": "string"}, "actor_id": map[string]interface{}{"type": "string"}, "actor_name": map[string]interface{}{"type": "string"}, "action": map[string]interface{}{"type": "string"}, "old_value": map[string]interface{}{"type": "object"}, "new_value": map[string]interface{}{"type": "object"}, "metadata": map[string]interface{}{"type": "object"}, "ip_address": map[string]interface{}{"type": "string"}, "user_agent": map[string]interface{}{"type": "string"}, "previous_hash": map[string]interface{}{"type": "string"}, "entry_hash": map[string]interface{}{"type": "string"}, "created_at": map[string]interface{}{"type": "string", "format": "date-time"}}}
	schemas["AuditLogAppendRequest"] = map[string]interface{}{"type": "object", "required": []interface{}{"event_type", "entity_type", "entity_id", "actor_id", "action"}, "properties": map[string]interface{}{"event_type": map[string]interface{}{"type": "string"}, "entity_type": map[string]interface{}{"type": "string"}, "entity_id": map[string]interface{}{"type": "string"}, "actor_id": map[string]interface{}{"type": "string"}, "actor_name": map[string]interface{}{"type": "string"}, "action": map[string]interface{}{"type": "string"}, "old_value": map[string]interface{}{"type": "object"}, "new_value": map[string]interface{}{"type": "object"}, "metadata": map[string]interface{}{"type": "object"}, "ip_address": map[string]interface{}{"type": "string"}, "user_agent": map[string]interface{}{"type": "string"}}}
	schemas["AuditLogQueryResult"] = map[string]interface{}{"type": "object", "properties": map[string]interface{}{"entries": map[string]interface{}{"type": "array", "items": map[string]interface{}{"$ref": "#/components/schemas/AuditLogEntry"}}, "total": map[string]interface{}{"type": "integer"}}}
	schemas["AuditLogVerifyResult"] = map[string]interface{}{"type": "object", "properties": map[string]interface{}{"valid": map[string]interface{}{"type": "boolean"}, "errors": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}}}
	schemas["DashboardResponse"] = map[string]interface{}{"type": "object", "properties": map[string]interface{}{"summary": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}, "value": map[string]interface{}{"type": "number"}, "unit": map[string]interface{}{"type": "string"}, "trend": map[string]interface{}{"type": "string", "enum": []interface{}{"up", "down", "flat"}}, "change_pct": map[string]interface{}{"type": "number"}}}}, "work_orders_by_state": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"state": map[string]interface{}{"type": "string"}, "count": map[string]interface{}{"type": "integer"}}}}, "inspections_by_state": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"state": map[string]interface{}{"type": "string"}, "count": map[string]interface{}{"type": "integer"}}}}, "inspections_trend": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"timestamp": map[string]interface{}{"type": "string", "format": "date-time"}, "value": map[string]interface{}{"type": "number"}, "label": map[string]interface{}{"type": "string"}}}}, "inspector_performance": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"inspector_id": map[string]interface{}{"type": "string"}, "inspector_name": map[string]interface{}{"type": "string"}, "total": map[string]interface{}{"type": "integer"}, "pass_count": map[string]interface{}{"type": "integer"}, "fail_count": map[string]interface{}{"type": "integer"}, "pass_rate": map[string]interface{}{"type": "number"}}}}, "asset_breakdown": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"asset_type": map[string]interface{}{"type": "string"}, "count": map[string]interface{}{"type": "integer"}}}}}}
	schemas["ReportConfig"] = map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "tenant_id": map[string]interface{}{"type": "string"}, "organization_id": map[string]interface{}{"type": "string"}, "name": map[string]interface{}{"type": "string"}, "type": map[string]interface{}{"type": "string", "enum": []interface{}{"inspection_summary", "work_order_status", "asset_inventory", "inspector_performance"}}, "format": map[string]interface{}{"type": "string", "enum": []interface{}{"csv", "pdf"}}, "schedule": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"frequency": map[string]interface{}{"type": "string", "enum": []interface{}{"daily", "weekly", "monthly"}}, "time": map[string]interface{}{"type": "string"}}}, "created_by": map[string]interface{}{"type": "string"}, "created_at": map[string]interface{}{"type": "string", "format": "date-time"}, "updated_at": map[string]interface{}{"type": "string", "format": "date-time"}}}
	schemas["ReportConfigCreate"] = map[string]interface{}{"type": "object", "required": []interface{}{"name", "type"}, "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}, "type": map[string]interface{}{"type": "string", "enum": []interface{}{"inspection_summary", "work_order_status", "asset_inventory", "inspector_performance"}}, "format": map[string]interface{}{"type": "string", "enum": []interface{}{"csv", "pdf"}}, "schedule": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"frequency": map[string]interface{}{"type": "string", "enum": []interface{}{"daily", "weekly", "monthly"}}, "time": map[string]interface{}{"type": "string"}}}}}
	schemas["ReportGenerateRequest"] = map[string]interface{}{"type": "object", "required": []interface{}{"config_id"}, "properties": map[string]interface{}{"config_id": map[string]interface{}{"type": "string"}, "date_from": map[string]interface{}{"type": "string"}, "date_to": map[string]interface{}{"type": "string"}, "format_override": map[string]interface{}{"type": "string", "enum": []interface{}{"csv", "pdf"}}}}
	schemas["GeneratedReport"] = map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "config_id": map[string]interface{}{"type": "string"}, "tenant_id": map[string]interface{}{"type": "string"}, "organization_id": map[string]interface{}{"type": "string"}, "status": map[string]interface{}{"type": "string", "enum": []interface{}{"pending", "generating", "ready", "failed"}}, "file_path": map[string]interface{}{"type": "string"}, "row_count": map[string]interface{}{"type": "integer"}, "error": map[string]interface{}{"type": "string"}, "created_at": map[string]interface{}{"type": "string", "format": "date-time"}}}
}

func addParameters(doc map[string]interface{}) {
	params := doc["components"].(map[string]interface{})["parameters"].(map[string]interface{})
	params["TenantID"] = map[string]interface{}{"name": "X-Tenant-ID", "in": "header", "required": true, "schema": map[string]interface{}{"type": "string"}, "description": "Tenant identifier"}
	params["OrganizationID"] = map[string]interface{}{"name": "X-Organization-ID", "in": "header", "required": true, "schema": map[string]interface{}{"type": "string"}, "description": "Organization identifier"}
}

func updateTags(doc map[string]interface{}) {
	doc["tags"] = []interface{}{
		map[string]interface{}{"name": "Operational", "description": "Health and readiness probes."},
		map[string]interface{}{"name": "Field sync", "description": "Signed, server-authoritative field transactions."},
		map[string]interface{}{"name": "Evidence", "description": "Encrypted evidence-object submission."},
		map[string]interface{}{"name": "Local provisioning", "description": "Explicitly local-only integration bridge."},
		map[string]interface{}{"name": "Identity", "description": "Optional pilot-only OIDC session boundary."},
		map[string]interface{}{"name": "Work orders", "description": "Work order management."},
		map[string]interface{}{"name": "License", "description": "License entitlement management."},
		map[string]interface{}{"name": "Feature flags", "description": "Feature flag runtime toggle."},
		map[string]interface{}{"name": "Training", "description": "Training course management."},
		map[string]interface{}{"name": "Settings", "description": "Tenant settings management."},
		map[string]interface{}{"name": "Inspections", "description": "Inspection record management."},
		map[string]interface{}{"name": "Search", "description": "Full-text search across entities."},
		map[string]interface{}{"name": "Audit log", "description": "Immutable audit log with hash chain."},
		map[string]interface{}{"name": "Analytics", "description": "Analytics dashboards and KPIs."},
		map[string]interface{}{"name": "Reports", "description": "Scheduled report generation."},
	}
}
