// Package auth provides authentication and authorization types.
package auth

// Role represents a user role in the system.
type Role string

// System roles - global scope
const (
	RolePlatformAdmin    Role = "platform_admin"    // Full platform access
	RolePlatformOperator Role = "platform_operator" // Operations, monitoring
	RoleSecurityAuditor  Role = "security_auditor"  // Read-only audit access
)

// Agency roles - agency scope
const (
	RoleAgencyAdmin      Role = "agency_admin"      // Manage agency workers, settings
	RoleAgencySupervisor Role = "agency_supervisor" // Supervise workers, escalations
	RoleCaseWorker       Role = "case_worker"       // Handle cases
	RoleDispatchOperator Role = "dispatch_operator" // Dispatch console access
	RoleFieldUnit        Role = "field_unit"        // Mobile field worker
	RoleAgencyViewer     Role = "agency_viewer"     // Read-only agency access
)

// Case roles - per-case assignment
const (
	RoleCaseLead     Role = "lead"     // Primary responsible
	RoleCaseSupport  Role = "support"  // Assisting
	RoleCaseReviewer Role = "reviewer" // Approval authority
	RoleCaseObserver Role = "observer" // Read-only
)

// Citizen roles
const (
	RoleCitizen         Role = "citizen"          // Basic authenticated citizen
	RoleCitizenVerified Role = "citizen_verified" // eID-verified citizen
)

// Permission represents a specific action on a resource.
type Permission string

// Case permissions
const (
	PermCaseCreate   Permission = "case.create"
	PermCaseRead     Permission = "case.read"
	PermCaseUpdate   Permission = "case.update"
	PermCaseDelete   Permission = "case.delete"
	PermCaseAssign   Permission = "case.assign"
	PermCaseTransfer Permission = "case.transfer"
	PermCaseClose    Permission = "case.close"
	PermCaseEscalate Permission = "case.escalate"
)

// Dispatch permissions
const (
	PermIncidentCreate     Permission = "incident.create"
	PermIncidentRead       Permission = "incident.read"
	PermIncidentUpdate     Permission = "incident.update"
	PermIncidentClose      Permission = "incident.close"
	PermUnitDispatch       Permission = "unit.dispatch"
	PermUnitStatusUpdate   Permission = "unit.status.update"
	PermUnitLocationUpdate Permission = "unit.location.update"
)

// Document permissions
const (
	PermDocumentCreate Permission = "document.create"
	PermDocumentRead   Permission = "document.read"
	PermDocumentUpdate Permission = "document.update"
	PermDocumentDelete Permission = "document.delete"
	PermDocumentSign   Permission = "document.sign"
)

// Admin permissions
const (
	PermAgencyCreate  Permission = "agency.create"
	PermAgencyUpdate  Permission = "agency.update"
	PermAgencyDelete  Permission = "agency.delete"
	PermWorkerCreate  Permission = "worker.create"
	PermWorkerUpdate  Permission = "worker.update"
	PermWorkerDelete  Permission = "worker.delete"
	PermAuditRead     Permission = "audit.read"
	PermAuditExport   Permission = "audit.export"
	PermSensitiveData Permission = "sensitive_data_access"
)

// RolePermissions maps roles to their default permissions.
var RolePermissions = map[Role][]Permission{
	RolePlatformAdmin: {
		PermCaseCreate, PermCaseRead, PermCaseUpdate, PermCaseDelete,
		PermCaseAssign, PermCaseTransfer, PermCaseClose, PermCaseEscalate,
		PermIncidentCreate, PermIncidentRead, PermIncidentUpdate, PermIncidentClose,
		PermUnitDispatch, PermUnitStatusUpdate,
		PermDocumentCreate, PermDocumentRead, PermDocumentUpdate, PermDocumentDelete, PermDocumentSign,
		PermAgencyCreate, PermAgencyUpdate, PermAgencyDelete,
		PermWorkerCreate, PermWorkerUpdate, PermWorkerDelete,
		PermAuditRead, PermAuditExport, PermSensitiveData,
	},
	RoleAgencyAdmin: {
		PermCaseCreate, PermCaseRead, PermCaseUpdate,
		PermCaseAssign, PermCaseTransfer, PermCaseClose, PermCaseEscalate,
		PermIncidentCreate, PermIncidentRead, PermIncidentUpdate, PermIncidentClose,
		PermUnitDispatch, PermUnitStatusUpdate,
		PermDocumentCreate, PermDocumentRead, PermDocumentUpdate, PermDocumentDelete, PermDocumentSign,
		PermAgencyUpdate, PermWorkerCreate, PermWorkerUpdate, PermWorkerDelete,
	},
	RoleAgencySupervisor: {
		PermCaseCreate, PermCaseRead, PermCaseUpdate,
		PermCaseAssign, PermCaseTransfer, PermCaseClose, PermCaseEscalate,
		PermDocumentCreate, PermDocumentRead, PermDocumentUpdate, PermDocumentSign,
	},
	RoleCaseWorker: {
		PermCaseCreate, PermCaseRead, PermCaseUpdate,
		PermCaseClose, PermCaseEscalate,
		PermDocumentCreate, PermDocumentRead, PermDocumentUpdate, PermDocumentSign,
	},
	RoleDispatchOperator: {
		PermIncidentCreate, PermIncidentRead, PermIncidentUpdate, PermIncidentClose,
		PermUnitDispatch, PermUnitStatusUpdate,
	},
	RoleFieldUnit: {
		PermIncidentCreate, PermIncidentRead,
		PermUnitStatusUpdate, PermUnitLocationUpdate,
	},
	RoleAgencyViewer: {
		PermCaseRead, PermDocumentRead, PermIncidentRead,
	},
	RoleSecurityAuditor: {
		PermAuditRead, PermAuditExport,
	},
	RoleCitizen: {
		PermCaseCreate, PermDocumentCreate,
	},
	RoleCitizenVerified: {
		PermCaseCreate, PermDocumentCreate, PermDocumentSign,
	},
}

// AccessLevel represents cross-agency sharing levels.
type AccessLevel int

const (
	AccessNone       AccessLevel = 0 // Revoked
	AccessRead       AccessLevel = 1 // View case, documents
	AccessComment    AccessLevel = 2 // Add notes, messages
	AccessContribute AccessLevel = 3 // Update, assign own workers
	AccessFull       AccessLevel = 4 // All except transfer ownership
)

// DataClassification represents data sensitivity levels.
type DataClassification int

const (
	DataPublic       DataClassification = 0 // Agency contact info
	DataInternal     DataClassification = 1 // Case statistics
	DataConfidential DataClassification = 2 // Case details
	DataRestricted   DataClassification = 3 // Medical records
	DataSecret       DataClassification = 4 // Witness protection
)

// HasPermission checks if a role has a specific permission.
func HasPermission(role Role, perm Permission) bool {
	perms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the user has any of the specified roles.
func HasAnyRole(userRoles []Role, requiredRoles ...Role) bool {
	for _, ur := range userRoles {
		for _, rr := range requiredRoles {
			if ur == rr {
				return true
			}
		}
	}
	return false
}
