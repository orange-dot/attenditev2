# Dispatch access control policies
package dispatch.access

import future.keywords.if
import future.keywords.in
import data.common.roles
import data.common.helpers

default allow := false

# Platform admins can access all dispatch resources
allow if {
    roles.is_system_admin
}

# Dispatch operators can access incidents in their agency
allow if {
    roles.is_dispatch_operator
    input.actor_agency_id == input.resource.agency_id
}

# Dispatch operators can read incidents in their jurisdiction
allow if {
    roles.is_dispatch_operator
    input.action == "read"
    incident_in_jurisdiction(input.resource, input.actor_agency_id)
}

# Field units can access their assigned incidents
allow if {
    roles.is_field_unit
    input.action == "read"
    is_assigned_to_incident(input.actor_id, input.resource)
}

# Field units can update status of assigned incidents
allow if {
    roles.is_field_unit
    input.action == "update_status"
    is_assigned_to_incident(input.actor_id, input.resource)
}

# Agency admins can access all dispatch resources in their agency
allow if {
    roles.is_agency_admin
    input.actor_agency_id == input.resource.agency_id
}

# Check if incident is in agency's jurisdiction
incident_in_jurisdiction(incident, agency_id) if {
    incident.jurisdiction == agency_id
}

incident_in_jurisdiction(incident, agency_id) if {
    agency_id in incident.responding_agencies
}

# Check if user is assigned to incident
is_assigned_to_incident(user_id, incident) if {
    some unit in incident.assigned_units
    unit.operator_id == user_id
}

is_assigned_to_incident(user_id, incident) if {
    some responder in incident.responders
    responder.user_id == user_id
}

# Denial reasons
reasons[msg] if {
    not allow
    msg := "Access denied: not authorized for this dispatch resource"
}

reasons[msg] if {
    not input.actor_agency_id
    msg := "Access denied: no agency context"
}
