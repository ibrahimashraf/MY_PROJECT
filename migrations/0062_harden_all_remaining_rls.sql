-- integin Hardening: Comprehensive RLS NULLIF & Non-Null Protection Across All Remaining Tables
-- Builds on 0061_kill_gin_and_dark_hardening.sql.
-- Guarantees that empty string session parameters can never access or insert multi-tenant rows.

BEGIN;

DROP POLICY IF EXISTS anomaly_alerts_isolation ON anomaly_alerts;
CREATE POLICY anomaly_alerts_isolation ON anomaly_alerts
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS anomaly_rules_isolation ON anomaly_rules;
CREATE POLICY anomaly_rules_isolation ON anomaly_rules
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS asset_entitlement_tenant_organization_isolation ON asset_entitlement;
CREATE POLICY asset_entitlement_tenant_organization_isolation ON asset_entitlement
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS asset_registry_tenant_isolation ON asset_registry;
CREATE POLICY asset_registry_tenant_isolation ON asset_registry
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS asset_tag_tenant_organization_isolation ON asset_tag;
CREATE POLICY asset_tag_tenant_organization_isolation ON asset_tag
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS assurance_projection_tenant_organization_isolation ON assurance_projection;
CREATE POLICY assurance_projection_tenant_organization_isolation ON assurance_projection
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS audit_log_tenant_organization_isolation ON audit_log;
CREATE POLICY audit_log_tenant_organization_isolation ON audit_log
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS audit_trail_export_tenant_isolation ON audit_trail_export;
CREATE POLICY audit_trail_export_tenant_isolation ON audit_trail_export
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS authority_package_tenant_isolation ON authority_package;
CREATE POLICY authority_package_tenant_isolation ON authority_package
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS bulk_export_job_tenant_isolation ON bulk_export_job;
CREATE POLICY bulk_export_job_tenant_isolation ON bulk_export_job
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS bulk_import_job_tenant_isolation ON bulk_import_job;
CREATE POLICY bulk_import_job_tenant_isolation ON bulk_import_job
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS bulk_import_row_tenant_isolation ON bulk_import_row;
CREATE POLICY bulk_import_row_tenant_isolation ON bulk_import_row
    USING (
        job_id IN (
            SELECT id FROM bulk_import_job 
            WHERE tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
              AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
              AND tenant_id IS NOT NULL AND organization_id IS NOT NULL
        )
    )
    WITH CHECK (
        job_id IN (
            SELECT id FROM bulk_import_job 
            WHERE tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
              AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
              AND tenant_id IS NOT NULL AND organization_id IS NOT NULL
        )
    );
DROP POLICY IF EXISTS certificate_artifact_tenant_isolation ON certificate_artifact;
CREATE POLICY certificate_artifact_tenant_isolation ON certificate_artifact
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS certificate_audit_event_tenant_isolation ON certificate_audit_event;
CREATE POLICY certificate_audit_event_tenant_isolation ON certificate_audit_event
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS certificate_number_sequence_tenant_isolation ON certificate_number_sequence;
CREATE POLICY certificate_number_sequence_tenant_isolation ON certificate_number_sequence
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS certificate_pack_tenant_isolation ON certificate_pack;
CREATE POLICY certificate_pack_tenant_isolation ON certificate_pack
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS certificate_pack_item_tenant_isolation ON certificate_pack_item;
CREATE POLICY certificate_pack_item_tenant_isolation ON certificate_pack_item
    USING (
        pack_id IN (
            SELECT id FROM certificate_pack 
            WHERE tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
              AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
              AND tenant_id IS NOT NULL AND organization_id IS NOT NULL
        )
    )
    WITH CHECK (
        pack_id IN (
            SELECT id FROM certificate_pack 
            WHERE tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
              AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
              AND tenant_id IS NOT NULL AND organization_id IS NOT NULL
        )
    );
DROP POLICY IF EXISTS certificate_policy_tenant_isolation ON certificate_policy;
CREATE POLICY certificate_policy_tenant_isolation ON certificate_policy
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS certificate_policy_public_binding_tenant_isolation ON certificate_policy_public_binding;
CREATE POLICY certificate_policy_public_binding_tenant_isolation ON certificate_policy_public_binding
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS certificate_record_tenant_isolation ON certificate_record;
CREATE POLICY certificate_record_tenant_isolation ON certificate_record
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS certificate_renewal_attempt_tenant_isolation ON certificate_renewal_attempt;
CREATE POLICY certificate_renewal_attempt_tenant_isolation ON certificate_renewal_attempt
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS certificate_snapshot_tenant_isolation ON certificate_snapshot;
CREATE POLICY certificate_snapshot_tenant_isolation ON certificate_snapshot
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS certificate_template_tenant_isolation ON certificate_template;
CREATE POLICY certificate_template_tenant_isolation ON certificate_template
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS certificate_template_cell_tenant_isolation ON certificate_template_cell;
CREATE POLICY certificate_template_cell_tenant_isolation ON certificate_template_cell
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS certificate_template_docx_tenant_isolation ON certificate_template_docx;
CREATE POLICY certificate_template_docx_tenant_isolation ON certificate_template_docx
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS client_audit_entry_tenant_isolation ON client_audit_entry;
CREATE POLICY client_audit_entry_tenant_isolation ON client_audit_entry
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS client_portal_acl_tenant_isolation ON client_portal_acl;
CREATE POLICY client_portal_acl_tenant_isolation ON client_portal_acl
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS client_portal_domain_tenant_isolation ON client_portal_domain;
CREATE POLICY client_portal_domain_tenant_isolation ON client_portal_domain
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS comment_library_isolation ON comment_library;
CREATE POLICY comment_library_isolation ON comment_library
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS compliance_action_tenant_isolation ON compliance_action;
CREATE POLICY compliance_action_tenant_isolation ON compliance_action
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS corrective_work_tenant_organization_isolation ON corrective_work;
CREATE POLICY corrective_work_tenant_organization_isolation ON corrective_work
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS course_tenant_isolation ON course;
CREATE POLICY course_tenant_isolation ON course
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS course_enrollment_tenant_isolation ON course_enrollment;
CREATE POLICY course_enrollment_tenant_isolation ON course_enrollment
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS csv_export_job_tenant_isolation ON csv_export_job;
CREATE POLICY csv_export_job_tenant_isolation ON csv_export_job
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS custody_chain_tenant_organization_isolation ON custody_chain;
CREATE POLICY custody_chain_tenant_organization_isolation ON custody_chain
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS data_api_token_tenant_isolation ON data_api_token;
CREATE POLICY data_api_token_tenant_isolation ON data_api_token
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS device_registry_tenant_isolation ON device_registry;
CREATE POLICY device_registry_tenant_isolation ON device_registry
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS equipment_type_tenant_isolation ON equipment_type;
CREATE POLICY equipment_type_tenant_isolation ON equipment_type
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS equipment_type_field_tenant_isolation ON equipment_type_field;
CREATE POLICY equipment_type_field_tenant_isolation ON equipment_type_field
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS escalation_event_isolation ON escalation_event;
CREATE POLICY escalation_event_isolation ON escalation_event
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS escalation_rule_isolation ON escalation_rule;
CREATE POLICY escalation_rule_isolation ON escalation_rule
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS event_log_tenant_isolation ON event_log;
CREATE POLICY event_log_tenant_isolation ON event_log
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS event_log_default_tenant_isolation ON event_log_default;
CREATE POLICY event_log_default_tenant_isolation ON event_log_default
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS evidence_metadata_tenant_isolation ON evidence_metadata;
CREATE POLICY evidence_metadata_tenant_isolation ON evidence_metadata
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS evidence_pack_tenant_organization_isolation ON evidence_pack;
CREATE POLICY evidence_pack_tenant_organization_isolation ON evidence_pack
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS evidence_policy_tenant_organization_isolation ON evidence_policy;
CREATE POLICY evidence_policy_tenant_organization_isolation ON evidence_policy
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS export_template_tenant_isolation ON export_template;
CREATE POLICY export_template_tenant_isolation ON export_template
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS failed_inspection_queue_tenant_isolation ON failed_inspection_queue;
CREATE POLICY failed_inspection_queue_tenant_isolation ON failed_inspection_queue
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS feature_flag_override_isolation ON feature_flag_override;
CREATE POLICY feature_flag_override_isolation ON feature_flag_override
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS form_field_tenant_organization_isolation ON form_field;
CREATE POLICY form_field_tenant_organization_isolation ON form_field
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS form_version_tenant_organization_isolation ON form_version;
CREATE POLICY form_version_tenant_organization_isolation ON form_version
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS hse_notification_tenant_isolation ON hse_notification;
CREATE POLICY hse_notification_tenant_isolation ON hse_notification
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS inspect_template_preset_isolation ON inspect_template_preset;
CREATE POLICY inspect_template_preset_isolation ON inspect_template_preset
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS inspection_comment_isolation ON inspection_comment;
CREATE POLICY inspection_comment_isolation ON inspection_comment
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS inspection_public_scope_tenant_isolation ON inspection_public_scope;
CREATE POLICY inspection_public_scope_tenant_isolation ON inspection_public_scope
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS inspection_public_scope_item_tenant_isolation ON inspection_public_scope_item;
CREATE POLICY inspection_public_scope_item_tenant_isolation ON inspection_public_scope_item
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
-- Drop duplicate legacy policy to prevent OR dilution on inspection_record
DROP POLICY IF EXISTS inspection_record_tenant_isolation ON inspection_record;
DROP POLICY IF EXISTS inspection_record_tenant_organization_isolation ON inspection_record;
CREATE POLICY inspection_record_tenant_organization_isolation ON inspection_record
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS integration_config_tenant_isolation ON integration_config;
CREATE POLICY integration_config_tenant_isolation ON integration_config
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS job_linkage_config_tenant_isolation ON job_linkage_config;
CREATE POLICY job_linkage_config_tenant_isolation ON job_linkage_config
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS license_audit_isolation ON license_audit;
CREATE POLICY license_audit_isolation ON license_audit
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS location_area_tenant_isolation ON location_area;
CREATE POLICY location_area_tenant_isolation ON location_area
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS location_branch_tenant_isolation ON location_branch;
CREATE POLICY location_branch_tenant_isolation ON location_branch
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS location_zone_tenant_isolation ON location_zone;
CREATE POLICY location_zone_tenant_isolation ON location_zone
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS manifest_proof_replay_tenant_isolation ON manifest_proof_replay;
CREATE POLICY manifest_proof_replay_tenant_isolation ON manifest_proof_replay
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS multi_inspect_batch_isolation ON multi_inspect_batch;
CREATE POLICY multi_inspect_batch_isolation ON multi_inspect_batch
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS multi_inspect_item_isolation ON multi_inspect_item;
CREATE POLICY multi_inspect_item_isolation ON multi_inspect_item
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS notification_template_isolation ON notification_template;
CREATE POLICY notification_template_isolation ON notification_template
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS offline_package_tenant_organization_isolation ON offline_package;
CREATE POLICY offline_package_tenant_organization_isolation ON offline_package
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS parts_catalog_tenant_isolation ON parts_catalog;
CREATE POLICY parts_catalog_tenant_isolation ON parts_catalog
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS photo_markup_tenant_isolation ON photo_markup;
CREATE POLICY photo_markup_tenant_isolation ON photo_markup
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS product_passport_dpp_tenant_isolation ON product_passport_dpp;
CREATE POLICY product_passport_dpp_tenant_isolation ON product_passport_dpp
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS qr_nfc_entry_tenant_organization_isolation ON qr_nfc_entry;
CREATE POLICY qr_nfc_entry_tenant_organization_isolation ON qr_nfc_entry
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS qr_nfc_entry_log_tenant_organization_isolation ON qr_nfc_entry_log;
CREATE POLICY qr_nfc_entry_log_tenant_organization_isolation ON qr_nfc_entry_log
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS quick_link_tenant_isolation ON quick_link;
CREATE POLICY quick_link_tenant_isolation ON quick_link
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS regulatory_monitor_tenant_isolation ON regulatory_monitor;
CREATE POLICY regulatory_monitor_tenant_isolation ON regulatory_monitor
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS release_pack_tenant_organization_isolation ON release_pack;
CREATE POLICY release_pack_tenant_organization_isolation ON release_pack
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS schedule_calendar_entry_tenant_isolation ON schedule_calendar_entry;
CREATE POLICY schedule_calendar_entry_tenant_isolation ON schedule_calendar_entry
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS scheduling_rule_tenant_isolation ON scheduling_rule;
CREATE POLICY scheduling_rule_tenant_isolation ON scheduling_rule
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS service_charge_tenant_isolation ON service_charge;
CREATE POLICY service_charge_tenant_isolation ON service_charge
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS short_link_bulk_jobs_isolation ON short_link_bulk_jobs;
CREATE POLICY short_link_bulk_jobs_isolation ON short_link_bulk_jobs
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS short_link_hmac_secrets_isolation ON short_link_hmac_secrets;
CREATE POLICY short_link_hmac_secrets_isolation ON short_link_hmac_secrets
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS short_link_scan_events_isolation ON short_link_scan_events;
CREATE POLICY short_link_scan_events_isolation ON short_link_scan_events
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS short_links_isolation ON short_links;
CREATE POLICY short_links_isolation ON short_links
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS sync_device_state_tenant_isolation ON sync_device_state;
CREATE POLICY sync_device_state_tenant_isolation ON sync_device_state
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS sync_held_transaction_tenant_isolation ON sync_held_transaction;
CREATE POLICY sync_held_transaction_tenant_isolation ON sync_held_transaction
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS sync_job_tenant_isolation ON sync_job;
CREATE POLICY sync_job_tenant_isolation ON sync_job
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS sync_receipt_tenant_isolation ON sync_receipt;
CREATE POLICY sync_receipt_tenant_isolation ON sync_receipt
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS technician_competency_tenant_isolation ON technician_competency;
CREATE POLICY technician_competency_tenant_isolation ON technician_competency
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS tenant_license_isolation ON tenant_license;
CREATE POLICY tenant_license_isolation ON tenant_license
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS tenant_setting_tenant_isolation ON tenant_setting;
CREATE POLICY tenant_setting_tenant_isolation ON tenant_setting
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS time_sheet_tenant_isolation ON time_sheet;
CREATE POLICY time_sheet_tenant_isolation ON time_sheet
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS timesheet_auto_capture_tenant_isolation ON timesheet_auto_capture;
CREATE POLICY timesheet_auto_capture_tenant_isolation ON timesheet_auto_capture
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS webhook_deliveries_isolation ON webhook_deliveries;
CREATE POLICY webhook_deliveries_isolation ON webhook_deliveries
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
-- Drop duplicate legacy policy to prevent OR dilution on work_order
DROP POLICY IF EXISTS work_order_tenant_isolation ON work_order;
DROP POLICY IF EXISTS work_order_tenant_organization_isolation ON work_order;
CREATE POLICY work_order_tenant_organization_isolation ON work_order
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
-- Drop duplicate legacy policy to prevent OR dilution on work_order_assignment
DROP POLICY IF EXISTS work_order_assignment_tenant_isolation ON work_order_assignment;
DROP POLICY IF EXISTS work_order_assignment_tenant_organization_isolation ON work_order_assignment;
CREATE POLICY work_order_assignment_tenant_organization_isolation ON work_order_assignment
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
-- Drop duplicate legacy policy to prevent OR dilution on work_order_assignment_scope
DROP POLICY IF EXISTS work_order_assignment_scope_tenant_isolation ON work_order_assignment_scope;
DROP POLICY IF EXISTS work_order_assignment_scope_tenant_organization_isolation ON work_order_assignment_scope;
CREATE POLICY work_order_assignment_scope_tenant_organization_isolation ON work_order_assignment_scope
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS work_order_evidence_tenant_organization_isolation ON work_order_evidence;
CREATE POLICY work_order_evidence_tenant_organization_isolation ON work_order_evidence
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS work_order_handover_tenant_organization_isolation ON work_order_handover;
CREATE POLICY work_order_handover_tenant_organization_isolation ON work_order_handover
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
-- Drop duplicate legacy policy to prevent OR dilution on work_order_operation
DROP POLICY IF EXISTS work_order_operation_tenant_isolation ON work_order_operation;
DROP POLICY IF EXISTS work_order_operation_tenant_organization_isolation ON work_order_operation;
CREATE POLICY work_order_operation_tenant_organization_isolation ON work_order_operation
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
-- Drop duplicate legacy policy to prevent OR dilution on work_order_provisional_record
DROP POLICY IF EXISTS work_order_provisional_record_tenant_isolation ON work_order_provisional_record;
DROP POLICY IF EXISTS work_order_provisional_record_tenant_organization_isolation ON work_order_provisional_record;
CREATE POLICY work_order_provisional_record_tenant_organization_isolation ON work_order_provisional_record
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
-- Drop duplicate legacy policy to prevent OR dilution on work_order_scope_item
DROP POLICY IF EXISTS work_order_scope_item_tenant_isolation ON work_order_scope_item;
DROP POLICY IF EXISTS work_order_scope_item_tenant_organization_isolation ON work_order_scope_item;
CREATE POLICY work_order_scope_item_tenant_organization_isolation ON work_order_scope_item
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS work_order_site_handover_tenant_organization_isolation ON work_order_site_handover;
CREATE POLICY work_order_site_handover_tenant_organization_isolation ON work_order_site_handover
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
-- Drop duplicate legacy policy to prevent OR dilution on work_order_state_event
DROP POLICY IF EXISTS work_order_state_event_tenant_isolation ON work_order_state_event;
DROP POLICY IF EXISTS work_order_state_event_tenant_organization_isolation ON work_order_state_event;
CREATE POLICY work_order_state_event_tenant_organization_isolation ON work_order_state_event
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
-- Drop duplicate legacy policy to prevent OR dilution on work_order_submission_item
DROP POLICY IF EXISTS work_order_submission_item_tenant_isolation ON work_order_submission_item;
DROP POLICY IF EXISTS work_order_submission_item_tenant_organization_isolation ON work_order_submission_item;
CREATE POLICY work_order_submission_item_tenant_organization_isolation ON work_order_submission_item
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
-- Drop duplicate legacy policy to prevent OR dilution on work_order_submission_segment
DROP POLICY IF EXISTS work_order_submission_segment_tenant_isolation ON work_order_submission_segment;
DROP POLICY IF EXISTS work_order_submission_segment_tenant_organization_isolation ON work_order_submission_segment;
CREATE POLICY work_order_submission_segment_tenant_organization_isolation ON work_order_submission_segment
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS work_package_tenant_isolation ON work_package;
CREATE POLICY work_package_tenant_isolation ON work_package
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS work_package_assignment_tenant_isolation ON work_package_assignment;
CREATE POLICY work_package_assignment_tenant_isolation ON work_package_assignment
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
DROP POLICY IF EXISTS work_package_assignment_context_tenant_isolation ON work_package_assignment_context;
CREATE POLICY work_package_assignment_context_tenant_isolation ON work_package_assignment_context
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );
COMMIT;

