package fhir

import (
	"fmt"
	"time"

	"github.com/serbia-gov/platform/internal/adapters/health"
)

// Patient represents a FHIR R4 Patient resource (simplified)
type Patient struct {
	ResourceType string              `json:"resourceType"`
	ID           string              `json:"id,omitempty"`
	Meta         *Meta               `json:"meta,omitempty"`
	Identifier   []Identifier        `json:"identifier,omitempty"`
	Active       bool                `json:"active"`
	Name         []HumanName         `json:"name,omitempty"`
	Telecom      []ContactPoint      `json:"telecom,omitempty"`
	Gender       string              `json:"gender,omitempty"`
	BirthDate    string              `json:"birthDate,omitempty"`
	Deceased     *DeceasedInfo       `json:"deceasedBoolean,omitempty"`
	Address      []Address           `json:"address,omitempty"`
}

// Meta holds resource metadata
type Meta struct {
	VersionID   string   `json:"versionId,omitempty"`
	LastUpdated string   `json:"lastUpdated,omitempty"`
	Source      string   `json:"source,omitempty"`
	Profile     []string `json:"profile,omitempty"`
}

// Identifier represents a FHIR Identifier
type Identifier struct {
	Use    string `json:"use,omitempty"` // usual, official, temp, secondary
	Type   *CodeableConcept `json:"type,omitempty"`
	System string `json:"system,omitempty"`
	Value  string `json:"value,omitempty"`
}

// CodeableConcept represents a FHIR CodeableConcept
type CodeableConcept struct {
	Coding []Coding `json:"coding,omitempty"`
	Text   string   `json:"text,omitempty"`
}

// Coding represents a FHIR Coding
type Coding struct {
	System  string `json:"system,omitempty"`
	Version string `json:"version,omitempty"`
	Code    string `json:"code,omitempty"`
	Display string `json:"display,omitempty"`
}

// HumanName represents a FHIR HumanName
type HumanName struct {
	Use    string   `json:"use,omitempty"` // usual, official, temp, nickname, anonymous, old, maiden
	Family string   `json:"family,omitempty"`
	Given  []string `json:"given,omitempty"`
	Prefix []string `json:"prefix,omitempty"`
	Suffix []string `json:"suffix,omitempty"`
}

// ContactPoint represents a FHIR ContactPoint
type ContactPoint struct {
	System string `json:"system,omitempty"` // phone, fax, email, pager, url, sms, other
	Value  string `json:"value,omitempty"`
	Use    string `json:"use,omitempty"` // home, work, temp, old, mobile
	Rank   int    `json:"rank,omitempty"`
}

// Address represents a FHIR Address
type Address struct {
	Use        string   `json:"use,omitempty"` // home, work, temp, old, billing
	Type       string   `json:"type,omitempty"` // postal, physical, both
	Text       string   `json:"text,omitempty"`
	Line       []string `json:"line,omitempty"`
	City       string   `json:"city,omitempty"`
	District   string   `json:"district,omitempty"`
	State      string   `json:"state,omitempty"`
	PostalCode string   `json:"postalCode,omitempty"`
	Country    string   `json:"country,omitempty"`
}

// DeceasedInfo represents deceased status
type DeceasedInfo struct {
	Boolean  bool   `json:"deceasedBoolean,omitempty"`
	DateTime string `json:"deceasedDateTime,omitempty"`
}

// Serbian-specific identifier systems
const (
	JMBGSystem = "https://fhir.srbija.gov.rs/sid/jmbg"
	LBOSystem  = "https://fhir.srbija.gov.rs/sid/lbo"
	LocalIDSystem = "https://fhir.srbija.gov.rs/sid/local"

	SerbianPatientProfile = "https://fhir.srbija.gov.rs/StructureDefinition/serbian-patient"
)

// ToFHIRPatient converts a health.PatientRecord to a FHIR Patient
func ToFHIRPatient(record *health.PatientRecord) *Patient {
	if record == nil {
		return nil
	}

	patient := &Patient{
		ResourceType: "Patient",
		ID:           record.LocalID,
		Active:       !record.Deceased,
		Meta: &Meta{
			LastUpdated: record.LastUpdated.Format(time.RFC3339),
			Source:      fmt.Sprintf("%s#%s", record.SourceSystem, record.SourceInstitution),
			Profile:     []string{SerbianPatientProfile},
		},
		Identifier: make([]Identifier, 0),
		Name: []HumanName{
			{
				Use:    "official",
				Family: record.LastName,
				Given:  []string{record.FirstName},
			},
		},
		BirthDate: record.DateOfBirth.Format("2006-01-02"),
	}

	// Add middle name if present
	if record.MiddleName != "" {
		patient.Name[0].Given = append(patient.Name[0].Given, record.MiddleName)
	}

	// Add JMBG identifier (required)
	patient.Identifier = append(patient.Identifier, Identifier{
		Use:    "official",
		System: JMBGSystem,
		Value:  record.JMBG,
		Type: &CodeableConcept{
			Coding: []Coding{
				{
					System:  "http://terminology.hl7.org/CodeSystem/v2-0203",
					Code:    "NI",
					Display: "National unique individual identifier",
				},
			},
			Text: "JMBG",
		},
	})

	// Add LBO identifier if present
	if record.LBO != "" {
		patient.Identifier = append(patient.Identifier, Identifier{
			Use:    "secondary",
			System: LBOSystem,
			Value:  record.LBO,
			Type: &CodeableConcept{
				Coding: []Coding{
					{
						System:  "http://terminology.hl7.org/CodeSystem/v2-0203",
						Code:    "HC",
						Display: "Health Card Number",
					},
				},
				Text: "LBO",
			},
		})
	}

	// Add local ID
	patient.Identifier = append(patient.Identifier, Identifier{
		Use:    "usual",
		System: LocalIDSystem,
		Value:  record.LocalID,
	})

	// Set gender
	switch record.Gender {
	case health.GenderMale:
		patient.Gender = "male"
	case health.GenderFemale:
		patient.Gender = "female"
	case health.GenderOther:
		patient.Gender = "other"
	default:
		patient.Gender = "unknown"
	}

	// Add telecom
	if record.Phone != "" {
		patient.Telecom = append(patient.Telecom, ContactPoint{
			System: "phone",
			Value:  record.Phone,
			Use:    "home",
		})
	}
	if record.Email != "" {
		patient.Telecom = append(patient.Telecom, ContactPoint{
			System: "email",
			Value:  record.Email,
		})
	}

	// Add address
	if record.Address != "" || record.City != "" {
		addr := Address{
			Use:  "home",
			Type: "physical",
		}
		if record.Address != "" {
			addr.Line = []string{record.Address}
		}
		if record.City != "" {
			addr.City = record.City
		}
		if record.PostalCode != "" {
			addr.PostalCode = record.PostalCode
		}
		addr.Country = "RS" // Serbia ISO code
		patient.Address = append(patient.Address, addr)
	}

	// Handle deceased status
	if record.Deceased {
		patient.Deceased = &DeceasedInfo{Boolean: true}
		if record.DeceasedAt != nil {
			patient.Deceased.DateTime = record.DeceasedAt.Format(time.RFC3339)
		}
	}

	return patient
}
