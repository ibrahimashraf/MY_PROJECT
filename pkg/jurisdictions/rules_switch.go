package jurisdictions

import (
	"strings"

	"integin/pkg/rulesengine"
)

// defaultDesignBasis is the fail-closed default applied to jurisdictions
// outside the ratified seed map below. ASD (allowable stress) is the least
// assumption-prone starting point; statutory ratification is a Phase 5 legal
// nexus deliverable.
const defaultDesignBasis = rulesengine.DesignBasisASD

// designBasisByISO2 seeds the deterministic ASD vs LRFD switch. US-code
// practice (US, SA, AE) verifies in ASD; Eurocode/limit-state practice (GB,
// DE, SG, AU, NO) verifies in LRFD. This is a default seed for engine wiring
// only and must be ratified against each jurisdiction's statutory design
// code before production use.
var designBasisByISO2 = map[string]rulesengine.DesignBasis{
	"US": rulesengine.DesignBasisASD,
	"SA": rulesengine.DesignBasisASD,
	"AE": rulesengine.DesignBasisASD,
	"GB": rulesengine.DesignBasisLRFD,
	"DE": rulesengine.DesignBasisLRFD,
	"SG": rulesengine.DesignBasisLRFD,
	"AU": rulesengine.DesignBasisLRFD,
	"NO": rulesengine.DesignBasisLRFD,
}

// DesignBasisForISO2 returns the design-computation philosophy the rules
// engine should apply for an asset jurisdiction. Unknown or empty
// jurisdictions resolve to the conservative default. The read-only map is
// safe for concurrent access.
func DesignBasisForISO2(iso2 string) rulesengine.DesignBasis {
	if b, ok := designBasisByISO2[strings.ToUpper(strings.TrimSpace(iso2))]; ok {
		return b
	}
	return defaultDesignBasis
}
