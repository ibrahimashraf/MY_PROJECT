package qrverification

import (
	"errors"

	"sync"
)

type Certificate struct {
	Token           string
	Environment     string
	Status          string
	Number          string
	Revision        int
	AssetType       string
	Manufacturer    string
	Model           string
	SerialNumber    string
	ManufactureDate string
	SWL             string
	CCUStamp        string
	Issuer          string
	InspectionDate  string
	ExpiryDate      string
	Findings        []string
	Photos          []string
	ClientName      string
	Site            string
	Inspector       string
	InternalID      string
}
type PublicView struct {
	Status            string
	CertificateNumber string
	Revision          int
	AssetType         string
	Manufacturer      string
	Model             string
	SerialNumber      string
	ManufactureDate   string
	SWL               string
	CCUStamp          string
	Issuer            string
	InspectionDate    string
	ExpiryDate        string
	TestCertificate   bool
}
type TechnicalView struct {
	PublicView
	Findings   []string
	Photos     []string
	ClientName string
	Site       string
	Inspector  string
	InternalID string
}
type Result struct {
	StatusCode int
	Public     *PublicView
	Technical  *TechnicalView
	Error      error
}
type Verifier struct {
	mu           sync.Mutex
	certificates map[string]Certificate
	requests     map[string]int
	limit        int
}

func New(limit int) *Verifier {
	if limit <= 0 {
		limit = 60
	}
	return &Verifier{certificates: make(map[string]Certificate), requests: make(map[string]int), limit: limit}
}
func (v *Verifier) Register(certificate Certificate) error {
	if certificate.Token == "" || certificate.Number == "" || certificate.Status == "" {
		return errors.New("certificate token, number, and status are required")
	}
	v.certificates[certificate.Token] = certificate
	return nil
}
func (v *Verifier) PublicVerify(token, requestKey string) Result {
	if !v.allow(requestKey) {
		return Result{StatusCode: 429, Error: errors.New("rate limit exceeded")}
	}
	certificate, ok := v.certificates[token]
	if !ok {
		return Result{StatusCode: 404, Error: errors.New("certificate not found")}
	}
	view := public(certificate)
	return Result{StatusCode: 200, Public: &view}
}
func (v *Verifier) TechnicalVerify(token, userID string, authorize func(string, Certificate) bool) Result {
	certificate, ok := v.certificates[token]
	if !ok {
		return Result{StatusCode: 404, Error: errors.New("certificate not found")}
	}
	if authorize == nil || !authorize(userID, certificate) {
		return Result{StatusCode: 403, Error: errors.New("technical verification unauthorized")}
	}
	view := TechnicalView{PublicView: public(certificate), Findings: append([]string(nil), certificate.Findings...), Photos: append([]string(nil), certificate.Photos...), ClientName: certificate.ClientName, Site: certificate.Site, Inspector: certificate.Inspector, InternalID: certificate.InternalID}
	return Result{StatusCode: 200, Technical: &view}
}
func (v *Verifier) allow(requestKey string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.requests[requestKey]++
	return v.requests[requestKey] <= v.limit
}
func public(c Certificate) PublicView {
	return PublicView{Status: c.Status, CertificateNumber: c.Number, Revision: c.Revision, AssetType: c.AssetType, Manufacturer: c.Manufacturer, Model: c.Model, SerialNumber: c.SerialNumber, ManufactureDate: c.ManufactureDate, SWL: c.SWL, CCUStamp: c.CCUStamp, Issuer: c.Issuer, InspectionDate: c.InspectionDate, ExpiryDate: c.ExpiryDate, TestCertificate: c.Environment == "TESTING"}
}
