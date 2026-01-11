package platform.case

import future.keywords.if
import future.keywords.in

# Default deny
default allow := false

# Allow if user is from owning agency
allow if {
    input.actor_agency_id == input.resource.owning_agency_id
}

# Allow read access if agency is in shared_with list
allow if {
    input.action == "read"
    input.actor_agency_id in input.resource.shared_with
}

# Allow read/write access if agency has appropriate access level
allow if {
    input.action in ["read", "write"]
    some share in input.resource.shares
    share.agency_id == input.actor_agency_id
    share.access_level in ["read", "write", "full"]
}

# Allow full access if agency has full access level
allow if {
    some share in input.resource.shares
    share.agency_id == input.actor_agency_id
    share.access_level == "full"
}

# Platform admins can access all cases
allow if {
    "platform_admin" in input.roles
}

# Agency admins can access their agency's cases
allow if {
    "agency_admin" in input.roles
    input.actor_agency_id == input.resource.owning_agency_id
}

# Workers assigned to case can access it
allow if {
    some assignment in input.resource.assignments
    assignment.worker_id == input.actor_id
    assignment.status == "active"
}

# Reasons for denial
reasons[msg] if {
    not allow
    msg := "Access denied: not authorized for this case"
}

reasons[msg] if {
    not input.actor_agency_id
    msg := "Access denied: no agency context"
}

# Field-level access control
fields["jmbg"] if {
    # Only allow JMBG access for specific roles
    "sensitive_data_access" in input.permissions
}

fields["jmbg"] if {
    # Or if user is from owning agency with supervisor role
    input.actor_agency_id == input.resource.owning_agency_id
    "supervisor" in input.roles
}

# Audit trail access
fields["audit_trail"] if {
    "audit_viewer" in input.permissions
}

fields["audit_trail"] if {
    "agency_admin" in input.roles
}
