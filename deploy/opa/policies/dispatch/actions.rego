# Dispatch action policies
package dispatch.actions

import future.keywords.if
import future.keywords.in
import data.common.roles

default allow_dispatch := false
default allow_status_update := false
default allow_location_update := false

# Only dispatch operators can dispatch units
allow_dispatch if {
    roles.is_dispatch_operator
    input.actor_agency_id == input.unit.agency_id
    input.unit.status == "available"
}

# Platform admins can dispatch any unit
allow_dispatch if {
    roles.is_system_admin
}

# Dispatch coordinators can dispatch units from other agencies (multi-agency)
allow_dispatch if {
    "dispatch_coordinator" in input.roles
    # Cross-agency dispatch for coordinated response
}

# Can dispatch to incidents in jurisdiction
allow_dispatch if {
    roles.is_dispatch_operator
    incident_in_jurisdiction(input.incident, input.actor_agency_id)
    input.unit.status == "available"
}

# Unit status update permissions
allow_status_update if {
    roles.is_dispatch_operator
    input.actor_agency_id == input.unit.agency_id
}

allow_status_update if {
    roles.is_field_unit
    input.actor_id == input.unit.operator_id
}

allow_status_update if {
    roles.is_system_admin
}

# Location update - only field units can update their own location
allow_location_update if {
    roles.is_field_unit
    input.actor_id == input.unit.operator_id
}

# Check if incident is in agency's jurisdiction
incident_in_jurisdiction(incident, agency_id) if {
    incident.jurisdiction == agency_id
}

incident_in_jurisdiction(incident, agency_id) if {
    agency_id in incident.responding_agencies
}

# Valid unit statuses for dispatch
valid_dispatch_statuses := {"available", "on_break"}

# Check if unit can be dispatched
can_dispatch_unit(unit) if {
    unit.status in valid_dispatch_statuses
    unit.is_active == true
}

# Priority-based dispatch rules
priority_allows_preempt(incident_priority, unit_current_priority) if {
    priority_value(incident_priority) > priority_value(unit_current_priority)
}

priority_value("critical") := 4
priority_value("high") := 3
priority_value("medium") := 2
priority_value("low") := 1
priority_value(_) := 0

# Denial reasons
reasons[msg] if {
    not allow_dispatch
    input.unit.status != "available"
    msg := sprintf("Cannot dispatch: unit status is %s", [input.unit.status])
}

reasons[msg] if {
    not allow_dispatch
    not roles.is_dispatch_operator
    not roles.is_system_admin
    msg := "Cannot dispatch: insufficient permissions"
}
