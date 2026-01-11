package fhir

import (
	"fmt"
	"time"

	"github.com/serbia-gov/platform/internal/adapters/health"
)

// Encounter represents a FHIR R4 Encounter resource (simplified)
type Encounter struct {
	ResourceType   string            `json:"resourceType"`
	ID             string            `json:"id,omitempty"`
	Meta           *Meta             `json:"meta,omitempty"`
	Identifier     []Identifier      `json:"identifier,omitempty"`
	Status         string            `json:"status"` // planned, arrived, triaged, in-progress, onleave, finished, cancelled
	Class          *Coding           `json:"class"`
	Type           []CodeableConcept `json:"type,omitempty"`
	Subject        *Reference        `json:"subject,omitempty"`
	Participant    []Participant     `json:"participant,omitempty"`
	Period         *Period           `json:"period,omitempty"`
	ReasonCode     []CodeableConcept `json:"reasonCode,omitempty"`
	Diagnosis      []EncounterDiagnosis `json:"diagnosis,omitempty"`
	Hospitalization *Hospitalization   `json:"hospitalization,omitempty"`
	Location       []EncounterLocation `json:"location,omitempty"`
	ServiceProvider *Reference        `json:"serviceProvider,omitempty"`
}

// Reference represents a FHIR Reference
type Reference struct {
	Reference  string `json:"reference,omitempty"`
	Type       string `json:"type,omitempty"`
	Identifier *Identifier `json:"identifier,omitempty"`
	Display    string `json:"display,omitempty"`
}

// Period represents a FHIR Period
type Period struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

// Participant represents an encounter participant
type Participant struct {
	Type       []CodeableConcept `json:"type,omitempty"`
	Period     *Period           `json:"period,omitempty"`
	Individual *Reference        `json:"individual,omitempty"`
}

// EncounterDiagnosis represents a diagnosis in an encounter
type EncounterDiagnosis struct {
	Condition *Reference       `json:"condition,omitempty"`
	Use       *CodeableConcept `json:"use,omitempty"`
	Rank      int              `json:"rank,omitempty"`
}

// Hospitalization represents hospitalization details
type Hospitalization struct {
	PreAdmissionIdentifier *Identifier      `json:"preAdmissionIdentifier,omitempty"`
	Origin                 *Reference       `json:"origin,omitempty"`
	AdmitSource            *CodeableConcept `json:"admitSource,omitempty"`
	ReAdmission            *CodeableConcept `json:"reAdmission,omitempty"`
	DischargeDisposition   *CodeableConcept `json:"dischargeDisposition,omitempty"`
	Destination            *Reference       `json:"destination,omitempty"`
}

// EncounterLocation represents a location during encounter
type EncounterLocation struct {
	Location *Reference `json:"location,omitempty"`
	Status   string     `json:"status,omitempty"` // planned, active, reserved, completed
	Period   *Period    `json:"period,omitempty"`
}

// Encounter class codes (V3 ActEncounterCode)
const (
	EncounterClassInpatient   = "IMP"
	EncounterClassOutpatient  = "AMB"
	EncounterClassEmergency   = "EMER"
	EncounterClassShortStay   = "SS"
	EncounterClassHomeHealth  = "HH"
	EncounterClassVirtual     = "VR"
)

// Serbian-specific systems
const (
	SerbianEncounterProfile = "https://fhir.srbija.gov.rs/StructureDefinition/serbian-encounter"
	SerbianDepartmentSystem = "https://fhir.srbija.gov.rs/CodeSystem/department"
	ICD10System            = "http://hl7.org/fhir/sid/icd-10"
)

// ToFHIREncounter converts a health.Hospitalization to a FHIR Encounter
func ToFHIREncounter(hosp *health.Hospitalization) *Encounter {
	if hosp == nil {
		return nil
	}

	encounter := &Encounter{
		ResourceType: "Encounter",
		ID:           hosp.ID,
		Meta: &Meta{
			LastUpdated: hosp.LastUpdated.Format(time.RFC3339),
			Source:      fmt.Sprintf("%s#%s", hosp.SourceSystem, hosp.SourceInstitution),
			Profile:     []string{SerbianEncounterProfile},
		},
		Identifier: []Identifier{
			{
				System: LocalIDSystem,
				Value:  hosp.ID,
			},
		},
		Subject: &Reference{
			Identifier: &Identifier{
				System: JMBGSystem,
				Value:  hosp.PatientJMBG,
			},
		},
	}

	// Set status based on hospitalization status
	switch hosp.Status {
	case "active":
		encounter.Status = "in-progress"
	case "discharged":
		encounter.Status = "finished"
	case "transferred":
		encounter.Status = "finished"
	default:
		encounter.Status = "unknown"
	}

	// Set class based on admission type
	encounter.Class = mapAdmissionTypeToClass(hosp.AdmissionType)

	// Set period
	encounter.Period = &Period{
		Start: hosp.AdmissionDate.Format(time.RFC3339),
	}
	if hosp.DischargeDate != nil {
		encounter.Period.End = hosp.DischargeDate.Format(time.RFC3339)
	}

	// Add type (department)
	if hosp.DepartmentCode != "" || hosp.Department != "" {
		encounter.Type = []CodeableConcept{
			{
				Coding: []Coding{
					{
						System:  SerbianDepartmentSystem,
						Code:    hosp.DepartmentCode,
						Display: hosp.Department,
					},
				},
				Text: hosp.Department,
			},
		}
	}

	// Add attending doctor as participant
	if hosp.AttendingDoctor != "" {
		encounter.Participant = append(encounter.Participant, Participant{
			Type: []CodeableConcept{
				{
					Coding: []Coding{
						{
							System:  "http://terminology.hl7.org/CodeSystem/v3-ParticipationType",
							Code:    "ATND",
							Display: "attender",
						},
					},
				},
			},
			Individual: &Reference{
				Display: hosp.AttendingDoctor,
				Identifier: &Identifier{
					Value: hosp.AttendingDoctorID,
				},
			},
		})
	}

	// Add primary diagnosis
	if hosp.PrimaryDiagnosis != nil {
		encounter.Diagnosis = append(encounter.Diagnosis, EncounterDiagnosis{
			Condition: &Reference{
				Display: hosp.PrimaryDiagnosis.Description,
			},
			Use: &CodeableConcept{
				Coding: []Coding{
					{
						System:  "http://terminology.hl7.org/CodeSystem/diagnosis-role",
						Code:    "AD",
						Display: "Admission diagnosis",
					},
				},
			},
			Rank: 1,
		})

		// Add ICD-10 to reason code
		encounter.ReasonCode = append(encounter.ReasonCode, CodeableConcept{
			Coding: []Coding{
				{
					System:  ICD10System,
					Code:    hosp.PrimaryDiagnosis.ICD10Code,
					Display: hosp.PrimaryDiagnosis.Description,
				},
			},
		})
	}

	// Add secondary diagnoses
	for i, diag := range hosp.SecondaryDiagnoses {
		encounter.Diagnosis = append(encounter.Diagnosis, EncounterDiagnosis{
			Condition: &Reference{
				Display: diag.Description,
			},
			Use: &CodeableConcept{
				Coding: []Coding{
					{
						System:  "http://terminology.hl7.org/CodeSystem/diagnosis-role",
						Code:    "CC",
						Display: "Chief complaint",
					},
				},
			},
			Rank: i + 2,
		})
	}

	// Add discharge diagnosis
	if hosp.DischargeDiagnosis != nil {
		encounter.Diagnosis = append(encounter.Diagnosis, EncounterDiagnosis{
			Condition: &Reference{
				Display: hosp.DischargeDiagnosis.Description,
			},
			Use: &CodeableConcept{
				Coding: []Coding{
					{
						System:  "http://terminology.hl7.org/CodeSystem/diagnosis-role",
						Code:    "DD",
						Display: "Discharge diagnosis",
					},
				},
			},
		})
	}

	// Add hospitalization details
	if hosp.DischargeType != "" {
		encounter.Hospitalization = &Hospitalization{
			AdmitSource: mapAdmitSource(hosp.AdmissionType),
			DischargeDisposition: mapDischargeDisposition(hosp.DischargeType),
		}
	}

	// Add location (room/bed)
	if hosp.Room != "" || hosp.Bed != "" {
		location := &Reference{}
		if hosp.Bed != "" {
			location.Display = fmt.Sprintf("%s - %s", hosp.Room, hosp.Bed)
		} else {
			location.Display = hosp.Room
		}
		encounter.Location = append(encounter.Location, EncounterLocation{
			Location: location,
			Status:   "active",
		})
	}

	// Add service provider (institution)
	encounter.ServiceProvider = &Reference{
		Display: hosp.SourceInstitution,
	}

	return encounter
}

// mapAdmissionTypeToClass maps admission type to FHIR encounter class
func mapAdmissionTypeToClass(admissionType string) *Coding {
	switch admissionType {
	case "emergency":
		return &Coding{
			System:  "http://terminology.hl7.org/CodeSystem/v3-ActCode",
			Code:    EncounterClassEmergency,
			Display: "emergency",
		}
	case "planned", "elective":
		return &Coding{
			System:  "http://terminology.hl7.org/CodeSystem/v3-ActCode",
			Code:    EncounterClassInpatient,
			Display: "inpatient encounter",
		}
	case "transfer":
		return &Coding{
			System:  "http://terminology.hl7.org/CodeSystem/v3-ActCode",
			Code:    EncounterClassInpatient,
			Display: "inpatient encounter",
		}
	default:
		return &Coding{
			System:  "http://terminology.hl7.org/CodeSystem/v3-ActCode",
			Code:    EncounterClassInpatient,
			Display: "inpatient encounter",
		}
	}
}

// mapAdmitSource maps admission type to admit source
func mapAdmitSource(admissionType string) *CodeableConcept {
	switch admissionType {
	case "emergency":
		return &CodeableConcept{
			Coding: []Coding{
				{
					System:  "http://terminology.hl7.org/CodeSystem/admit-source",
					Code:    "emd",
					Display: "From accident/emergency department",
				},
			},
		}
	case "transfer":
		return &CodeableConcept{
			Coding: []Coding{
				{
					System:  "http://terminology.hl7.org/CodeSystem/admit-source",
					Code:    "hosp-trans",
					Display: "Transferred from other hospital",
				},
			},
		}
	default:
		return &CodeableConcept{
			Coding: []Coding{
				{
					System:  "http://terminology.hl7.org/CodeSystem/admit-source",
					Code:    "gp",
					Display: "General Practitioner referral",
				},
			},
		}
	}
}

// mapDischargeDisposition maps discharge type to FHIR disposition
func mapDischargeDisposition(dischargeType string) *CodeableConcept {
	switch dischargeType {
	case "home":
		return &CodeableConcept{
			Coding: []Coding{
				{
					System:  "http://terminology.hl7.org/CodeSystem/discharge-disposition",
					Code:    "home",
					Display: "Home",
				},
			},
		}
	case "transfer":
		return &CodeableConcept{
			Coding: []Coding{
				{
					System:  "http://terminology.hl7.org/CodeSystem/discharge-disposition",
					Code:    "other-hcf",
					Display: "Other healthcare facility",
				},
			},
		}
	case "deceased":
		return &CodeableConcept{
			Coding: []Coding{
				{
					System:  "http://terminology.hl7.org/CodeSystem/discharge-disposition",
					Code:    "exp",
					Display: "Expired",
				},
			},
		}
	default:
		return &CodeableConcept{
			Coding: []Coding{
				{
					System:  "http://terminology.hl7.org/CodeSystem/discharge-disposition",
					Code:    "oth",
					Display: "Other",
				},
			},
		}
	}
}
