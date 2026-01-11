package types

// Address represents a physical address
type Address struct {
	Street     string  `json:"street"`
	City       string  `json:"city"`
	PostalCode string  `json:"postal_code"`
	Country    string  `json:"country"` // ISO 3166-1 alpha-2, default "RS"
	Lat        float64 `json:"lat,omitempty"`
	Lng        float64 `json:"lng,omitempty"`
}

// NewAddress creates a new address with Serbia as default country
func NewAddress(street, city, postalCode string) Address {
	return Address{
		Street:     street,
		City:       city,
		PostalCode: postalCode,
		Country:    "RS",
	}
}

// WithCoordinates adds geographic coordinates to the address
func (a Address) WithCoordinates(lat, lng float64) Address {
	a.Lat = lat
	a.Lng = lng
	return a
}

// ContactInfo represents contact information
type ContactInfo struct {
	Email  string `json:"email,omitempty"`
	Phone  string `json:"phone,omitempty"`
	Mobile string `json:"mobile,omitempty"`
}
