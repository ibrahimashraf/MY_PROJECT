package notifications

import "testing"

func TestMandatoryCategories(t *testing.T) {
	if !CategorySecurityEvent.Mandatory() || !CategorySystemAlert.Mandatory() {
		t.Fatal("security and system categories must be mandatory")
	}
	if CategoryInspectionAssigned.Mandatory() {
		t.Fatal("inspection assignment should respect preferences")
	}
}

func TestDeliveryStatesAndChannels(t *testing.T) {
	if ChannelInApp == ChannelEmail {
		t.Fatal("channels must be distinct")
	}
	if DeliveryQueued == DeliveryGivenUp {
		t.Fatal("delivery states must be distinct")
	}
}
