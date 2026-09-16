package cfihos

import (
	"encoding/csv"
	"io"
)

// Tag represents a CFIHOS Tag object (ISO 18101).
type Tag struct {
	ID             string
	Facility       string
	System         string
	EquipmentClass string
	Status         string
	Properties     map[string]string
}

// Equipment represents a physical piece of equipment linked to a CFIHOS Tag.
type Equipment struct {
	ID           string
	TagID        string
	SerialNumber string
	Manufacturer string
	Model        string
	Status       string
}

// WriteCSV serializes CFIHOS data into the standard handover dataset CSV format.
func WriteCSV(w io.Writer, tags []Tag, equipment []Equipment) error {
	cw := csv.NewWriter(w)
	// Write Header
	if err := cw.Write([]string{"Entity", "ID", "TagID", "Facility", "System", "Class", "SerialNumber", "Manufacturer", "Model", "Status"}); err != nil {
		return err
	}
	
	// Write Tags
	for _, t := range tags {
		if err := cw.Write([]string{"Tag", t.ID, "", t.Facility, t.System, t.EquipmentClass, "", "", "", t.Status}); err != nil {
			return err
		}
	}
	
	// Write Equipment
	for _, e := range equipment {
		if err := cw.Write([]string{"Equipment", e.ID, e.TagID, "", "", "", e.SerialNumber, e.Manufacturer, e.Model, e.Status}); err != nil {
			return err
		}
	}
	
	cw.Flush()
	return cw.Error()
}
