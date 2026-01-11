# Common helper functions for Serbia Government Platform
package common.helpers

import future.keywords.if
import future.keywords.in

# Access levels for cross-agency sharing
access_levels := {
    "none": 0,
    "read": 1,
    "comment": 2,
    "contribute": 3,
    "full": 4
}

# Data classification levels
data_classification := {
    "public": 0,
    "internal": 1,
    "confidential": 2,
    "restricted": 3,
    "secret": 4
}

# Map action to required access level
required_access_level(action) := 1 if action == "read"
required_access_level(action) := 2 if action == "comment"
required_access_level(action) := 3 if action in {"update", "assign"}
required_access_level(action) := 4 if action in {"transfer", "close", "delete"}
required_access_level(_) := 0

# Check if user has required access level for agency
has_access_level(agency_id, resource, required_level) if {
    some share in resource.shares
    share.agency_id == agency_id
    access_levels[share.access_level] >= required_level
}

# Check if user is in their working hours (basic implementation)
is_working_hours if {
    # For production: would check actual time against user's agency schedule
    true
}

# Check if request is from allowed IP range
is_allowed_ip if {
    # For production: would validate against agency IP ranges
    true
}

# Mask JMBG for display: "0101990******"
mask_jmbg(jmbg) := masked if {
    count(jmbg) >= 7
    masked := concat("", [substring(jmbg, 0, 7), "******"])
}

mask_jmbg(jmbg) := "***********" if {
    count(jmbg) < 7
}

# Check if user is from same agency as resource
same_agency(user_agency_id, resource) if {
    user_agency_id == resource.owning_agency_id
}

same_agency(user_agency_id, resource) if {
    user_agency_id == resource.owner_agency_id
}

# Check if user is assigned to case
is_assigned_to_case(user_id, case_resource) if {
    some assignment in case_resource.assignments
    assignment.worker_id == user_id
    assignment.status == "active"
}

# Check if case is shared with user's agency
case_shared_with_agency(agency_id, case_resource) if {
    agency_id in case_resource.shared_with
}

case_shared_with_agency(agency_id, case_resource) if {
    some share in case_resource.shares
    share.agency_id == agency_id
}

# Get user's role on a case
case_role(user_id, case_resource) := role if {
    some assignment in case_resource.assignments
    assignment.worker_id == user_id
    role := assignment.role
}

# Check if user is lead on case
is_case_lead(user_id, case_resource) if {
    case_role(user_id, case_resource) == "lead"
}

# Check if action is read-only
is_read_only_action(action) if {
    action in {"read", "view", "list", "get"}
}

# Check if action modifies data
is_write_action(action) if {
    action in {"create", "update", "delete", "assign", "transfer", "close"}
}

# Emergency access check - allows bypassing some restrictions
is_emergency_access if {
    input.context.emergency == true
    input.context.emergency_type in {"life_threat", "child_protection"}
}

# Time-based access restriction
within_access_window if {
    # Check if current time is within allowed access window
    not input.resource.access_restricted
}

within_access_window if {
    input.resource.access_restricted
    time.now_ns() < input.resource.access_expires_at
}
