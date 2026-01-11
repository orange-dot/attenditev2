package platform.document

import future.keywords.if
import future.keywords.in

# Default deny
default allow := false

# Allow if user is from owning agency
allow if {
    input.actor_agency_id == input.resource.owner_agency_id
}

# Allow read access if agency is in shared_with list
allow if {
    input.action == "read"
    input.actor_agency_id in input.resource.shared_with
}

# Platform admins can access all documents
allow if {
    "platform_admin" in input.roles
}

# Agency admins can access their agency's documents
allow if {
    "agency_admin" in input.roles
    input.actor_agency_id == input.resource.owner_agency_id
}

# Document creator can always access
allow if {
    input.actor_id == input.resource.created_by
}

# Signers can access documents they need to sign
allow if {
    input.action == "read"
    some sig in input.resource.signatures
    sig.signer_id == input.actor_id
    sig.status == "pending"
}

# Signers can sign documents assigned to them
allow if {
    input.action == "sign"
    some sig in input.resource.signatures
    sig.signer_id == input.actor_id
    sig.status == "pending"
}

# Case access implies document access for case documents
allow if {
    input.resource.case_id
    input.action == "read"
    # Would check case access here - simplified for MVP
}

# Prevent modification of archived/voided documents
deny if {
    input.action in ["write", "delete"]
    input.resource.status in ["archived", "void"]
}

# Reasons for denial
reasons[msg] if {
    not allow
    msg := "Access denied: not authorized for this document"
}

reasons[msg] if {
    deny
    input.resource.status == "archived"
    msg := "Access denied: document is archived"
}

reasons[msg] if {
    deny
    input.resource.status == "void"
    msg := "Access denied: document is voided"
}

# Field-level access
fields["signature_data"] if {
    # Only signers and admins can see signature data
    some sig in input.resource.signatures
    sig.signer_id == input.actor_id
}

fields["signature_data"] if {
    "platform_admin" in input.roles
}
