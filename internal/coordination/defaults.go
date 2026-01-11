package coordination

import (
	"time"
)

// DefaultProtocols returns the default coordination protocols for the system
func DefaultProtocols() []*Protocol {
	return []*Protocol{
		HospitalAdmissionProtocol(),
		HospitalDischargeProtocol(),
		ChildProtectionProtocol(),
		DomesticViolenceProtocol(),
		VulnerablePersonProtocol(),
		EmergencyProtocol(),
	}
}

// HospitalAdmissionProtocol handles hospital admission events
func HospitalAdmissionProtocol() *Protocol {
	return &Protocol{
		ID:          "hospital-admission",
		Name:        "Hospital Admission Notification",
		Description: "Notifies relevant social workers when their clients are admitted to hospital",
		TriggerType: EventTypeAdmission,
		Conditions: []Condition{
			{
				Field:    "has_open_cases",
				Operator: "eq",
				Value:    true,
			},
		},
		Actions: []Action{
			{
				Type:   "notify",
				Target: "assigned_social_worker",
				Parameters: map[string]any{
					"template":          "hospital_admission",
					"notification_type": "push",
				},
			},
			{
				Type: "route",
				Parameters: map[string]any{
					"targets": []any{"csr"},
				},
			},
		},
		Escalation: &Escalation{
			Levels: []EscalationLevel{
				{
					Level:        1,
					Timeout:      2 * time.Hour,
					Targets:      []string{"assigned_social_worker"},
					Notification: "Reminder: Hospital admission event pending acknowledgment",
				},
				{
					Level:        2,
					Timeout:      4 * time.Hour,
					Targets:      []string{"supervisor"},
					Notification: "Escalation: Hospital admission event not acknowledged by worker",
				},
				{
					Level:        3,
					Timeout:      8 * time.Hour,
					Targets:      []string{"department_head"},
					Notification: "Critical: Hospital admission event requires management attention",
				},
			},
			MaxLevel: 3,
		},
		Timeout:  8 * time.Hour,
		IsActive: true,
	}
}

// HospitalDischargeProtocol handles hospital discharge events
func HospitalDischargeProtocol() *Protocol {
	return &Protocol{
		ID:          "hospital-discharge",
		Name:        "Hospital Discharge Coordination",
		Description: "Coordinates follow-up care when vulnerable persons are discharged",
		TriggerType: EventTypeDischarge,
		Conditions: []Condition{
			{
				Field:    "is_beneficiary",
				Operator: "eq",
				Value:    true,
			},
		},
		Actions: []Action{
			{
				Type:   "notify",
				Target: "assigned_social_worker",
				Parameters: map[string]any{
					"template":          "hospital_discharge",
					"notification_type": "push",
				},
			},
			{
				Type: "add_target",
				Parameters: map[string]any{
					"agency": "csr",
				},
			},
		},
		Escalation: &Escalation{
			Levels: []EscalationLevel{
				{
					Level:        1,
					Timeout:      4 * time.Hour,
					Targets:      []string{"assigned_social_worker"},
					Notification: "Discharge follow-up pending",
				},
				{
					Level:        2,
					Timeout:      8 * time.Hour,
					Targets:      []string{"supervisor"},
					Notification: "Discharge coordination requires supervisor attention",
				},
			},
			MaxLevel: 2,
		},
		Timeout:  24 * time.Hour,
		IsActive: true,
	}
}

// ChildProtectionProtocol handles child protection concerns
func ChildProtectionProtocol() *Protocol {
	return &Protocol{
		ID:          "child-protection",
		Name:        "Child Protection Protocol",
		Description: "Immediate response protocol for child protection concerns",
		TriggerType: EventTypeChildProtection,
		Conditions:  []Condition{}, // Always trigger for child protection
		Actions: []Action{
			{
				Type: "set_priority",
				Parameters: map[string]any{
					"priority": "critical",
				},
			},
			{
				Type:   "notify",
				Target: "child_protection_team",
				Parameters: map[string]any{
					"template":          "child_protection",
					"notification_type": "push",
				},
			},
			{
				Type:   "notify",
				Target: "csr_duty_officer",
				Parameters: map[string]any{
					"template":          "child_protection",
					"notification_type": "sms",
				},
			},
			{
				Type: "route",
				Parameters: map[string]any{
					"targets": []any{"csr", "police_unit"},
				},
			},
		},
		Escalation: &Escalation{
			Levels: []EscalationLevel{
				{
					Level:        1,
					Timeout:      15 * time.Minute,
					Targets:      []string{"child_protection_team"},
					Notification: "URGENT: Child protection case requires immediate response",
				},
				{
					Level:        2,
					Timeout:      30 * time.Minute,
					Targets:      []string{"csr_director", "supervisor"},
					Notification: "CRITICAL: Child protection case not acknowledged",
				},
				{
					Level:        3,
					Timeout:      1 * time.Hour,
					Targets:      []string{"ministry_coordinator"},
					Notification: "EMERGENCY: Child protection case requires ministry intervention",
				},
			},
			MaxLevel: 3,
		},
		Timeout:  15 * time.Minute,
		IsActive: true,
	}
}

// DomesticViolenceProtocol handles domestic violence concerns
func DomesticViolenceProtocol() *Protocol {
	return &Protocol{
		ID:          "domestic-violence",
		Name:        "Domestic Violence Protocol",
		Description: "Response protocol for domestic violence situations",
		TriggerType: EventTypeDomesticViolence,
		Conditions:  []Condition{},
		Actions: []Action{
			{
				Type: "set_priority",
				Parameters: map[string]any{
					"priority": "critical",
				},
			},
			{
				Type:   "notify",
				Target: "dv_response_team",
				Parameters: map[string]any{
					"template":          "domestic_violence",
					"notification_type": "push",
				},
			},
			{
				Type:   "notify",
				Target: "csr_duty_officer",
				Parameters: map[string]any{
					"template":          "domestic_violence",
					"notification_type": "sms",
				},
			},
			{
				Type: "route",
				Parameters: map[string]any{
					"targets": []any{"csr", "police_dv_unit", "shelter_services"},
				},
			},
		},
		Escalation: &Escalation{
			Levels: []EscalationLevel{
				{
					Level:        1,
					Timeout:      15 * time.Minute,
					Targets:      []string{"dv_response_team"},
					Notification: "URGENT: Domestic violence case requires response",
				},
				{
					Level:        2,
					Timeout:      30 * time.Minute,
					Targets:      []string{"csr_director", "police_supervisor"},
					Notification: "CRITICAL: Domestic violence case not acknowledged",
				},
			},
			MaxLevel: 2,
		},
		Timeout:  15 * time.Minute,
		IsActive: true,
	}
}

// VulnerablePersonProtocol handles vulnerable person events
func VulnerablePersonProtocol() *Protocol {
	return &Protocol{
		ID:          "vulnerable-person",
		Name:        "Vulnerable Person Protocol",
		Description: "Coordination for vulnerable persons (elderly alone, disabled without support)",
		TriggerType: EventTypeVulnerablePerson,
		Conditions:  []Condition{},
		Actions: []Action{
			{
				Type:   "notify",
				Target: "assigned_social_worker",
				Parameters: map[string]any{
					"template":          "vulnerable_person",
					"notification_type": "push",
				},
			},
			{
				Type: "route",
				Parameters: map[string]any{
					"targets": []any{"csr"},
				},
			},
		},
		Escalation: &Escalation{
			Levels: []EscalationLevel{
				{
					Level:        1,
					Timeout:      2 * time.Hour,
					Targets:      []string{"assigned_social_worker"},
					Notification: "Vulnerable person case pending",
				},
				{
					Level:        2,
					Timeout:      4 * time.Hour,
					Targets:      []string{"supervisor"},
					Notification: "Vulnerable person case requires supervisor attention",
				},
			},
			MaxLevel: 2,
		},
		Timeout:  4 * time.Hour,
		IsActive: true,
	}
}

// EmergencyProtocol handles emergency events from health systems
func EmergencyProtocol() *Protocol {
	return &Protocol{
		ID:          "emergency",
		Name:        "Emergency Protocol",
		Description: "Immediate response for emergency cases",
		TriggerType: EventTypeEmergency,
		Conditions:  []Condition{},
		Actions: []Action{
			{
				Type: "set_priority",
				Parameters: map[string]any{
					"priority": "urgent",
				},
			},
			{
				Type:   "notify",
				Target: "emergency_coordinator",
				Parameters: map[string]any{
					"template":          "emergency",
					"notification_type": "push",
				},
			},
		},
		Escalation: &Escalation{
			Levels: []EscalationLevel{
				{
					Level:        1,
					Timeout:      30 * time.Minute,
					Targets:      []string{"emergency_coordinator"},
					Notification: "Emergency case requires coordination",
				},
				{
					Level:        2,
					Timeout:      1 * time.Hour,
					Targets:      []string{"supervisor"},
					Notification: "Emergency case escalated",
				},
			},
			MaxLevel: 2,
		},
		Timeout:  30 * time.Minute,
		IsActive: true,
	}
}

// RegisterDefaultProtocols registers all default protocols with the service
func RegisterDefaultProtocols(service *Service) error {
	for _, protocol := range DefaultProtocols() {
		if err := service.RegisterProtocol(protocol); err != nil {
			return err
		}
	}
	return nil
}
