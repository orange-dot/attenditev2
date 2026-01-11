# Common role definitions for Serbia Government Platform
package common.roles

import future.keywords.if
import future.keywords.in

# System roles - global scope
system_roles := {
    "platform_admin",
    "platform_operator",
    "security_auditor"
}

# Agency roles - agency scope
agency_roles := {
    "agency_admin",
    "agency_supervisor",
    "case_worker",
    "dispatch_operator",
    "field_unit",
    "agency_viewer"
}

# Case roles - per-case assignment
case_roles := {
    "lead",
    "support",
    "reviewer",
    "observer"
}

# Citizen roles
citizen_roles := {
    "citizen",
    "citizen_verified"
}

# Check if user has system admin role
is_system_admin if {
    input.roles[_] == "platform_admin"
}

# Check if user has platform operator role
is_platform_operator if {
    input.roles[_] == "platform_operator"
}

# Check if user is security auditor
is_security_auditor if {
    input.roles[_] == "security_auditor"
}

# Check if user is agency admin
is_agency_admin if {
    input.roles[_] == "agency_admin"
}

# Check if user is agency supervisor
is_agency_supervisor if {
    input.roles[_] == "agency_supervisor"
}

# Check if user is case worker
is_case_worker if {
    input.roles[_] == "case_worker"
}

# Check if user is dispatch operator
is_dispatch_operator if {
    input.roles[_] == "dispatch_operator"
}

# Check if user is field unit
is_field_unit if {
    input.roles[_] == "field_unit"
}

# Check if user is verified citizen
is_verified_citizen if {
    input.roles[_] == "citizen_verified"
}

# Check if user has any admin role
has_admin_role if {
    some role in input.roles
    role in {"platform_admin", "agency_admin"}
}

# Check if user has supervisory role
has_supervisory_role if {
    some role in input.roles
    role in {"platform_admin", "agency_admin", "agency_supervisor"}
}

# Check if user can access sensitive data
can_access_sensitive_data if {
    "sensitive_data_access" in input.permissions
}

can_access_sensitive_data if {
    is_system_admin
}

# Role hierarchy - higher roles include lower role permissions
role_includes(higher, lower) if {
    higher == "platform_admin"
}

role_includes("agency_admin", "agency_supervisor") if true
role_includes("agency_admin", "case_worker") if true
role_includes("agency_admin", "agency_viewer") if true
role_includes("agency_supervisor", "case_worker") if true
role_includes("agency_supervisor", "agency_viewer") if true
role_includes("case_worker", "agency_viewer") if true
