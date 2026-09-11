package main

import "log"

// warnIfUnconfigured logs loud boot-time warnings when operator-critical
// subsystems are not configured. Pure function for testability.
func warnIfUnconfigured(tsaConfigured bool, tusEnabled bool, oidcConfigured bool) {
	if !tsaConfigured {
		log.Print("WARNING: TSA roots not provisioned — certificates issue WITHOUT trusted timestamps")
	}
	if !tusEnabled {
		log.Print("WARNING: /uploads (TUS) is disabled — set INTEGIN_TUS_SCRATCH_DIR to enable")
	} else if !oidcConfigured {
		log.Print("WARNING: TUS enabled without OIDC validator — authenticated upload disabled at runtime")
	}
}
