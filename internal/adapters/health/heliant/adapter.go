package heliant

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/denisenkom/go-mssqldb" // SQL Server driver
	"github.com/serbia-gov/platform/internal/adapters/health"
)

// Adapter implements health.Adapter for Heliant HIS
type Adapter struct {
	db           *sql.DB
	config       Config

	// Event channels
	admissionChan chan health.AdmissionEvent
	dischargeChan chan health.DischargeEvent
	emergencyChan chan health.EmergencyEvent

	// State
	running    bool
	mu         sync.RWMutex
	cancel     context.CancelFunc
	lastPoll   time.Time
	wg         sync.WaitGroup
}

// Config holds Heliant adapter configuration
type Config struct {
	health.Config

	// Heliant-specific settings
	PatientTable        string `json:"patient_table"`
	HospitalizationTable string `json:"hospitalization_table"`
	LabResultTable      string `json:"lab_result_table"`
	PrescriptionTable   string `json:"prescription_table"`
	DiagnosisTable      string `json:"diagnosis_table"`
}

// DefaultHeliantConfig returns default Heliant configuration
func DefaultHeliantConfig() Config {
	return Config{
		Config:               health.DefaultConfig(),
		PatientTable:        "dbo.Patients",
		HospitalizationTable: "dbo.Hospitalizations",
		LabResultTable:      "dbo.LabResults",
		PrescriptionTable:   "dbo.Prescriptions",
		DiagnosisTable:      "dbo.Diagnoses",
	}
}

// New creates a new Heliant adapter
func New(cfg Config) (*Adapter, error) {
	return &Adapter{
		config:        cfg,
		admissionChan: make(chan health.AdmissionEvent, cfg.EventBufferSize),
		dischargeChan: make(chan health.DischargeEvent, cfg.EventBufferSize),
		emergencyChan: make(chan health.EmergencyEvent, cfg.EventBufferSize),
	}, nil
}

// Start initializes the database connection and starts polling
func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return fmt.Errorf("adapter already running")
	}

	// Build connection string
	connStr := fmt.Sprintf("server=%s;port=%d;database=%s;user id=%s;password=%s",
		a.config.Host,
		a.config.Port,
		a.config.Database,
		a.config.User,
		a.config.Password,
	)

	if a.config.SSLMode != "disable" {
		connStr += ";encrypt=true;TrustServerCertificate=true"
	}

	// Connect to database
	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	a.db = db
	a.running = true
	a.lastPoll = time.Now().Add(-a.config.PollInterval)

	// Start polling goroutine
	pollCtx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	a.wg.Add(1)
	go a.pollLoop(pollCtx)

	return nil
}

// Stop stops the adapter and closes connections
func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	// Cancel polling
	if a.cancel != nil {
		a.cancel()
	}

	// Wait for goroutines
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Close channels
	close(a.admissionChan)
	close(a.dischargeChan)
	close(a.emergencyChan)

	// Close database
	if a.db != nil {
		a.db.Close()
	}

	a.running = false
	return nil
}

// Health checks database connectivity
func (a *Adapter) Health(ctx context.Context) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.running {
		return fmt.Errorf("adapter not running")
	}

	return a.db.PingContext(ctx)
}

// SourceSystem returns the source system name
func (a *Adapter) SourceSystem() string {
	return "heliant"
}

// SourceInstitution returns the institution name
func (a *Adapter) SourceInstitution() string {
	return a.config.InstitutionName
}

// IsConnected returns connection status
func (a *Adapter) IsConnected() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running && a.db != nil
}

// FetchPatientRecord retrieves patient data by JMBG
func (a *Adapter) FetchPatientRecord(ctx context.Context, jmbg string) (*health.PatientRecord, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("adapter not connected")
	}

	query := fmt.Sprintf(`
		SELECT
			PatientID,
			JMBG,
			LBO,
			FirstName,
			LastName,
			MiddleName,
			DateOfBirth,
			Gender,
			Address,
			City,
			PostalCode,
			Phone,
			Email,
			InsuranceStatus,
			InsuranceType,
			IsDeceased,
			DeceasedDate,
			LastModified
		FROM %s
		WHERE JMBG = @jmbg
	`, a.config.PatientTable)

	row := a.db.QueryRowContext(ctx, query, sql.Named("jmbg", jmbg))

	var record health.PatientRecord
	var genderCode string
	var deceased sql.NullBool
	var deceasedAt sql.NullTime
	var middleName, lbo, email sql.NullString
	var insStatus, insType sql.NullString

	err := row.Scan(
		&record.LocalID,
		&record.JMBG,
		&lbo,
		&record.FirstName,
		&record.LastName,
		&middleName,
		&record.DateOfBirth,
		&genderCode,
		&record.Address,
		&record.City,
		&record.PostalCode,
		&record.Phone,
		&email,
		&insStatus,
		&insType,
		&deceased,
		&deceasedAt,
		&record.LastUpdated,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("patient not found: %s", jmbg)
		}
		return nil, fmt.Errorf("failed to fetch patient: %w", err)
	}

	// Map nullable fields
	if lbo.Valid {
		record.LBO = lbo.String
	}
	if middleName.Valid {
		record.MiddleName = middleName.String
	}
	if email.Valid {
		record.Email = email.String
	}
	if insStatus.Valid {
		record.InsuranceStatus = insStatus.String
	}
	if insType.Valid {
		record.InsuranceType = insType.String
	}
	if deceased.Valid {
		record.Deceased = deceased.Bool
	}
	if deceasedAt.Valid {
		record.DeceasedAt = &deceasedAt.Time
	}

	// Map gender
	record.Gender = mapGender(genderCode)

	// Set source
	record.SourceSystem = a.SourceSystem()
	record.SourceInstitution = a.SourceInstitution()

	return &record, nil
}

// FetchPatientByLBO retrieves patient data by LBO
func (a *Adapter) FetchPatientByLBO(ctx context.Context, lbo string) (*health.PatientRecord, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("adapter not connected")
	}

	// First get JMBG from LBO
	query := fmt.Sprintf(`SELECT JMBG FROM %s WHERE LBO = @lbo`, a.config.PatientTable)
	var jmbg string
	err := a.db.QueryRowContext(ctx, query, sql.Named("lbo", lbo)).Scan(&jmbg)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("patient not found by LBO: %s", lbo)
		}
		return nil, err
	}

	return a.FetchPatientRecord(ctx, jmbg)
}

// FetchHospitalizations retrieves hospitalizations for a patient
func (a *Adapter) FetchHospitalizations(ctx context.Context, jmbg string, from, to time.Time) ([]health.Hospitalization, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("adapter not connected")
	}

	query := fmt.Sprintf(`
		SELECT
			h.HospitalizationID,
			p.JMBG,
			h.AdmissionDate,
			h.DischargeDate,
			h.Department,
			h.DepartmentCode,
			h.Room,
			h.Bed,
			h.AdmissionType,
			h.DischargeType,
			h.AttendingDoctor,
			h.AttendingDoctorID,
			h.PrimaryDiagnosisICD,
			h.PrimaryDiagnosisText,
			h.Status,
			h.LastModified
		FROM %s h
		INNER JOIN %s p ON h.PatientID = p.PatientID
		WHERE p.JMBG = @jmbg
		  AND h.AdmissionDate >= @from
		  AND h.AdmissionDate <= @to
		ORDER BY h.AdmissionDate DESC
	`, a.config.HospitalizationTable, a.config.PatientTable)

	rows, err := a.db.QueryContext(ctx, query,
		sql.Named("jmbg", jmbg),
		sql.Named("from", from),
		sql.Named("to", to),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query hospitalizations: %w", err)
	}
	defer rows.Close()

	var hospitalizations []health.Hospitalization
	for rows.Next() {
		var h health.Hospitalization
		var dischargeDate sql.NullTime
		var room, bed, dischargeType sql.NullString
		var doctorID, deptCode sql.NullString
		var diagICD, diagText sql.NullString

		err := rows.Scan(
			&h.ID,
			&h.PatientJMBG,
			&h.AdmissionDate,
			&dischargeDate,
			&h.Department,
			&deptCode,
			&room,
			&bed,
			&h.AdmissionType,
			&dischargeType,
			&h.AttendingDoctor,
			&doctorID,
			&diagICD,
			&diagText,
			&h.Status,
			&h.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan hospitalization: %w", err)
		}

		// Map nullable fields
		if dischargeDate.Valid {
			h.DischargeDate = &dischargeDate.Time
		}
		if room.Valid {
			h.Room = room.String
		}
		if bed.Valid {
			h.Bed = bed.String
		}
		if dischargeType.Valid {
			h.DischargeType = dischargeType.String
		}
		if doctorID.Valid {
			h.AttendingDoctorID = doctorID.String
		}
		if deptCode.Valid {
			h.DepartmentCode = deptCode.String
		}

		// Map primary diagnosis
		if diagICD.Valid && diagText.Valid {
			h.PrimaryDiagnosis = &health.Diagnosis{
				ICD10Code:   diagICD.String,
				Description: diagText.String,
				Type:        "primary",
				DiagnosedAt: h.AdmissionDate,
			}
		}

		h.SourceSystem = a.SourceSystem()
		h.SourceInstitution = a.SourceInstitution()

		hospitalizations = append(hospitalizations, h)
	}

	return hospitalizations, nil
}

// FetchLabResults retrieves lab results for a patient
func (a *Adapter) FetchLabResults(ctx context.Context, jmbg string, from, to time.Time) ([]health.LabResult, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("adapter not connected")
	}

	query := fmt.Sprintf(`
		SELECT
			l.LabResultID,
			p.JMBG,
			l.TestCode,
			l.TestName,
			l.LOINCCode,
			l.Value,
			l.Unit,
			l.ReferenceMin,
			l.ReferenceMax,
			l.Interpretation,
			l.CollectedAt,
			l.ReportedAt,
			l.OrderedBy,
			l.Laboratory,
			l.Notes,
			l.LastModified
		FROM %s l
		INNER JOIN %s p ON l.PatientID = p.PatientID
		WHERE p.JMBG = @jmbg
		  AND l.CollectedAt >= @from
		  AND l.CollectedAt <= @to
		ORDER BY l.CollectedAt DESC
	`, a.config.LabResultTable, a.config.PatientTable)

	rows, err := a.db.QueryContext(ctx, query,
		sql.Named("jmbg", jmbg),
		sql.Named("from", from),
		sql.Named("to", to),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query lab results: %w", err)
	}
	defer rows.Close()

	var results []health.LabResult
	for rows.Next() {
		var r health.LabResult
		var loinc, unit, refMin, refMax, interp, orderedBy, lab, notes sql.NullString

		err := rows.Scan(
			&r.ID,
			&r.PatientJMBG,
			&r.TestCode,
			&r.TestName,
			&loinc,
			&r.Value,
			&unit,
			&refMin,
			&refMax,
			&interp,
			&r.CollectedAt,
			&r.ReportedAt,
			&orderedBy,
			&lab,
			&notes,
			&r.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lab result: %w", err)
		}

		// Map nullable fields
		if loinc.Valid {
			r.LOINCCode = loinc.String
		}
		if unit.Valid {
			r.Unit = unit.String
		}
		if refMin.Valid {
			r.ReferenceMin = refMin.String
		}
		if refMax.Valid {
			r.ReferenceMax = refMax.String
		}
		if interp.Valid {
			r.Interpretation = interp.String
		}
		if orderedBy.Valid {
			r.OrderedBy = orderedBy.String
		}
		if lab.Valid {
			r.Laboratory = lab.String
		}
		if notes.Valid {
			r.Notes = notes.String
		}

		r.SourceSystem = a.SourceSystem()
		r.SourceInstitution = a.SourceInstitution()

		results = append(results, r)
	}

	return results, nil
}

// FetchPrescriptions retrieves prescriptions for a patient
func (a *Adapter) FetchPrescriptions(ctx context.Context, jmbg string, activeOnly bool) ([]health.Prescription, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("adapter not connected")
	}

	query := fmt.Sprintf(`
		SELECT
			r.PrescriptionID,
			p.JMBG,
			r.MedicationName,
			r.MedicationCode,
			r.ATCCode,
			r.Dosage,
			r.DosageUnit,
			r.Frequency,
			r.Route,
			r.Duration,
			r.Quantity,
			r.Refills,
			r.PrescribedAt,
			r.PrescribedBy,
			r.PrescribedByID,
			r.ValidUntil,
			r.DispensedAt,
			r.DispensedBy,
			r.Status,
			r.Instructions,
			r.DiagnosisICD10,
			r.IsChronicMed,
			r.LastModified
		FROM %s r
		INNER JOIN %s p ON r.PatientID = p.PatientID
		WHERE p.JMBG = @jmbg
	`, a.config.PrescriptionTable, a.config.PatientTable)

	if activeOnly {
		query += ` AND r.Status = 'active' AND (r.ValidUntil IS NULL OR r.ValidUntil > GETDATE())`
	}

	query += ` ORDER BY r.PrescribedAt DESC`

	rows, err := a.db.QueryContext(ctx, query, sql.Named("jmbg", jmbg))
	if err != nil {
		return nil, fmt.Errorf("failed to query prescriptions: %w", err)
	}
	defer rows.Close()

	var prescriptions []health.Prescription
	for rows.Next() {
		var rx health.Prescription
		var medCode, atc, dosageUnit, route, duration sql.NullString
		var prescribedByID, dispensedBy, instructions, diagICD sql.NullString
		var validUntil, dispensedAt sql.NullTime
		var quantity, refills sql.NullInt32

		err := rows.Scan(
			&rx.ID,
			&rx.PatientJMBG,
			&rx.MedicationName,
			&medCode,
			&atc,
			&rx.Dosage,
			&dosageUnit,
			&rx.Frequency,
			&route,
			&duration,
			&quantity,
			&refills,
			&rx.PrescribedAt,
			&rx.PrescribedBy,
			&prescribedByID,
			&validUntil,
			&dispensedAt,
			&dispensedBy,
			&rx.Status,
			&instructions,
			&diagICD,
			&rx.IsChronicMed,
			&rx.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan prescription: %w", err)
		}

		// Map nullable fields
		if medCode.Valid {
			rx.MedicationCode = medCode.String
		}
		if atc.Valid {
			rx.ATCCode = atc.String
		}
		if dosageUnit.Valid {
			rx.DosageUnit = dosageUnit.String
		}
		if route.Valid {
			rx.Route = route.String
		}
		if duration.Valid {
			rx.Duration = duration.String
		}
		if quantity.Valid {
			rx.Quantity = int(quantity.Int32)
		}
		if refills.Valid {
			rx.Refills = int(refills.Int32)
		}
		if prescribedByID.Valid {
			rx.PrescribedByID = prescribedByID.String
		}
		if validUntil.Valid {
			rx.ValidUntil = &validUntil.Time
		}
		if dispensedAt.Valid {
			rx.DispensedAt = &dispensedAt.Time
		}
		if dispensedBy.Valid {
			rx.DispensedBy = dispensedBy.String
		}
		if instructions.Valid {
			rx.Instructions = instructions.String
		}
		if diagICD.Valid {
			rx.DiagnosisICD10 = diagICD.String
		}

		rx.SourceSystem = a.SourceSystem()
		rx.SourceInstitution = a.SourceInstitution()

		prescriptions = append(prescriptions, rx)
	}

	return prescriptions, nil
}

// FetchDiagnoses retrieves diagnoses for a patient
func (a *Adapter) FetchDiagnoses(ctx context.Context, jmbg string, from, to time.Time) ([]health.Diagnosis, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("adapter not connected")
	}

	query := fmt.Sprintf(`
		SELECT
			d.DiagnosisID,
			d.ICD10Code,
			d.Description,
			d.Type,
			d.DiagnosedAt,
			d.DiagnosedBy,
			d.ResolvedAt,
			d.Notes
		FROM %s d
		INNER JOIN %s p ON d.PatientID = p.PatientID
		WHERE p.JMBG = @jmbg
		  AND d.DiagnosedAt >= @from
		  AND d.DiagnosedAt <= @to
		ORDER BY d.DiagnosedAt DESC
	`, a.config.DiagnosisTable, a.config.PatientTable)

	rows, err := a.db.QueryContext(ctx, query,
		sql.Named("jmbg", jmbg),
		sql.Named("from", from),
		sql.Named("to", to),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query diagnoses: %w", err)
	}
	defer rows.Close()

	var diagnoses []health.Diagnosis
	for rows.Next() {
		var d health.Diagnosis
		var resolvedAt sql.NullTime
		var diagBy, notes sql.NullString

		err := rows.Scan(
			&d.ID,
			&d.ICD10Code,
			&d.Description,
			&d.Type,
			&d.DiagnosedAt,
			&diagBy,
			&resolvedAt,
			&notes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan diagnosis: %w", err)
		}

		if diagBy.Valid {
			d.DiagnosedBy = diagBy.String
		}
		if resolvedAt.Valid {
			d.ResolvedAt = &resolvedAt.Time
		}
		if notes.Valid {
			d.Notes = notes.String
		}

		diagnoses = append(diagnoses, d)
	}

	return diagnoses, nil
}

// SubscribeAdmissions registers a handler for admission events
func (a *Adapter) SubscribeAdmissions(ctx context.Context, handler health.AdmissionHandler) error {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-a.admissionChan:
				if !ok {
					return
				}
				handler(event)
			}
		}
	}()
	return nil
}

// SubscribeDischarges registers a handler for discharge events
func (a *Adapter) SubscribeDischarges(ctx context.Context, handler health.DischargeHandler) error {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-a.dischargeChan:
				if !ok {
					return
				}
				handler(event)
			}
		}
	}()
	return nil
}

// SubscribeEmergencies registers a handler for emergency events
func (a *Adapter) SubscribeEmergencies(ctx context.Context, handler health.EmergencyHandler) error {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-a.emergencyChan:
				if !ok {
					return
				}
				handler(event)
			}
		}
	}()
	return nil
}

// pollLoop polls for new admissions and discharges
func (a *Adapter) pollLoop(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(a.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			lastPoll := a.lastPoll
			a.lastPoll = time.Now()
			a.mu.Unlock()

			// Poll for new admissions
			if err := a.pollAdmissions(ctx, lastPoll); err != nil {
				// Log error but continue
				fmt.Printf("Error polling admissions: %v\n", err)
			}

			// Poll for new discharges
			if err := a.pollDischarges(ctx, lastPoll); err != nil {
				fmt.Printf("Error polling discharges: %v\n", err)
			}
		}
	}
}

// pollAdmissions checks for new admissions since lastPoll
func (a *Adapter) pollAdmissions(ctx context.Context, since time.Time) error {
	query := fmt.Sprintf(`
		SELECT
			h.HospitalizationID,
			h.AdmissionDate,
			p.JMBG,
			p.FirstName + ' ' + p.LastName as PatientName,
			h.Department,
			h.AdmissionType,
			h.PrimaryDiagnosisICD
		FROM %s h
		INNER JOIN %s p ON h.PatientID = p.PatientID
		WHERE h.AdmissionDate > @since
		ORDER BY h.AdmissionDate ASC
	`, a.config.HospitalizationTable, a.config.PatientTable)

	rows, err := a.db.QueryContext(ctx, query, sql.Named("since", since))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var event health.AdmissionEvent
		var diagICD sql.NullString

		err := rows.Scan(
			&event.EventID,
			&event.Timestamp,
			&event.PatientJMBG,
			&event.PatientName,
			&event.Department,
			&event.AdmissionType,
			&diagICD,
		)
		if err != nil {
			continue
		}

		if diagICD.Valid {
			event.DiagnosisICD = diagICD.String
		}

		event.SourceSystem = a.SourceSystem()
		event.SourceInst = a.SourceInstitution()

		select {
		case a.admissionChan <- event:
		default:
			// Channel full, skip event
		}
	}

	return nil
}

// pollDischarges checks for new discharges since lastPoll
func (a *Adapter) pollDischarges(ctx context.Context, since time.Time) error {
	query := fmt.Sprintf(`
		SELECT
			h.HospitalizationID,
			h.DischargeDate,
			p.JMBG,
			p.FirstName + ' ' + p.LastName as PatientName,
			h.Department,
			h.DischargeType,
			h.AdmissionDate,
			h.PrimaryDiagnosisICD
		FROM %s h
		INNER JOIN %s p ON h.PatientID = p.PatientID
		WHERE h.DischargeDate > @since
		  AND h.DischargeDate IS NOT NULL
		ORDER BY h.DischargeDate ASC
	`, a.config.HospitalizationTable, a.config.PatientTable)

	rows, err := a.db.QueryContext(ctx, query, sql.Named("since", since))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var event health.DischargeEvent
		var diagICD sql.NullString

		err := rows.Scan(
			&event.EventID,
			&event.Timestamp,
			&event.PatientJMBG,
			&event.PatientName,
			&event.Department,
			&event.DischargeType,
			&event.AdmissionDate,
			&diagICD,
		)
		if err != nil {
			continue
		}

		event.DischargeDate = event.Timestamp
		if diagICD.Valid {
			event.DiagnosisICD = diagICD.String
		}

		event.SourceSystem = a.SourceSystem()
		event.SourceInst = a.SourceInstitution()

		select {
		case a.dischargeChan <- event:
		default:
			// Channel full, skip event
		}
	}

	return nil
}

// mapGender maps Heliant gender code to health.Gender
func mapGender(code string) health.Gender {
	switch code {
	case "M", "m", "1":
		return health.GenderMale
	case "F", "f", "Z", "z", "2":
		return health.GenderFemale
	case "O", "o", "3":
		return health.GenderOther
	default:
		return health.GenderUnknown
	}
}

// Verify interface implementation
var _ health.Adapter = (*Adapter)(nil)
