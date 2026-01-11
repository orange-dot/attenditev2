# Admin access control policies
package admin.access

import future.keywords.if
import future.keywords.in
import data.common.roles

default allow := false

# ============================================
# Agency Management
# ============================================

# Platform admins can manage all agencies
allow if {
    input.resource_type == "agency"
    input.action in {"create", "read", "update", "delete"}
    roles.is_system_admin
}

# Platform operators can read agencies
allow if {
    input.resource_type == "agency"
    input.action == "read"
    roles.is_platform_operator
}

# Agency admins can update their own agency
allow if {
    input.resource_type == "agency"
    input.action == "update"
    roles.is_agency_admin
    input.actor_agency_id == input.resource.id
}

# ============================================
# Worker Management
# ============================================

# Platform admins can manage all workers
allow if {
    input.resource_type == "worker"
    input.action in {"create", "read", "update", "delete"}
    roles.is_system_admin
}

# Agency admins can manage workers in their agency
allow if {
    input.resource_type == "worker"
    input.action in {"create", "read", "update", "delete"}
    roles.is_agency_admin
    input.actor_agency_id == input.resource.agency_id
}

# Agency supervisors can read workers in their agency
allow if {
    input.resource_type == "worker"
    input.action == "read"
    roles.is_agency_supervisor
    input.actor_agency_id == input.resource.agency_id
}

# ============================================
# Audit Access
# ============================================

# Platform admins can access all audit logs
allow if {
    input.resource_type == "audit"
    input.action in {"read", "export"}
    roles.is_system_admin
}

# Security auditors can read and export audit logs
allow if {
    input.resource_type == "audit"
    input.action in {"read", "export"}
    roles.is_security_auditor
}

# Agency admins can read audit logs for their agency
allow if {
    input.resource_type == "audit"
    input.action == "read"
    roles.is_agency_admin
    input.resource.agency_id == input.actor_agency_id
}

# ============================================
# System Configuration
# ============================================

# Only platform admins can modify system config
allow if {
    input.resource_type == "system_config"
    input.action in {"read", "update"}
    roles.is_system_admin
}

# Platform operators can read system config
allow if {
    input.resource_type == "system_config"
    input.action == "read"
    roles.is_platform_operator
}

# ============================================
# Role Management
# ============================================

# Platform admins can assign any role
allow if {
    input.action == "assign_role"
    roles.is_system_admin
}

# Agency admins can assign agency-level roles
allow if {
    input.action == "assign_role"
    roles.is_agency_admin
    input.role in {"case_worker", "dispatch_operator", "field_unit", "agency_viewer", "agency_supervisor"}
    input.target_agency_id == input.actor_agency_id
}

# ============================================
# De-pseudonymization Approval
# ============================================

# Supervisors can approve de-pseudonymization requests in their agency
allow if {
    input.resource_type == "depseudonymization_request"
    input.action == "approve"
    roles.has_supervisory_role
    input.resource.facility_code == input.actor_facility_code
}

# Platform admins can approve any de-pseudonymization request
allow if {
    input.resource_type == "depseudonymization_request"
    input.action == "approve"
    roles.is_system_admin
}

# ============================================
# Denial Reasons
# ============================================

reasons[msg] if {
    not allow
    input.resource_type == "agency"
    msg := "Access denied: insufficient permissions for agency management"
}

reasons[msg] if {
    not allow
    input.resource_type == "worker"
    msg := "Access denied: insufficient permissions for worker management"
}

reasons[msg] if {
    not allow
    input.resource_type == "audit"
    msg := "Access denied: insufficient permissions for audit access"
}

reasons[msg] if {
    not allow
    input.action == "assign_role"
    msg := "Access denied: cannot assign this role"
}
