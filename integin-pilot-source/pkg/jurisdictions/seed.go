package jurisdictions

// TopJurisdictions returns the seed data for the 8 priority sovereign jurisdictions
// covering the 7 compliance pillars. These profiles are validated at Registry construction time.
func TopJurisdictions() []CountryProfile {
	return []CountryProfile{
		ksa(), uae(), usa(), uk(), germany(), singapore(), australia(), norway(),
	}
}

func ksa() CountryProfile {
	return CountryProfile{
		ISO2: "SA", ISO3: "SAU", NumericCode: "682",
		CommonName: "Saudi Arabia", OfficialName: "Kingdom of Saudi Arabia",
		PrimaryLanguage: "ar", SecondaryLanguage: "en",
		CurrencyCode: "SAR", CurrencySymbol: "﷼",
		Region: "Asia", SubRegion: "Western Asia",
		Lifecycle: LifecycleActive,
		TaxAuthority: TaxAuthority{
			Name: "ZATCA", TaxIDLabel: "VAT Number",
			TaxIDRegex: `^3\d{14}$`, CommercialReg: "Commercial Registration (CR)",
		},
		SafetyRegulators: []SafetyRegulator{
			{Name: "Saudi Council of Engineers (SCE)", Website: "https://www.sce.gov.sa"},
			{Name: "Saudi Standards, Metrology and Quality Organization (SASO)", Website: "https://www.saso.gov.sa"},
		},
		SubDivisions: []SubDivisionProfile{
			{Code: "RD", Name: "Riyadh", Notes: "Capital region"},
			{Code: "MC", Name: "Makkah"},
			{Code: "ED", Name: "Eastern Province", Notes: "Major oil & gas region"},
			{Code: "AS", Name: "Asir"},
		},
		ActivePillars: []PillarType{
			PillarSafetyRegulation, PillarTaxFiscal, PillarAccreditation,
			PillarCertificateMarking, PillarCurrencyTrade, PillarEnvironmentalESG,
			PillarLaborWorkforce,
		},
		Notes: "ZATCA e-invoicing mandate; SASO Conformity Assessment; SCE inspector licensing required.",
	}
}

func uae() CountryProfile {
	return CountryProfile{
		ISO2: "AE", ISO3: "ARE", NumericCode: "784",
		CommonName: "United Arab Emirates", OfficialName: "United Arab Emirates",
		PrimaryLanguage: "ar", SecondaryLanguage: "en",
		CurrencyCode: "AED", CurrencySymbol: "د.إ",
		Region: "Asia", SubRegion: "Western Asia",
		Lifecycle: LifecycleActive,
		TaxAuthority: TaxAuthority{
			Name: "Federal Tax Authority (FTA)", TaxIDLabel: "TRN (Tax Registration Number)",
			TaxIDRegex: `^1\d{13}$`, CommercialReg: "Trade License",
		},
		SafetyRegulators: []SafetyRegulator{
			{Name: "Abu Dhabi Department of Economic Development (ADDED)"},
			{Name: "Dubai Municipality"},
			{Name: "Emirates Authority for Standardization and Metrology (ESMA)"},
		},
		SubDivisions: []SubDivisionProfile{
			{Code: "AUH", Name: "Abu Dhabi"},
			{Code: "DXB", Name: "Dubai"},
			{Code: "SHJ", Name: "Sharjah"},
			{Code: "AJM", Name: "Ajman"},
		},
		ActivePillars: []PillarType{
			PillarSafetyRegulation, PillarTaxFiscal, PillarAccreditation,
			PillarCertificateMarking, PillarCurrencyTrade, PillarEnvironmentalESG,
			PillarLaborWorkforce,
		},
		Notes: "ADNOC/ADNOC Group inspection standards; 5% VAT; free zone特殊 regimes.",
	}
}

func usa() CountryProfile {
	return CountryProfile{
		ISO2: "US", ISO3: "USA", NumericCode: "840",
		CommonName: "United States", OfficialName: "United States of America",
		PrimaryLanguage: "en", SecondaryLanguage: "es",
		CurrencyCode: "USD", CurrencySymbol: "$",
		Region: "Americas", SubRegion: "Northern America",
		Lifecycle: LifecycleActive,
		TaxAuthority: TaxAuthority{
			Name: "Internal Revenue Service (IRS)", TaxIDLabel: "EIN / TIN",
			TaxIDRegex: `^\d{2}-\d{7}$`, CommercialReg: "State Registration",
		},
		SafetyRegulators: []SafetyRegulator{
			{Name: "Occupational Safety and Health Administration (OSHA)", Website: "https://www.osha.gov"},
			{Name: "American Society of Mechanical Engineers (ASME)", Website: "https://www.asme.org"},
			{Name: "National Institute for Standards and Technology (NIST)"},
		},
		SubDivisions: []SubDivisionProfile{
			{Code: "TX", Name: "Texas", Notes: "Major oil & gas"},
			{Code: "CA", Name: "California"},
			{Code: "LA", Name: "Louisiana", Notes: "Petrochemical corridor"},
		},
		ActivePillars: []PillarType{
			PillarSafetyRegulation, PillarAccreditation, PillarCertificateMarking,
			PillarCurrencyTrade, PillarEnvironmentalESG, PillarLaborWorkforce,
		},
		Notes: "OSHA 29 CFR 1926; ASME B30 series; state-level OSHA plans (CA, TX, LA).",
	}
}

func uk() CountryProfile {
	return CountryProfile{
		ISO2: "GB", ISO3: "GBR", NumericCode: "826",
		CommonName: "United Kingdom", OfficialName: "United Kingdom of Great Britain and Northern Ireland",
		PrimaryLanguage: "en",
		CurrencyCode:    "GBP", CurrencySymbol: "£",
		Region: "Europe", SubRegion: "Northern Europe",
		Lifecycle: LifecycleActive,
		TaxAuthority: TaxAuthority{
			Name: "HM Revenue & Customs (HMRC)", TaxIDLabel: "UTR / VAT Number",
			TaxIDRegex: `^\d{9}$|GB\d{9}$|GB\d{12}$`, CommercialReg: "Companies House",
		},
		SafetyRegulators: []SafetyRegulator{
			{Name: "Health and Safety Executive (HSE)", Website: "https://www.hse.gov.uk"},
			{Name: "British Standards Institution (BSI)", Website: "https://www.bsigroup.com"},
		},
		SubDivisions: []SubDivisionProfile{
			{Code: "ENG", Name: "England"},
			{Code: "SCT", Name: "Scotland"},
			{Code: "WLS", Name: "Wales"},
			{Code: "NIR", Name: "Northern Ireland"},
		},
		ActivePillars: []PillarType{
			PillarSafetyRegulation, PillarTaxFiscal, PillarAccreditation,
			PillarCertificateMarking, PillarCurrencyTrade, PillarEnvironmentalESG,
			PillarLaborWorkforce,
		},
		Notes: "LOLER 1998; PUWER 1998; BSI EN standards; UKCA marking post-Brexit.",
	}
}

func germany() CountryProfile {
	return CountryProfile{
		ISO2: "DE", ISO3: "DEU", NumericCode: "276",
		CommonName: "Germany", OfficialName: "Federal Republic of Germany",
		PrimaryLanguage: "de", SecondaryLanguage: "en",
		CurrencyCode: "EUR", CurrencySymbol: "€",
		Region: "Europe", SubRegion: "Western Europe",
		Lifecycle: LifecycleActive,
		TaxAuthority: TaxAuthority{
			Name: "Bundeszentralamt für Steuern (BZSt)", TaxIDLabel: "USt-IdNr.",
			TaxIDRegex: `^DE\d{9}$`, CommercialReg: "Handelsregister",
		},
		SafetyRegulators: []SafetyRegulator{
			{Name: "Berufsgenossenschaft (BG) - Employers' Liability Insurance"},
			{Name: "Deutsches Institut für Normung (DIN)", Website: "https://www.din.de"},
			{Name: "TÜV SÜD / TÜV Rheinland"},
		},
		SubDivisions: []SubDivisionProfile{
			{Code: "NW", Name: "North Rhine-Westphalia", Notes: "Industrial heartland"},
			{Code: "BY", Name: "Bavaria"},
			{Code: "HE", Name: "Hesse"},
		},
		ActivePillars: []PillarType{
			PillarSafetyRegulation, PillarTaxFiscal, PillarAccreditation,
			PillarCertificateMarking, PillarCurrencyTrade, PillarEnvironmentalESG,
			PillarLaborWorkforce,
		},
		Notes: "EU Machinery Directive 2006/42/EC; DIN EN standards; BG mandatory insurance; CE marking.",
	}
}

func singapore() CountryProfile {
	return CountryProfile{
		ISO2: "SG", ISO3: "SGP", NumericCode: "702",
		CommonName: "Singapore", OfficialName: "Republic of Singapore",
		PrimaryLanguage: "en", SecondaryLanguage: "zh",
		CurrencyCode: "SGD", CurrencySymbol: "S$",
		Region: "Asia", SubRegion: "South-Eastern Asia",
		Lifecycle: LifecycleActive,
		TaxAuthority: TaxAuthority{
			Name: "Inland Revenue Authority of Singapore (IRAS)", TaxIDLabel: "UEN / GST Reg No.",
			TaxIDRegex: `^\d{8}[A-Z]$`, CommercialReg: "ACRA Business Profile",
		},
		SafetyRegulators: []SafetyRegulator{
			{Name: "Ministry of Manpower (MOM)", Website: "https://www.mom.gov.sg"},
			{Name: "Building and Construction Authority (BCA)"},
			{Name: "Enterprise Singapore (EnterpriseSG)"},
		},
		ActivePillars: []PillarType{
			PillarSafetyRegulation, PillarTaxFiscal, PillarAccreditation,
			PillarCertificateMarking, PillarCurrencyTrade, PillarEnvironmentalESG,
			PillarLaborWorkforce,
		},
		Notes: "WSH Act; 9% GST; MOM Factories and Machinery Act; EnterpriseSG standards.",
	}
}

func australia() CountryProfile {
	return CountryProfile{
		ISO2: "AU", ISO3: "AUS", NumericCode: "036",
		CommonName: "Australia", OfficialName: "Commonwealth of Australia",
		PrimaryLanguage: "en",
		CurrencyCode:    "AUD", CurrencySymbol: "A$",
		Region: "Oceania", SubRegion: "Australia and New Zealand",
		Lifecycle: LifecycleActive,
		TaxAuthority: TaxAuthority{
			Name: "Australian Taxation Office (ATO)", TaxIDLabel: "ABN / TFN",
			TaxIDRegex: `^\d{11}$`, CommercialReg: "ASIC Registration",
		},
		SafetyRegulators: []SafetyRegulator{
			{Name: "Safe Work Australia", Website: "https://www.safeworkaustralia.gov.au"},
			{Name: "Standards Australia", Website: "https://www.standards.org.au"},
			{Name: "WorkSafe (state-level)"},
		},
		SubDivisions: []SubDivisionProfile{
			{Code: "NSW", Name: "New South Wales"},
			{Code: "VIC", Name: "Victoria"},
			{Code: "QLD", Name: "Queensland"},
			{Code: "WA", Name: "Western Australia", Notes: "Mining & resources"},
		},
		ActivePillars: []PillarType{
			PillarSafetyRegulation, PillarTaxFiscal, PillarAccreditation,
			PillarCertificateMarking, PillarCurrencyTrade, PillarEnvironmentalESG,
			PillarLaborWorkforce,
		},
		Notes: "WHS Act 2011; harmonized model WHS laws; AS/NZS standards; 10% GST.",
	}
}

func norway() CountryProfile {
	return CountryProfile{
		ISO2: "NO", ISO3: "NOR", NumericCode: "578",
		CommonName: "Norway", OfficialName: "Kingdom of Norway",
		PrimaryLanguage: "no", SecondaryLanguage: "en",
		CurrencyCode: "NOK", CurrencySymbol: "kr",
		Region: "Europe", SubRegion: "Northern Europe",
		Lifecycle: LifecycleActive,
		TaxAuthority: TaxAuthority{
			Name: "Skatteetaten (Norwegian Tax Administration)", TaxIDLabel: "Org.nr / MVA",
			TaxIDRegex: `^\d{9}$`, CommercialReg: "Brønnøysund Register",
		},
		SafetyRegulators: []SafetyRegulator{
			{Name: "Petroleum Safety Authority Norway (PSA)", Website: "https://www.psa.no"},
			{Name: "Norwegian Directorate of Labour Inspection (Arbeidstilsynet)"},
			{Name: "DNV (Det Norske Veritas)", Website: "https://www.dnv.com"},
		},
		ActivePillars: []PillarType{
			PillarSafetyRegulation, PillarTaxFiscal, PillarAccreditation,
			PillarCertificateMarking, PillarCurrencyTrade, PillarEnvironmentalESG,
			PillarLaborWorkforce,
		},
		Notes: "PSA offshore petroleum; DNV type-approval; NOK 25% VAT; EU/EEA harmonized standards.",
	}
}
