package socialcard

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/serbia-gov/platform/internal/adapters/social"
)

// Client implements social.Adapter for Socijalna Karta API via Servisna magistrala
type Client struct {
	baseURL    string
	httpClient *http.Client
	config     Config
}

// Config holds configuration for Socijalna Karta client
type Config struct {
	// API endpoint
	BaseURL string `json:"base_url"`

	// mTLS certificates
	CertFile   string `json:"cert_file"`
	KeyFile    string `json:"key_file"`
	CACertFile string `json:"ca_cert_file"`

	// Timeouts
	Timeout         time.Duration `json:"timeout"`
	RetryAttempts   int           `json:"retry_attempts"`
	RetryDelay      time.Duration `json:"retry_delay"`

	// Rate limiting
	MaxRequestsPerSecond int `json:"max_requests_per_second"`
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		BaseURL:              "https://servisna-magistrala.gov.rs/api/v1/socijalna-karta",
		Timeout:              30 * time.Second,
		RetryAttempts:        3,
		RetryDelay:           1 * time.Second,
		MaxRequestsPerSecond: 10,
	}
}

// New creates a new Socijalna Karta client
func New(cfg Config) (*Client, error) {
	// Load client certificate
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA certificate
	caCert, err := os.ReadFile(cfg.CACertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// Configure TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: cfg.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return &Client{
		baseURL:    cfg.BaseURL,
		httpClient: client,
		config:     cfg,
	}, nil
}

// FetchBeneficiaryStatus retrieves beneficiary status from Socijalna Karta
func (c *Client) FetchBeneficiaryStatus(ctx context.Context, jmbg string) (*social.BeneficiaryStatus, error) {
	url := fmt.Sprintf("%s/beneficiary/%s/status", c.baseURL, jmbg)

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch beneficiary status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("beneficiary not found: %s", jmbg)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiResp beneficiaryStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return mapBeneficiaryStatus(&apiResp), nil
}

// FetchFamilyComposition retrieves family composition from Socijalna Karta
func (c *Client) FetchFamilyComposition(ctx context.Context, jmbg string) (*social.FamilyUnit, error) {
	url := fmt.Sprintf("%s/beneficiary/%s/family", c.baseURL, jmbg)

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch family composition: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("family not found for: %s", jmbg)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiResp familyCompositionResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return mapFamilyUnit(&apiResp), nil
}

// FetchPropertyData retrieves property data from Socijalna Karta
func (c *Client) FetchPropertyData(ctx context.Context, jmbg string) (*social.PropertyData, error) {
	url := fmt.Sprintf("%s/beneficiary/%s/property", c.baseURL, jmbg)

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch property data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // No property data is valid
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiResp propertyDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return mapPropertyData(&apiResp), nil
}

// FetchIncomeData retrieves income data from Socijalna Karta
func (c *Client) FetchIncomeData(ctx context.Context, jmbg string) (*social.IncomeData, error) {
	url := fmt.Sprintf("%s/beneficiary/%s/income", c.baseURL, jmbg)

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch income data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // No income data is valid
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiResp incomeDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return mapIncomeData(&apiResp), nil
}

// doRequest performs an HTTP request with retry logic
func (c *Client) doRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt < c.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.config.RetryDelay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Don't retry on client errors (4xx)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return resp, nil
		}

		// Retry on server errors (5xx)
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// API response types (mapped from Socijalna Karta API)

type beneficiaryStatusResponse struct {
	JMBG                     string  `json:"jmbg"`
	NovcanaSocijalnaPomoc    bool    `json:"novcana_socijalna_pomoc"`
	IznosNSP                 float64 `json:"iznos_nsp,omitempty"`
	DatumPocetka             string  `json:"datum_pocetka,omitempty"`
	DatumZavrsetka           string  `json:"datum_zavrsetka,omitempty"`
	DecijaDodatak            bool    `json:"decija_dodatak"`
	IznosDD                  float64 `json:"iznos_dd,omitempty"`
	DatumPocetkaDD           string  `json:"datum_pocetka_dd,omitempty"`
	Invalidnina              bool    `json:"invalidnina"`
	IznosInvalidnine         float64 `json:"iznos_invalidnine,omitempty"`
	NegaStarih               bool    `json:"nega_starih"`
	TipNegeStarih            string  `json:"tip_nege_starih,omitempty"`
	Zaposlen                 bool    `json:"zaposlen"`
	StatusZaposlenja         string  `json:"status_zaposlenja,omitempty"`
	Penzioner                bool    `json:"penzioner"`
	ImaZdravstvenoOsiguranje bool    `json:"ima_zdravstveno_osiguranje"`
	TipOsiguranja            string  `json:"tip_osiguranja,omitempty"`
	RizicnaGrupa             bool    `json:"rizicna_grupa"`
	NivoRizika               string  `json:"nivo_rizika,omitempty"`
	FaktoriRizika            []string `json:"faktori_rizika,omitempty"`
	DatumAzuriranja          string  `json:"datum_azuriranja"`
}

type familyCompositionResponse struct {
	SifraDomacinstva string `json:"sifra_domacinstva"`
	NosilacJMBG      string `json:"nosilac_jmbg"`
	Adresa           string `json:"adresa"`
	Opstina          string `json:"opstina"`
	Clanovi          []struct {
		JMBG          string  `json:"jmbg"`
		Ime           string  `json:"ime"`
		Prezime       string  `json:"prezime"`
		DatumRodjenja string  `json:"datum_rodjenja"`
		Srodstvo      string  `json:"srodstvo"`
		Pol           string  `json:"pol"`
		Zaposlen      bool    `json:"zaposlen"`
		Student       bool    `json:"student"`
		Invaliditet   bool    `json:"invaliditet"`
		TipInval      string  `json:"tip_invaliditeta,omitempty"`
		Prihod        float64 `json:"prihod,omitempty"`
	} `json:"clanovi"`
	UkupanPrihod    float64 `json:"ukupan_prihod,omitempty"`
	PrihodPoGlavi   float64 `json:"prihod_po_glavi,omitempty"`
	TipStanovanja   string  `json:"tip_stanovanja,omitempty"`
	StatusStanovanja string `json:"status_stanovanja,omitempty"`
	DatumAzuriranja string  `json:"datum_azuriranja"`
}

type propertyDataResponse struct {
	JMBG           string `json:"jmbg"`
	PosedNekret    bool   `json:"poseduje_nekretnine"`
	Nekretnine     []struct {
		Tip           string  `json:"tip"`
		Lokacija      string  `json:"lokacija"`
		PovrsinaM2    float64 `json:"povrsina_m2,omitempty"`
		UdeoProcenata float64 `json:"udeo_procenata"`
		Vrednost      float64 `json:"vrednost,omitempty"`
		Opterecena    bool    `json:"opterecena"`
	} `json:"nekretnine,omitempty"`
	PosedVozilo    bool   `json:"poseduje_vozilo"`
	Vozila         []struct {
		Tip        string  `json:"tip"`
		Marka      string  `json:"marka,omitempty"`
		Godiste    int     `json:"godiste,omitempty"`
		UdeoProc   float64 `json:"udeo_procenata"`
	} `json:"vozila,omitempty"`
	ImaUstede      bool   `json:"ima_ustede"`
	RasponUstede   string `json:"raspon_ustede,omitempty"`
	DatumAzuriranja string `json:"datum_azuriranja"`
}

type incomeDataResponse struct {
	JMBG           string `json:"jmbg"`
	IzvoriPrihoda  []struct {
		Tip        string  `json:"tip"`
		Iznos      float64 `json:"iznos"`
		Ucestalost string  `json:"ucestalost"`
		Poslodavac string  `json:"poslodavac,omitempty"`
		OdDatuma   string  `json:"od_datuma,omitempty"`
	} `json:"izvori_prihoda"`
	UkupnoMesecno   float64 `json:"ukupno_mesecno"`
	UkupnoGodisnje  float64 `json:"ukupno_godisnje"`
	DatumAzuriranja string  `json:"datum_azuriranja"`
}

// Mapping functions

func mapBeneficiaryStatus(resp *beneficiaryStatusResponse) *social.BeneficiaryStatus {
	status := &social.BeneficiaryStatus{
		JMBG:                      resp.JMBG,
		ReceivesCashAssistance:    resp.NovcanaSocijalnaPomoc,
		CashAssistanceAmount:      resp.IznosNSP,
		ReceivesChildAllowance:    resp.DecijaDodatak,
		ChildAllowanceAmount:      resp.IznosDD,
		ReceivesDisabilityBenefit: resp.Invalidnina,
		DisabilityBenefitAmount:   resp.IznosInvalidnine,
		ReceivesElderlyCare:       resp.NegaStarih,
		ElderlyCareType:           resp.TipNegeStarih,
		IsEmployed:                resp.Zaposlen,
		EmploymentStatus:          resp.StatusZaposlenja,
		IsRetired:                 resp.Penzioner,
		HasHealthInsurance:        resp.ImaZdravstvenoOsiguranje,
		HealthInsuranceType:       resp.TipOsiguranja,
		IsAtRisk:                  resp.RizicnaGrupa,
		RiskLevel:                 resp.NivoRizika,
		RiskFactors:               resp.FaktoriRizika,
		DataSource:                "socijalna_karta",
	}

	// Parse dates
	if resp.DatumAzuriranja != "" {
		if t, err := time.Parse("2006-01-02", resp.DatumAzuriranja); err == nil {
			status.LastUpdated = t
		}
	}
	if resp.DatumPocetka != "" {
		if t, err := time.Parse("2006-01-02", resp.DatumPocetka); err == nil {
			status.CashAssistanceSince = &t
		}
	}

	return status
}

func mapFamilyUnit(resp *familyCompositionResponse) *social.FamilyUnit {
	unit := &social.FamilyUnit{
		HouseholdID:     resp.SifraDomacinstva,
		HeadOfFamily:    resp.NosilacJMBG,
		Address:         resp.Adresa,
		Municipality:    resp.Opstina,
		TotalIncome:     resp.UkupanPrihod,
		IncomePerCapita: resp.PrihodPoGlavi,
		HousingType:     resp.TipStanovanja,
		HousingStatus:   resp.StatusStanovanja,
		Members:         make([]social.FamilyMember, 0, len(resp.Clanovi)),
	}

	for _, c := range resp.Clanovi {
		member := social.FamilyMember{
			JMBG:           c.JMBG,
			FirstName:      c.Ime,
			LastName:       c.Prezime,
			Relationship:   mapRelationship(c.Srodstvo),
			Gender:         c.Pol,
			IsEmployed:     c.Zaposlen,
			IsStudent:      c.Student,
			HasDisability:  c.Invaliditet,
			DisabilityType: c.TipInval,
			Income:         c.Prihod,
		}

		if c.DatumRodjenja != "" {
			if t, err := time.Parse("2006-01-02", c.DatumRodjenja); err == nil {
				member.DateOfBirth = t
			}
		}

		unit.Members = append(unit.Members, member)
	}

	if resp.DatumAzuriranja != "" {
		if t, err := time.Parse("2006-01-02", resp.DatumAzuriranja); err == nil {
			unit.LastUpdated = t
		}
	}

	return unit
}

func mapPropertyData(resp *propertyDataResponse) *social.PropertyData {
	data := &social.PropertyData{
		JMBG:           resp.JMBG,
		OwnsRealEstate: resp.PosedNekret,
		OwnsVehicle:    resp.PosedVozilo,
		HasSavings:     resp.ImaUstede,
		SavingsRange:   resp.RasponUstede,
		DataSource:     "socijalna_karta",
	}

	for _, n := range resp.Nekretnine {
		data.Properties = append(data.Properties, social.Property{
			Type:         mapPropertyType(n.Tip),
			Location:     n.Lokacija,
			SizeM2:       n.PovrsinaM2,
			OwnershipPct: n.UdeoProcenata,
			Value:        n.Vrednost,
			Encumbered:   n.Opterecena,
		})
	}

	for _, v := range resp.Vozila {
		data.Vehicles = append(data.Vehicles, social.Vehicle{
			Type:              mapVehicleType(v.Tip),
			Brand:             v.Marka,
			YearOfManufacture: v.Godiste,
			OwnershipPct:      v.UdeoProc,
		})
	}

	if resp.DatumAzuriranja != "" {
		if t, err := time.Parse("2006-01-02", resp.DatumAzuriranja); err == nil {
			data.LastUpdated = t
		}
	}

	return data
}

func mapIncomeData(resp *incomeDataResponse) *social.IncomeData {
	data := &social.IncomeData{
		JMBG:         resp.JMBG,
		TotalMonthly: resp.UkupnoMesecno,
		TotalYearly:  resp.UkupnoGodisnje,
		DataSource:   "socijalna_karta",
	}

	for _, i := range resp.IzvoriPrihoda {
		source := social.IncomeSource{
			Type:      mapIncomeType(i.Tip),
			Amount:    i.Iznos,
			Frequency: mapFrequency(i.Ucestalost),
			Employer:  i.Poslodavac,
		}

		if i.OdDatuma != "" {
			if t, err := time.Parse("2006-01-02", i.OdDatuma); err == nil {
				source.Since = &t
			}
		}

		data.Sources = append(data.Sources, source)
	}

	if resp.DatumAzuriranja != "" {
		if t, err := time.Parse("2006-01-02", resp.DatumAzuriranja); err == nil {
			data.LastUpdated = t
		}
	}

	return data
}

// Helper mapping functions

func mapRelationship(srodstvo string) string {
	switch srodstvo {
	case "nosilac":
		return "head"
	case "suprug", "supruga":
		return "spouse"
	case "dete", "sin", "cerka":
		return "child"
	case "roditelj", "otac", "majka":
		return "parent"
	default:
		return "other"
	}
}

func mapPropertyType(tip string) string {
	switch tip {
	case "stan":
		return "apartment"
	case "kuca":
		return "house"
	case "zemljiste":
		return "land"
	case "poslovni_prostor":
		return "commercial"
	default:
		return tip
	}
}

func mapVehicleType(tip string) string {
	switch tip {
	case "automobil":
		return "car"
	case "motor", "motocikl":
		return "motorcycle"
	case "kamion":
		return "truck"
	default:
		return tip
	}
}

func mapIncomeType(tip string) string {
	switch tip {
	case "zarada", "plata":
		return "salary"
	case "penzija":
		return "pension"
	case "socijalna_pomoc", "naknada":
		return "benefit"
	case "zakupnina":
		return "rental"
	default:
		return "other"
	}
}

func mapFrequency(ucestalost string) string {
	switch ucestalost {
	case "mesecno":
		return "monthly"
	case "godisnje":
		return "yearly"
	default:
		return ucestalost
	}
}
